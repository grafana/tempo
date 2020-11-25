package app

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"github.com/cortexproject/cortex/pkg/querier/frontend"
	frontend2 "github.com/grafana/tempo/modules/frontend"
	"net/http"

	"github.com/cortexproject/cortex/pkg/ring"
	"github.com/cortexproject/cortex/pkg/ring/kv/memberlist"
	"github.com/cortexproject/cortex/pkg/util"
	"github.com/cortexproject/cortex/pkg/util/flagext"
	"github.com/cortexproject/cortex/pkg/util/grpc/healthcheck"
	"github.com/cortexproject/cortex/pkg/util/modules"
	"github.com/cortexproject/cortex/pkg/util/services"
	"github.com/go-kit/kit/log/level"

	"github.com/weaveworks/common/middleware"
	"github.com/weaveworks/common/server"
	"github.com/weaveworks/common/signals"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/grafana/tempo/modules/compactor"
	"github.com/grafana/tempo/modules/distributor"
	"github.com/grafana/tempo/modules/ingester"
	ingester_client "github.com/grafana/tempo/modules/ingester/client"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/modules/querier"
	"github.com/grafana/tempo/modules/storage"
	tempo_util "github.com/grafana/tempo/pkg/util"
)

const metricsNamespace = "tempo"

// Config is the root config for App.
type Config struct {
	Target      string `yaml:"target,omitempty"`
	AuthEnabled bool   `yaml:"auth_enabled,omitempty"`
	HTTPPrefix  string `yaml:"http_prefix"`

	Server         server.Config            `yaml:"server,omitempty"`
	Distributor    distributor.Config       `yaml:"distributor,omitempty"`
	IngesterClient ingester_client.Config   `yaml:"ingester_client,omitempty"`
	Querier        querier.Config           `yaml:"querier,omitempty"`
	Frontend       frontend2.FrontendConfig `yaml:"frontend,omitempty"`
	Compactor      compactor.Config         `yaml:"compactor,omitempty"`
	Ingester       ingester.Config          `yaml:"ingester,omitempty"`
	StorageConfig  storage.Config           `yaml:"storage,omitempty"`
	LimitsConfig   overrides.Limits         `yaml:"overrides,omitempty"`
	MemberlistKV   memberlist.KVConfig      `yaml:"memberlist,omitempty"`
}

// RegisterFlagsAndApplyDefaults registers flag.
func (c *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	c.Target = All
	// global settings
	f.StringVar(&c.Target, "target", All, "target module")
	f.BoolVar(&c.AuthEnabled, "auth.enabled", true, "Set to false to disable auth.")

	// Server settings
	flagext.DefaultValues(&c.Server)
	c.Server.LogLevel.RegisterFlags(f)
	f.IntVar(&c.Server.HTTPListenPort, "server.http-listen-port", 80, "HTTP server listen port.")
	f.IntVar(&c.Server.GRPCListenPort, "server.grpc-listen-port", 9095, "gRPC server listen port.")

	// Memberlist settings
	fs := flag.NewFlagSet("", flag.PanicOnError)
	c.MemberlistKV.RegisterFlags(fs, "")
	_ = fs.Parse([]string{})
	f.Var(&c.MemberlistKV.JoinMembers, "memberlist.host-port", "Host port to connect to memberlist cluster.")
	f.IntVar(&c.MemberlistKV.TCPTransport.BindPort, "memberlist.bind-port", 7946, "Port for memberlist to communicate on")

	// Everything else
	flagext.DefaultValues(&c.IngesterClient)
	flagext.DefaultValues(&c.LimitsConfig)

	c.Distributor.RegisterFlagsAndApplyDefaults(tempo_util.PrefixConfig(prefix, "distributor"), f)
	c.Ingester.RegisterFlagsAndApplyDefaults(tempo_util.PrefixConfig(prefix, "ingester"), f)
	c.Querier.RegisterFlagsAndApplyDefaults(tempo_util.PrefixConfig(prefix, "querier"), f)
	c.Frontend.ApplyDefaults()
	c.Compactor.RegisterFlagsAndApplyDefaults(tempo_util.PrefixConfig(prefix, "compactor"), f)
	c.StorageConfig.RegisterFlagsAndApplyDefaults(tempo_util.PrefixConfig(prefix, "storage"), f)

}

// CheckConfig checks if config values are suspect.
func (c *Config) CheckConfig() {
	if c.Ingester.CompleteBlockTimeout < c.StorageConfig.Trace.BlocklistPoll {
		level.Warn(util.Logger).Log("msg", "ingester.complete_block_timeout < storage.trace.blocklist_poll",
			"explan", "You may receive 404s between the time the ingesters have flushed a trace and the querier is aware of the new block")
	}

	if c.Compactor.Compactor.BlockRetention < c.StorageConfig.Trace.BlocklistPoll {
		level.Warn(util.Logger).Log("msg", "compactor.compaction.compacted_block_timeout < storage.trace.blocklist_poll",
			"explan", "Queriers and Compactors may attempt to read a block that no longer exists")
	}

	if c.StorageConfig.Trace.Backend == "s3" && c.Compactor.Compactor.FlushSizeBytes < 5242880 {
		level.Warn(util.Logger).Log("msg", "c.Compactor.Compactor.FlushSizeBytes < 5242880",
			"explan", "Compaction flush size should be 5MB or higher for S3 backend")
	}
}

