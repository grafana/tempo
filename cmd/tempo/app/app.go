package app

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	cortex_frontend "github.com/cortexproject/cortex/pkg/frontend/v1"
	"github.com/cortexproject/cortex/pkg/util"
	"github.com/cortexproject/cortex/pkg/util/log"
	"github.com/go-kit/log/level"
	"github.com/gorilla/mux"
	"github.com/grafana/dskit/flagext"
	"github.com/grafana/dskit/grpcutil"
	"github.com/grafana/dskit/kv/memberlist"
	"github.com/grafana/dskit/modules"
	"github.com/grafana/dskit/ring"
	"github.com/grafana/dskit/services"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/pkg/errors"
	"github.com/prometheus/common/version"
	"github.com/weaveworks/common/middleware"
	"github.com/weaveworks/common/server"
	"github.com/weaveworks/common/signals"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
	"gopkg.in/yaml.v3"

	"github.com/grafana/tempo/modules/compactor"
	"github.com/grafana/tempo/modules/distributor"
	"github.com/grafana/tempo/modules/distributor/receiver"
	"github.com/grafana/tempo/modules/frontend"
	"github.com/grafana/tempo/modules/ingester"
	ingester_client "github.com/grafana/tempo/modules/ingester/client"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/modules/querier"
	"github.com/grafana/tempo/modules/storage"
	tempo_util "github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb"
)

const metricsNamespace = "tempo"
const apiDocs = "https://grafana.com/docs/tempo/latest/api_docs/"

// Config is the root config for App.
type Config struct {
	Target              string `yaml:"target,omitempty"`
	AuthEnabled         bool   `yaml:"auth_enabled,omitempty"`
	MultitenancyEnabled bool   `yaml:"multitenancy_enabled,omitempty"`
	SearchEnabled       bool   `yaml:"search_enabled,omitempty"`
	HTTPAPIPrefix       string `yaml:"http_api_prefix"`
	UseOTelTracer       bool   `yaml:"use_otel_tracer,omitempty"`

	Server         server.Config          `yaml:"server,omitempty"`
	Distributor    distributor.Config     `yaml:"distributor,omitempty"`
	IngesterClient ingester_client.Config `yaml:"ingester_client,omitempty"`
	Querier        querier.Config         `yaml:"querier,omitempty"`
	Frontend       frontend.Config        `yaml:"query_frontend,omitempty"`
	Compactor      compactor.Config       `yaml:"compactor,omitempty"`
	Ingester       ingester.Config        `yaml:"ingester,omitempty"`
	StorageConfig  storage.Config         `yaml:"storage,omitempty"`
	LimitsConfig   overrides.Limits       `yaml:"overrides,omitempty"`
	MemberlistKV   memberlist.KVConfig    `yaml:"memberlist,omitempty"`
}

