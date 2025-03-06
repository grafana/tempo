package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-kit/log/level"
	"github.com/gorilla/mux"
	"github.com/grafana/dskit/grpcutil"
	"github.com/grafana/dskit/kv/memberlist"
	"github.com/grafana/dskit/middleware"
	"github.com/grafana/dskit/modules"
	"github.com/grafana/dskit/ring"
	"github.com/grafana/dskit/server"
	"github.com/grafana/dskit/services"
	"github.com/grafana/dskit/signals"
	"github.com/grafana/tempo/modules/backendworker"
	"github.com/grafana/tempo/modules/blockbuilder"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/prometheus/common/version"
	"go.uber.org/atomic"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
	"gopkg.in/yaml.v3"

	"github.com/grafana/tempo/cmd/tempo/build"
	"github.com/grafana/tempo/modules/backendscheduler"
	"github.com/grafana/tempo/modules/compactor"
	"github.com/grafana/tempo/modules/distributor"
	"github.com/grafana/tempo/modules/distributor/receiver"
	frontend_v1 "github.com/grafana/tempo/modules/frontend/v1"
	"github.com/grafana/tempo/modules/generator"
	"github.com/grafana/tempo/modules/ingester"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/modules/querier"
	"github.com/grafana/tempo/modules/storage"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/cache"
	"github.com/grafana/tempo/pkg/usagestats"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/log"
	util_log "github.com/grafana/tempo/pkg/util/log"
)

const (
	metricsNamespace = "tempo"
	apiDocs          = "https://grafana.com/docs/tempo/latest/api_docs/"
)

var (
	statFeatureEnabledAuth         = usagestats.NewInt("feature_enabled_auth_stats")
	statFeatureEnabledMultitenancy = usagestats.NewInt("feature_enabled_multitenancy")
)

// App is the root datastructure.
type App struct {
	cfg Config

	Server         TempoServer
	InternalServer *server.Server

	readRings            map[string]*ring.Ring
	partitionRing        *ring.PartitionInstanceRing
	partitionRingWatcher *ring.PartitionRingWatcher
	generatorRingWatcher *ring.PartitionRingWatcher
	Overrides            overrides.Service
	distributor          *distributor.Distributor
	querier              *querier.Querier
	frontend             *frontend_v1.Frontend
	compactor            *compactor.Compactor
	ingester             *ingester.Ingester
	generator            *generator.Generator
	blockBuilder         *blockbuilder.BlockBuilder
	store                storage.Store
	usageReport          *usagestats.Reporter
	cacheProvider        cache.Provider
	MemberlistKV         *memberlist.KVInitService
	backendScheduler     *backendscheduler.BackendScheduler
	backendWorker        *backendworker.BackendWorker

	HTTPAuthMiddleware       middleware.Interface
	TracesConsumerMiddleware receiver.Middleware

	ModuleManager *modules.Manager
	serviceMap    map[string]services.Service
	deps          map[string][]string
}

// New makes a new app.
func New(cfg Config) (*App, error) {
	app := &App{
		cfg:       cfg,
		readRings: map[string]*ring.Ring{},
		Server:    newTempoServer(),
	}

	usagestats.Edition("oss")

	statFeatureEnabledAuth.Set(0)
	if cfg.AuthEnabled {
		statFeatureEnabledAuth.Set(1)
	}

	statFeatureEnabledMultitenancy.Set(0)
	if cfg.MultitenancyEnabled {
		statFeatureEnabledMultitenancy.Set(1)
	}

	app.setupAuthMiddleware()

	if err := app.setupModuleManager(); err != nil {
		return nil, fmt.Errorf("failed to setup module manager: %w", err)
	}

	return app, nil
}

