package app

import (
	"flag"
	"fmt"

	"github.com/cortexproject/cortex/pkg/chunk"
	"github.com/cortexproject/cortex/pkg/ring"
	"github.com/cortexproject/cortex/pkg/util"

	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/weaveworks/common/middleware"
	"github.com/weaveworks/common/server"
	"google.golang.org/grpc"

	"github.com/joe-elliott/frigg/pkg/distributor"
	"github.com/joe-elliott/frigg/pkg/ingester"
	"github.com/joe-elliott/frigg/pkg/ingester/client"
	"github.com/joe-elliott/frigg/pkg/storage"
	"github.com/joe-elliott/frigg/pkg/util/validation"
)

// Config is the root config for App.
type Config struct {
	Target      moduleName `yaml:"target,omitempty"`
	AuthEnabled bool       `yaml:"auth_enabled,omitempty"`
	HTTPPrefix  string     `yaml:"http_prefix"`

	Server           server.Config      `yaml:"server,omitempty"`
	Distributor      distributor.Config `yaml:"distributor,omitempty"`
	IngesterClient   client.Config      `yaml:"ingester_client,omitempty"`
	Ingester         ingester.Config    `yaml:"ingester,omitempty"`
	StorageConfig    storage.Config     `yaml:"storage_config,omitempty"`
	ChunkStoreConfig chunk.StoreConfig  `yaml:"chunk_store_config,omitempty"`
	SchemaConfig     chunk.SchemaConfig `yaml:"schema_config,omitempty"`
	LimitsConfig     validation.Limits  `yaml:"limits_config,omitempty"`
}

// RegisterFlags registers flag.
func (c *Config) RegisterFlags(f *flag.FlagSet) {
	c.Server.MetricsNamespace = "frigg"
	c.Target = All
	c.Server.ExcludeRequestInLog = true
	f.Var(&c.Target, "target", "target module (default All)")
	f.BoolVar(&c.AuthEnabled, "auth.enabled", true, "Set to false to disable auth.")

	c.Server.RegisterFlags(f)
	c.Distributor.RegisterFlags(f)
	c.IngesterClient.RegisterFlags(f)
	c.Ingester.RegisterFlags(f)
	c.StorageConfig.RegisterFlags(f)
	c.ChunkStoreConfig.RegisterFlags(f)
	c.SchemaConfig.RegisterFlags(f)
	c.LimitsConfig.RegisterFlags(f)
}

// App is the root datastructure.
type App struct {
	cfg Config

	server      *server.Server
	ring        *ring.Ring
	overrides   *validation.Overrides
	distributor *distributor.Distributor
	ingester    *ingester.Ingester
	store       storage.Store

	httpAuthMiddleware middleware.Interface
}

// New makes a new app.
func New(cfg Config) (*App, error) {
	app := &App{
		cfg: cfg,
	}

	app.setupAuthMiddleware()

	if err := app.init(cfg.Target); err != nil {
		return nil, err
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
				switch info.FullMethod {
				// Don't check auth header on TransferChunks, as we weren't originally
				// sending it and this could cause transfers to fail on update.
				//
				// Also don't check auth /frontend.Frontend/Process, as this handles
				// queries for multiple users.
				case "/logproto.Ingester/TransferChunks", "/frontend.Frontend/Process":
					return handler(srv, ss)
				default:
					return middleware.StreamServerUserHeaderInterceptor(srv, ss, info, handler)
				}
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

func (t *App) init(m moduleName) error {
	// initialize all of our dependencies first
	for _, dep := range orderedDeps(m) {
		if err := t.initModule(dep); err != nil {
			return err
		}
	}
	// lastly, initialize the requested module
	return t.initModule(m)
}

func (t *App) initModule(m moduleName) error {
	level.Info(util.Logger).Log("msg", "initialising", "module", m)
	if modules[m].init != nil {
		if err := modules[m].init(t); err != nil {
			return errors.Wrap(err, fmt.Sprintf("error initialising module: %s", m))
		}
	}
	return nil
}

// Run starts, and blocks until a signal is received.
func (t *App) Run() error {
	return t.server.Run()
}

// Stop gracefully stops a Loki.
func (t *App) Stop() error {
	t.stopping(t.cfg.Target)
	t.stop(t.cfg.Target)
	t.server.Shutdown()
	return nil
}

func (t *App) stop(m moduleName) {
	t.stopModule(m)
	deps := orderedDeps(m)
	// iterate over our deps in reverse order and call stopModule
	for i := len(deps) - 1; i >= 0; i-- {
		t.stopModule(deps[i])
	}
}

func (t *App) stopModule(m moduleName) {
	level.Info(util.Logger).Log("msg", "stopping", "module", m)
	if modules[m].stop != nil {
		if err := modules[m].stop(t); err != nil {
			level.Error(util.Logger).Log("msg", "error stopping", "module", m, "err", err)
		}
	}
}

func (t *App) stopping(m moduleName) {
	t.stoppingModule(m)
	deps := orderedDeps(m)
	// iterate over our deps in reverse order and call stoppingModule
	for i := len(deps) - 1; i >= 0; i-- {
		t.stoppingModule(deps[i])
	}
}

func (t *App) stoppingModule(m moduleName) {
	level.Info(util.Logger).Log("msg", "notifying module about stopping", "module", m)
	if modules[m].stopping != nil {
		if err := modules[m].stopping(t); err != nil {
			level.Error(util.Logger).Log("msg", "error stopping", "module", m, "err", err)
		}
	}
}