// RegisterFlagsAndApplyDefaults registers flag.
func (c *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	c.Target = SingleBinary
	// global settings
	f.StringVar(&c.Target, "target", SingleBinary, "target module")
	f.BoolVar(&c.AuthEnabled, "auth.enabled", false, "Set to true to enable auth (deprecated: use multitenancy.enabled)")
	f.BoolVar(&c.MultitenancyEnabled, "multitenancy.enabled", false, "Set to true to enable multitenancy.")
	f.BoolVar(&c.SearchEnabled, "search.enabled", false, "Set to true to enable search (unstable).")
	f.StringVar(&c.HTTPAPIPrefix, "http-api-prefix", "", "String prefix for all http api endpoints.")
	f.BoolVar(&c.UseOTelTracer, "use-otel-tracer", false, "Set to true to replace the OpenTracing tracer with the OpenTelemetry tracer")

	// Server settings
	flagext.DefaultValues(&c.Server)
	c.Server.LogLevel.RegisterFlags(f)

	// The following GRPC server settings are added to address this issue - https://github.com/grafana/tempo/issues/493
	// The settings prevent the grpc server from sending a GOAWAY message if a client sends heartbeat messages
	// too frequently (due to lack of real traffic).
	c.Server.GRPCServerMinTimeBetweenPings = 10 * time.Second
	c.Server.GRPCServerPingWithoutStreamAllowed = true

	f.IntVar(&c.Server.HTTPListenPort, "server.http-listen-port", 80, "HTTP server listen port.")
	f.IntVar(&c.Server.GRPCListenPort, "server.grpc-listen-port", 9095, "gRPC server listen port.")

	// Memberlist settings
	fs := flag.NewFlagSet("", flag.PanicOnError) // create a new flag set b/c we don't want all of the memberlist settings in our flags. we're just doing this to get defaults
	c.MemberlistKV.RegisterFlags(fs)
	_ = fs.Parse([]string{})
	// these defaults were chosen to balance resource usage vs. ring propagation speed. they are a "toned down" version of
	// the memberlist defaults
	c.MemberlistKV.RetransmitMult = 2
	c.MemberlistKV.GossipInterval = time.Second
	c.MemberlistKV.GossipNodes = 2
	c.MemberlistKV.EnableCompression = false

	f.Var(&c.MemberlistKV.JoinMembers, "memberlist.host-port", "Host port to connect to memberlist cluster.")
	f.IntVar(&c.MemberlistKV.TCPTransport.BindPort, "memberlist.bind-port", 7946, "Port for memberlist to communicate on")

	// Everything else
	flagext.DefaultValues(&c.IngesterClient)
	c.IngesterClient.GRPCClientConfig.GRPCCompression = "snappy"
	flagext.DefaultValues(&c.LimitsConfig)

	c.Distributor.RegisterFlagsAndApplyDefaults(tempo_util.PrefixConfig(prefix, "distributor"), f)
	c.Ingester.RegisterFlagsAndApplyDefaults(tempo_util.PrefixConfig(prefix, "ingester"), f)
	c.Querier.RegisterFlagsAndApplyDefaults(tempo_util.PrefixConfig(prefix, "querier"), f)
	c.Frontend.RegisterFlagsAndApplyDefaults(tempo_util.PrefixConfig(prefix, "frontend"), f)
	c.Compactor.RegisterFlagsAndApplyDefaults(tempo_util.PrefixConfig(prefix, "compactor"), f)
	c.StorageConfig.RegisterFlagsAndApplyDefaults(tempo_util.PrefixConfig(prefix, "storage"), f)

}

// MultitenancyIsEnabled checks if multitenancy is enabled
func (c *Config) MultitenancyIsEnabled() bool {
	return c.MultitenancyEnabled || c.AuthEnabled
}

// CheckConfig checks if config values are suspect.
func (c *Config) CheckConfig() {
	if c.Ingester.CompleteBlockTimeout < c.StorageConfig.Trace.BlocklistPoll {
		level.Warn(log.Logger).Log("msg", "ingester.complete_block_timeout < storage.trace.blocklist_poll",
			"explan", "You may receive 404s between the time the ingesters have flushed a trace and the querier is aware of the new block")
	}

	if c.Compactor.Compactor.BlockRetention < c.StorageConfig.Trace.BlocklistPoll {
		level.Warn(log.Logger).Log("msg", "compactor.compaction.compacted_block_timeout < storage.trace.blocklist_poll",
			"explan", "Queriers and Compactors may attempt to read a block that no longer exists")
	}

	if c.Compactor.Compactor.RetentionConcurrency == 0 {
		level.Warn(log.Logger).Log("msg", "c.Compactor.Compactor.RetentionConcurrency must be greater than zero. Using default.", "default", tempodb.DefaultRetentionConcurrency)
	}

	if c.StorageConfig.Trace.Backend == "s3" && c.Compactor.Compactor.FlushSizeBytes < 5242880 {
		level.Warn(log.Logger).Log("msg", "c.Compactor.Compactor.FlushSizeBytes < 5242880",
			"explan", "Compaction flush size should be 5MB or higher for S3 backend")
	}

	if c.StorageConfig.Trace.BlocklistPollConcurrency == 0 {
		level.Warn(log.Logger).Log("msg", "c.StorageConfig.Trace.BlocklistPollConcurrency must be greater than zero. Using default.", "default", tempodb.DefaultBlocklistPollConcurrency)
	}
}

func newDefaultConfig() *Config {
	defaultConfig := &Config{}
	defaultFS := flag.NewFlagSet("", flag.PanicOnError)
	defaultConfig.RegisterFlagsAndApplyDefaults("", defaultFS)
	return defaultConfig
}