func (t *App) setupAuthMiddleware() {
	if t.cfg.MultitenancyIsEnabled() {

		// don't check auth for these gRPC methods, since single call is used for multiple users
		noGRPCAuthOn := []string{
			"/frontend.Frontend/Process",
			"/frontend.Frontend/NotifyClientShutdown",
			"/tempopb.BackendScheduler/Next",
			"/tempopb.BackendScheduler/UpdateJob",
		}
		ignoredMethods := map[string]bool{}
		for _, m := range noGRPCAuthOn {
			ignoredMethods[m] = true
		}
		spew.Dump("ignoredMethods", ignoredMethods)

		t.cfg.Server.GRPCMiddleware = []grpc.UnaryServerInterceptor{
			func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
				spew.Dump("unary server interceptor", info.FullMethod)
				if ignoredMethods[info.FullMethod] {
					spew.Dump("unary server interceptor ignored", info.FullMethod)
					return handler(ctx, req)
				}
				spew.Dump("unary server interceptor not ignored", info.FullMethod)
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
		return fmt.Errorf("failed to init module services: %w", err)
	}
	t.serviceMap = serviceMap

	servs := []services.Service(nil)
	for _, s := range serviceMap {
		servs = append(servs, s)
	}

	sm, err := services.NewManager(servs...)
	if err != nil {
		return fmt.Errorf("failed to start service manager: %w", err)
	}

	// Used to delay shutdown but return "not ready" during this delay.
	shutdownRequested := atomic.NewBool(false)
	// before starting servers, register /ready handler and gRPC health check service.
	if t.cfg.InternalServer.Enable {
		t.InternalServer.HTTP.Path("/ready").Methods("GET").Handler(t.readyHandler(sm, shutdownRequested))
	}

	t.Server.HTTPRouter().Path(addHTTPAPIPrefix(&t.cfg, api.PathBuildInfo)).Handler(t.buildinfoHandler()).Methods("GET")

	t.Server.HTTPRouter().Path("/ready").Handler(t.readyHandler(sm, shutdownRequested))
	t.Server.HTTPRouter().Path("/status").Handler(t.statusHandler()).Methods("GET")
	t.Server.HTTPRouter().Path("/status/{endpoint}").Handler(t.statusHandler()).Methods("GET")
	grpc_health_v1.RegisterHealthServer(t.Server.GRPC(),
		grpcutil.NewHealthCheckFrom(
			grpcutil.WithShutdownRequested(shutdownRequested),
			grpcutil.WithManager(sm),
		))

	// Let's listen for events from this manager, and log them.
	healthy := func() { level.Info(log.Logger).Log("msg", "Tempo started") }
	stopped := func() { level.Info(log.Logger).Log("msg", "Tempo stopped") }
	serviceFailed := func(service services.Service) {
		// if any service fails, stop everything
		sm.StopAsync()

		// let's find out which module failed
		for m, s := range serviceMap {
			if s == service {
				err = service.FailureCase()
				if errors.Is(err, modules.ErrStopProcess) {
					level.Info(log.Logger).Log("msg", "received stop signal via return error", "module", m, "err", err)
				} else if errors.Is(err, context.Canceled) {
					return
				} else if err != nil {
					level.Error(log.Logger).Log("msg", "module failed", "module", m, "err", err)
				}
				return
			}
		}

		level.Error(log.Logger).Log("msg", "module failed", "module", "unknown", "err", service.FailureCase())
	}
	sm.AddListener(services.NewManagerListener(healthy, stopped, serviceFailed))

	// Setup signal handler. If signal arrives, we stop the manager, which stops all the services.
	handler := signals.NewHandler(t.Server.Log())
	go func() {
		handler.Loop()

		shutdownRequested.Store(true)
		t.Server.SetKeepAlivesEnabled(false)

		if t.cfg.ShutdownDelay > 0 {
			time.Sleep(t.cfg.ShutdownDelay)
		}

		sm.StopAsync()
	}()

	// Start all services. This can really only fail if some service is already
	// in other state than New, which should not be the case.
	err = sm.StartAsync(context.Background())
	if err != nil {
		return fmt.Errorf("failed to start service manager: %w", err)
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
		defaultCfg := NewDefaultConfig()

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
		output = NewDefaultConfig()
	case "":
		output = t.cfg
	default:
		return fmt.Errorf("unknown value for mode query parameter: %v", mode)
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

func (t *App) readyHandler(sm *services.Manager, shutdownRequested *atomic.Bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if shutdownRequested.Load() {
			level.Debug(util_log.Logger).Log("msg", "application is stopping")
			http.Error(w, "Application is stopping", http.StatusServiceUnavailable)
			return
		}

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

		// Generator has a special check that makes sure that it was able to register into the ring,
		// and that all other ring entries are OK too.
		if t.generator != nil {
			if err := t.generator.CheckReady(r.Context()); err != nil {
				http.Error(w, "Generator not ready: "+err.Error(), http.StatusServiceUnavailable)
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
	if t.Overrides == nil {
		_, err := w.Write([]byte(fmt.Sprintf("overrides module not loaded in %s\n", t.cfg.Target)))
		return err
	}
	return t.Overrides.WriteStatusRuntimeConfig(w, r)
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
						err = fmt.Errorf("%s: %w", e.Error(), err)
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

	err := t.Server.HTTPRouter().Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
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
		return fmt.Errorf("error walking routes: %w", err)
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
		return fmt.Errorf("error writing status endpoints: %w", err)
	}

	return nil
}

func (t *App) buildinfoHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(build.GetVersion())

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			level.Error(log.Logger).Log("msg", "error writing response", "err", err)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}
}