// App is the root datastructure.
type App struct {
	cfg Config

	server       *server.Server
	ring         *ring.Ring
	overrides    *overrides.Overrides
	distributor  *distributor.Distributor
	querier      *querier.Querier
	frontend     *frontend.Frontend
	compactor    *compactor.Compactor
	ingester     *ingester.Ingester
	store        storage.Store
	memberlistKV *memberlist.KVInitService

	httpAuthMiddleware middleware.Interface
	moduleManager      *modules.Manager
	serviceMap         map[string]services.Service
}

// New makes a new app.
func New(cfg Config) (*App, error) {
	app := &App{
		cfg: cfg,
	}

	app.setupAuthMiddleware()

	if err := app.setupModuleManager(); err != nil {
		return nil, fmt.Errorf("failed to setup module manager %w", err)
	}

	return app, nil
}

func (t *App) setupAuthMiddleware() {
	if t.cfg.AuthEnabled {
		t.cfg.Server.GRPCMiddleware = []grpc.UnaryServerInterceptor{
			middleware.ServerUserHeaderInterceptor,
		}
		t.cfg.Server.GRPCStreamMiddleware = []grpc.StreamServerInterceptor{
			func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
				return middleware.StreamServerUserHeaderInterceptor(srv, ss, info, handler)
			},
		}
		t.httpAuthMiddleware = middleware.AuthenticateUser
	} else {
		t.cfg.Server.GRPCMiddleware = []grpc.UnaryServerInterceptor{
			fakeGRPCAuthUniaryMiddleware,
		}
		t.cfg.Server.GRPCStreamMiddleware = []grpc.StreamServerInterceptor{
			fakeGRPCAuthStreamMiddleware,
		}
		t.httpAuthMiddleware = fakeHTTPAuthMiddleware
	}
}

// Run starts, and blocks until a signal is received.
func (t *App) Run() error {
	if !t.moduleManager.IsUserVisibleModule(t.cfg.Target) {
		level.Warn(util.Logger).Log("msg", "selected target is an internal module, is this intended?", "target", t.cfg.Target)
	}

	serviceMap, err := t.moduleManager.InitModuleServices(t.cfg.Target)
	if err != nil {
		return fmt.Errorf("failed to init module services %w", err)
	}
	t.serviceMap = serviceMap

	servs := []services.Service(nil)
	for _, s := range serviceMap {
		servs = append(servs, s)
	}

	sm, err := services.NewManager(servs...)
	if err != nil {
		return fmt.Errorf("failed to start service manager %w", err)
	}

	// before starting servers, register /ready handler and gRPC health check service.
	t.server.HTTP.Path("/ready").Handler(t.readyHandler(sm))
	grpc_health_v1.RegisterHealthServer(t.server.GRPC, healthcheck.New(sm))

	// Let's listen for events from this manager, and log them.
	healthy := func() { level.Info(util.Logger).Log("msg", "Tempo started") }
	stopped := func() { level.Info(util.Logger).Log("msg", "Tempo stopped") }
	serviceFailed := func(service services.Service) {
		// if any service fails, stop everything
		sm.StopAsync()

		// let's find out which module failed
		for m, s := range serviceMap {
			if s == service {
				if service.FailureCase() == util.ErrStopProcess {
					level.Info(util.Logger).Log("msg", "received stop signal via return error", "module", m, "err", service.FailureCase())
				} else {
					level.Error(util.Logger).Log("msg", "module failed", "module", m, "err", service.FailureCase())
				}
				return
			}
		}

		level.Error(util.Logger).Log("msg", "module failed", "module", "unknown", "err", service.FailureCase())
	}
	sm.AddListener(services.NewManagerListener(healthy, stopped, serviceFailed))

	// Setup signal handler. If signal arrives, we stop the manager, which stops all the services.
	handler := signals.NewHandler(t.server.Log)
	go func() {
		handler.Loop()
		sm.StopAsync()
	}()

	// Start all services. This can really only fail if some service is already
	// in other state than New, which should not be the case.
	err = sm.StartAsync(context.Background())
	if err != nil {
		return fmt.Errorf("failed to start service manager %w", err)
	}

	return sm.AwaitStopped(context.Background())
}

func (t *App) readyHandler(sm *services.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !sm.IsHealthy() {
			msg := bytes.Buffer{}
			msg.WriteString("Some services are not Running:\n")

			byState := sm.ServicesByState()
			for st, ls := range byState {
				msg.WriteString(fmt.Sprintf("%v: %d\n", st, len(ls)))
			}

			http.Error(w, msg.String(), http.StatusServiceUnavailable)
			return
		}

		// Ingester has a special check that makes sure that it was able to register into the ring,
		// and that all other ring entries are OK too.
		if t.ingester != nil {
			if err := t.ingester.CheckReady(r.Context()); err != nil {
				http.Error(w, "Ingester not ready: "+err.Error(), http.StatusServiceUnavailable)
				return
			}
		}

		http.Error(w, "ready", http.StatusOK)
	}
}