// App is the root datastructure.
type App struct {
	cfg Config

	Server       *server.Server
	ring         *ring.Ring
	overrides    *overrides.Overrides
	distributor  *distributor.Distributor
	querier      *querier.Querier
	frontend     *cortex_frontend.Frontend
	compactor    *compactor.Compactor
	ingester     *ingester.Ingester
	store        storage.Store
	MemberlistKV *memberlist.KVInitService

	HTTPAuthMiddleware       middleware.Interface
	TracesConsumerMiddleware receiver.Middleware

	ModuleManager *modules.Manager
	serviceMap    map[string]services.Service
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
	if t.cfg.MultitenancyIsEnabled() {

		// don't check auth for these gRPC methods, since single call is used for multiple users
		noGRPCAuthOn := []string{
			"/frontend.Frontend/Process",
			"/frontend.Frontend/NotifyClientShutdown",
		}
		ignoredMethods := map[string]bool{}
		for _, m := range noGRPCAuthOn {
			ignoredMethods[m] = true
		}

		t.cfg.Server.GRPCMiddleware = []grpc.UnaryServerInterceptor{
			func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
				if ignoredMethods[info.FullMethod] {
					return handler(ctx, req)
				}
				return middleware.ServerUserHeaderInterceptor(ctx, req, info, handler)
			},
		}
		t.cfg.Server.GRPCStreamMiddleware = []grpc.StreamServerInterceptor{
			func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
				if ignoredMethods[info.FullMethod] {
					return handler(srv, ss)
				}
				return middleware.StreamServerUserHeaderInterceptor(srv, ss, info, handler)
			},
		}
		t.HTTPAuthMiddleware = middleware.AuthenticateUser
		t.TracesConsumerMiddleware = receiver.MultiTenancyMiddleware()
	} else {
		t.cfg.Server.GRPCMiddleware = []grpc.UnaryServerInterceptor{
			fakeGRPCAuthUniaryMiddleware,
		}
		t.cfg.Server.GRPCStreamMiddleware = []grpc.StreamServerInterceptor{
			fakeGRPCAuthStreamMiddleware,
		}
		t.HTTPAuthMiddleware = fakeHTTPAuthMiddleware
		t.TracesConsumerMiddleware = receiver.FakeTenantMiddleware()
	}
}

// Run starts, and blocks until a signal is received.
func (t *App) Run() error {
	if !t.ModuleManager.IsUserVisibleModule(t.cfg.Target) {
		level.Warn(log.Logger).Log("msg", "selected target is an internal module, is this intended?", "target", t.cfg.Target)
	}

	serviceMap, err := t.ModuleManager.InitModuleServices(t.cfg.Target)
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
	t.Server.HTTP.Path("/ready").Handler(t.readyHandler(sm))
	t.Server.HTTP.Path("/status").Handler(t.statusHandler()).Methods("GET")
	t.Server.HTTP.Path("/status/{endpoint}").Handler(t.statusHandler()).Methods("GET")
	grpc_health_v1.RegisterHealthServer(t.Server.GRPC, grpcutil.NewHealthCheck(sm))

	// Let's listen for events from this manager, and log them.
	healthy := func() { level.Info(log.Logger).Log("msg", "Tempo started") }
	stopped := func() { level.Info(log.Logger).Log("msg", "Tempo stopped") }
	serviceFailed := func(service services.Service) {
		// if any service fails, stop everything
		sm.StopAsync()

		// let's find out which module failed
		for m, s := range serviceMap {
			if s == service {
				if service.FailureCase() == modules.ErrStopProcess {
					level.Info(log.Logger).Log("msg", "received stop signal via return error", "module", m, "err", service.FailureCase())
				} else {
					level.Error(log.Logger).Log("msg", "module failed", "module", m, "err", service.FailureCase())
				}
				return
			}
		}

		level.Error(log.Logger).Log("msg", "module failed", "module", "unknown", "err", service.FailureCase())
	}
	sm.AddListener(services.NewManagerListener(healthy, stopped, serviceFailed))

	// Setup signal handler. If signal arrives, we stop the manager, which stops all the services.
	handler := signals.NewHandler(t.Server.Log)
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

func (t *App) writeStatusVersion(w io.Writer) error {
	_, err := w.Write([]byte(version.Print("tempo") + "\n"))
	if err != nil {
		return err
	}

	return nil
}

func (t *App) writeStatusConfig(w io.Writer, r *http.Request) error {
	var output interface{}

	mode := r.URL.Query().Get("mode")
	switch mode {
	case "diff":
		defaultCfg := newDefaultConfig()

		defaultCfgYaml, err := util.YAMLMarshalUnmarshal(defaultCfg)
		if err != nil {
			return err
		}

		cfgYaml, err := util.YAMLMarshalUnmarshal(t.cfg)
		if err != nil {
			return err
		}

		output, err = util.DiffConfig(defaultCfgYaml, cfgYaml)
		if err != nil {
			return err
		}
	case "defaults":
		output = newDefaultConfig()
	case "":
		output = t.cfg
	default:
		return errors.Errorf("unknown value for mode query parameter: %v", mode)
	}

	out, err := yaml.Marshal(output)
	if err != nil {
		return err
	}

	_, err = w.Write([]byte("---\n"))
	if err != nil {
		return err
	}

	_, err = w.Write(out)
	if err != nil {
		return err
	}

	return nil
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

		// Query Frontend has a special check that makes sure that a querier is attached before it signals
		// itself as ready
		if t.frontend != nil {
			if err := t.frontend.CheckReady(r.Context()); err != nil {
				http.Error(w, "Query Frontend not ready: "+err.Error(), http.StatusServiceUnavailable)
				return
			}
		}

		http.Error(w, "ready", http.StatusOK)
	}
}

func (t *App) writeRuntimeConfig(w io.Writer, r *http.Request) error {
	// Querier and query-frontend services do not run the overrides module
	if t.overrides == nil {
		_, err := w.Write([]byte(fmt.Sprintf("overrides module not loaded in %s\n", t.cfg.Target)))
		return err
	}
	return t.overrides.WriteStatusRuntimeConfig(w, r)
}

func (t *App) statusHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var errs []error
		msg := bytes.Buffer{}

		simpleEndpoints := map[string]func(io.Writer) error{
			"version":   t.writeStatusVersion,
			"services":  t.writeStatusServices,
			"endpoints": t.writeStatusEndpoints,
		}

		wrapStatus := func(endpoint string) {
			msg.WriteString("GET /status/" + endpoint + "\n")

			switch endpoint {
			case "runtime_config":
				err := t.writeRuntimeConfig(&msg, r)
				if err != nil {
					errs = append(errs, err)
				}
			case "config":
				err := t.writeStatusConfig(&msg, r)
				if err != nil {
					errs = append(errs, err)
				}
			default:
				err := simpleEndpoints[endpoint](&msg)
				if err != nil {
					errs = append(errs, err)
				}
			}
		}

		vars := mux.Vars(r)

		if endpoint, ok := vars["endpoint"]; ok {
			wrapStatus(endpoint)
		} else {
			wrapStatus("version")
			wrapStatus("services")
			wrapStatus("endpoints")
			wrapStatus("runtime_config")
			wrapStatus("config")
		}

		w.Header().Set("Content-Type", "text/plain")

		joinErrors := func(errs []error) error {
			if len(errs) == 0 {
				return nil
			}
			var err error

			for _, e := range errs {
				if e != nil {
					if err == nil {
						err = e
					} else {
						err = errors.Wrap(err, e.Error())
					}
				}
			}
			return err
		}

		err := joinErrors(errs)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}

		if _, err := w.Write(msg.Bytes()); err != nil {
			level.Error(log.Logger).Log("msg", "error writing response", "err", err)
		}
	}
}

func (t *App) writeStatusServices(w io.Writer) error {
	svcNames := make([]string, 0, len(t.serviceMap))
	for name := range t.serviceMap {
		svcNames = append(svcNames, name)
	}

	sort.Strings(svcNames)

	x := table.NewWriter()
	x.SetOutputMirror(w)
	x.AppendHeader(table.Row{"service name", "status", "failure case"})

	for _, name := range svcNames {
		service := t.serviceMap[name]

		var e string

		if err := service.FailureCase(); err != nil {
			e = err.Error()
		}

		x.AppendRows([]table.Row{
			{name, service.State(), e},
		})
	}

	x.AppendSeparator()
	x.Render()

	return nil
}

func (t *App) writeStatusEndpoints(w io.Writer) error {
	type endpoint struct {
		name  string
		regex string
	}

	endpoints := []endpoint{}

	err := t.Server.HTTP.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		e := endpoint{}

		pathTemplate, err := route.GetPathTemplate()
		if err == nil {
			e.name = pathTemplate
		}

		pathRegexp, err := route.GetPathRegexp()
		if err == nil {
			e.regex = pathRegexp
		}

		endpoints = append(endpoints, e)

		return nil
	})
	if err != nil {
		return errors.Wrap(err, "error walking routes")
	}

	sort.Slice(endpoints[:], func(i, j int) bool {
		return endpoints[i].name < endpoints[j].name
	})

	x := table.NewWriter()
	x.SetOutputMirror(w)
	x.AppendHeader(table.Row{"name", "regex"})

	for _, e := range endpoints {
		x.AppendRows([]table.Row{
			{e.name, e.regex},
		})
	}

	x.AppendSeparator()
	x.Render()

	_, err = w.Write([]byte(fmt.Sprintf("\nAPI documentation: %s\n\n", apiDocs)))
	if err != nil {
		return errors.Wrap(err, "error writing status endpoints")
	}

	return nil
}
