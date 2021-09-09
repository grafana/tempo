package app

import (
	"fmt"
	"net/http"
	"path"

	"github.com/cortexproject/cortex/pkg/cortex"
	cortex_frontend "github.com/cortexproject/cortex/pkg/frontend"
	cortex_transport "github.com/cortexproject/cortex/pkg/frontend/transport"
	cortex_frontend_v1pb "github.com/cortexproject/cortex/pkg/frontend/v1/frontendv1pb"
	"github.com/cortexproject/cortex/pkg/ring"
	"github.com/cortexproject/cortex/pkg/ring/kv/codec"
	"github.com/cortexproject/cortex/pkg/ring/kv/memberlist"
	"github.com/cortexproject/cortex/pkg/util/log"
	"github.com/go-kit/kit/log/level"
	"github.com/grafana/dskit/modules"
	"github.com/grafana/dskit/services"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/weaveworks/common/middleware"
	"github.com/weaveworks/common/server"

	"github.com/grafana/tempo/modules/compactor"
	"github.com/grafana/tempo/modules/distributor"
	"github.com/grafana/tempo/modules/frontend"
	"github.com/grafana/tempo/modules/ingester"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/modules/querier"
	tempo_storage "github.com/grafana/tempo/modules/storage"
	tempo_ring "github.com/grafana/tempo/pkg/ring"
	"github.com/grafana/tempo/pkg/tempopb"
)

// The various modules that make up tempo.
const (
	Ring          string = "ring"
	Overrides     string = "overrides"
	Server        string = "server"
	Distributor   string = "distributor"
	Ingester      string = "ingester"
	Querier       string = "querier"
	QueryFrontend string = "query-frontend"
	Compactor     string = "compactor"
	Store         string = "store"
	MemberlistKV  string = "memberlist-kv"
	All           string = "all"
)

const (
	apiPathTraces          string = "/api/traces/{traceID}"
	apiPathSearch          string = "/api/search"
	apiPathSearchTags      string = "/api/search/tags"
	apiPathSearchTagValues string = "/api/search/tag/{tagName}/values"
	apiPathEcho            string = "/api/echo"
)

func (t *App) initServer() (services.Service, error) {
	t.cfg.Server.MetricsNamespace = metricsNamespace
	t.cfg.Server.ExcludeRequestInLog = true

	cortex.DisableSignalHandling(&t.cfg.Server)

	server, err := server.New(t.cfg.Server)
	if err != nil {
		return nil, fmt.Errorf("failed to create server %w", err)
	}

	servicesToWaitFor := func() []services.Service {
		svs := []services.Service(nil)
		for m, s := range t.serviceMap {
			// Server should not wait for itself.
			if m != Server {
				svs = append(svs, s)
			}
		}
		return svs
	}

	t.Server = server
	s := cortex.NewServerService(server, servicesToWaitFor)

	return s, nil
}

func (t *App) initRing() (services.Service, error) {
	ring, err := tempo_ring.New(t.cfg.Ingester.LifecyclerConfig.RingConfig, "ingester", t.cfg.Ingester.OverrideRingKey, prometheus.DefaultRegisterer)
	if err != nil {
		return nil, fmt.Errorf("failed to create ring %w", err)
	}
	t.ring = ring

	prometheus.MustRegister(t.ring)
	t.Server.HTTP.Handle("/ingester/ring", t.ring)

	return t.ring, nil
}

func (t *App) initOverrides() (services.Service, error) {
	overrides, err := overrides.NewOverrides(t.cfg.LimitsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create overrides %w", err)
	}
	t.overrides = overrides

	t.Server.HTTP.Handle("/runtime_config", overrides.Handler())

	return t.overrides, nil
}

func (t *App) initDistributor() (services.Service, error) {
	// todo: make ingester client a module instead of passing the config everywhere
	distributor, err := distributor.New(t.cfg.Distributor, t.cfg.IngesterClient, t.ring, t.overrides, t.cfg.MultitenancyIsEnabled(), t.cfg.Server.LogLevel, t.cfg.SearchEnabled)
	if err != nil {
		return nil, fmt.Errorf("failed to create distributor %w", err)
	}
	t.distributor = distributor

	if distributor.DistributorRing != nil {
		prometheus.MustRegister(distributor.DistributorRing)
		t.Server.HTTP.Handle("/distributor/ring", distributor.DistributorRing)
	}

	return t.distributor, nil
}

func (t *App) initIngester() (services.Service, error) {
	t.cfg.Ingester.LifecyclerConfig.ListenPort = t.cfg.Server.GRPCListenPort
	ingester, err := ingester.New(t.cfg.Ingester, t.store, t.overrides)
	if err != nil {
		return nil, fmt.Errorf("failed to create ingester %w", err)
	}
	t.ingester = ingester

	tempopb.RegisterPusherServer(t.Server.GRPC, t.ingester)
	tempopb.RegisterQuerierServer(t.Server.GRPC, t.ingester)
	t.Server.HTTP.Path("/flush").Handler(http.HandlerFunc(t.ingester.FlushHandler))
	t.Server.HTTP.Path("/shutdown").Handler(http.HandlerFunc(t.ingester.ShutdownHandler))
	return t.ingester, nil
}

func (t *App) initQuerier() (services.Service, error) {
	// validate worker config
	// if we're not in single binary mode and worker address is not specified - bail
	if t.cfg.Target != All && t.cfg.Querier.Worker.FrontendAddress == "" {
		return nil, fmt.Errorf("frontend worker address not specified")
	} else if t.cfg.Target == All {
		// if we're in single binary mode with no worker address specified, register default endpoint
		if t.cfg.Querier.Worker.FrontendAddress == "" {
			t.cfg.Querier.Worker.FrontendAddress = fmt.Sprintf("127.0.0.1:%d", t.cfg.Server.GRPCListenPort)
			level.Warn(log.Logger).Log("msg", "Worker address is empty in single binary mode.  Attempting automatic worker configuration.  If queries are unresponsive consider configuring the worker explicitly.", "address", t.cfg.Querier.Worker.FrontendAddress)
		}
	}

	// do not enable polling if this is the single binary. in that case the compactor will take care of polling
	enablePolling := t.cfg.Target == Querier

	// todo: make ingester client a module instead of passing config everywhere
	querier, err := querier.New(t.cfg.Querier, t.cfg.IngesterClient, t.ring, t.store, t.overrides, enablePolling)
	if err != nil {
		return nil, fmt.Errorf("failed to create querier %w", err)
	}
	t.querier = querier

	middleware := middleware.Merge(
		t.HTTPAuthMiddleware,
	)

	tracesHandler := middleware.Wrap(http.HandlerFunc(t.querier.TraceByIDHandler))
	t.Server.HTTP.Handle(path.Join("/querier", addHTTPAPIPrefix(&t.cfg, apiPathTraces)), tracesHandler)

	if t.cfg.SearchEnabled {
		searchHandler := middleware.Wrap(http.HandlerFunc(t.querier.SearchHandler))
		t.Server.HTTP.Handle(path.Join("/querier", addHTTPAPIPrefix(&t.cfg, apiPathSearch)), searchHandler)

		searchTagsHandler := middleware.Wrap(http.HandlerFunc(t.querier.SearchTagsHandler))
		t.Server.HTTP.Handle(path.Join("/querier", addHTTPAPIPrefix(&t.cfg, apiPathSearchTags)), searchTagsHandler)

		searchTagValuesHandler := middleware.Wrap(http.HandlerFunc(t.querier.SearchTagValuesHandler))
		t.Server.HTTP.Handle(path.Join("/querier", addHTTPAPIPrefix(&t.cfg, apiPathSearchTagValues)), searchTagValuesHandler)
	}

	return t.querier, t.querier.CreateAndRegisterWorker(t.Server.HTTPServer.Handler)
}

func (t *App) initQueryFrontend() (services.Service, error) {
	if t.cfg.Frontend.QueryShards < frontend.MinQueryShards || t.cfg.Frontend.QueryShards > frontend.MaxQueryShards {
		return nil, fmt.Errorf("frontend query shards should be between %d and %d (both inclusive)", frontend.MinQueryShards, frontend.MaxQueryShards)
	}

	cortexTripper, v1, _, err := cortex_frontend.InitFrontend(t.cfg.Frontend.Config, frontend.CortexNoQuerierLimits{}, 0, log.Logger, prometheus.DefaultRegisterer)
	if err != nil {
		return nil, err
	}
	t.frontend = v1

	tripperware, err := frontend.NewTripperware(t.cfg.Frontend, t.cfg.HTTPAPIPrefix, log.Logger, prometheus.DefaultRegisterer)
	if err != nil {
		return nil, err
	}
	roundTripper := tripperware(cortexTripper)

	frontendHandler := cortex_transport.NewHandler(t.cfg.Frontend.Config.Handler, roundTripper, log.Logger, prometheus.DefaultRegisterer)

	frontendHandler = middleware.Merge(
		t.HTTPAuthMiddleware,
	).Wrap(frontendHandler)

	// register grpc server for queriers to connect to
	cortex_frontend_v1pb.RegisterFrontendServer(t.Server.GRPC, t.frontend)

	// http query endpoint
	t.Server.HTTP.Handle(addHTTPAPIPrefix(&t.cfg, apiPathTraces), frontendHandler)

	// http search endpoints
	if t.cfg.SearchEnabled {
		t.Server.HTTP.Handle(addHTTPAPIPrefix(&t.cfg, apiPathSearch), frontendHandler)
		t.Server.HTTP.Handle(addHTTPAPIPrefix(&t.cfg, apiPathSearchTags), frontendHandler)
		t.Server.HTTP.Handle(addHTTPAPIPrefix(&t.cfg, apiPathSearchTagValues), frontendHandler)
	}

	// http query echo endpoint
	t.Server.HTTP.Handle(addHTTPAPIPrefix(&t.cfg, apiPathEcho), echoHandler())

	return t.frontend, nil
}

func (t *App) initCompactor() (services.Service, error) {
	compactor, err := compactor.New(t.cfg.Compactor, t.store, t.overrides)
	if err != nil {
		return nil, fmt.Errorf("failed to create compactor %w", err)
	}
	t.compactor = compactor

	if t.compactor.Ring != nil {
		prometheus.MustRegister(t.compactor.Ring)
		t.Server.HTTP.Handle("/compactor/ring", t.compactor.Ring)
	}

	return t.compactor, nil
}

func (t *App) initStore() (services.Service, error) {
	store, err := tempo_storage.NewStore(t.cfg.StorageConfig, log.Logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create store %w", err)
	}
	t.store = store

	return t.store, nil
}

func (t *App) initMemberlistKV() (services.Service, error) {
	t.cfg.MemberlistKV.MetricsRegisterer = prometheus.DefaultRegisterer
	t.cfg.MemberlistKV.MetricsNamespace = metricsNamespace
	t.cfg.MemberlistKV.Codecs = []codec.Codec{
		ring.GetCodec(),
	}

	t.MemberlistKV = memberlist.NewKVInitService(&t.cfg.MemberlistKV, log.Logger)

	t.cfg.Ingester.LifecyclerConfig.RingConfig.KVStore.MemberlistKV = t.MemberlistKV.GetMemberlistKV
	t.cfg.Distributor.DistributorRing.KVStore.MemberlistKV = t.MemberlistKV.GetMemberlistKV
	t.cfg.Compactor.ShardingRing.KVStore.MemberlistKV = t.MemberlistKV.GetMemberlistKV

	t.Server.HTTP.Handle("/memberlist", t.MemberlistKV)

	return t.MemberlistKV, nil
}

func (t *App) setupModuleManager() error {
	mm := modules.NewManager(log.Logger)

	mm.RegisterModule(Server, t.initServer, modules.UserInvisibleModule)
	mm.RegisterModule(MemberlistKV, t.initMemberlistKV, modules.UserInvisibleModule)
	mm.RegisterModule(Ring, t.initRing, modules.UserInvisibleModule)
	mm.RegisterModule(Overrides, t.initOverrides, modules.UserInvisibleModule)
	mm.RegisterModule(Distributor, t.initDistributor)
	mm.RegisterModule(Ingester, t.initIngester)
	mm.RegisterModule(Querier, t.initQuerier)
	mm.RegisterModule(QueryFrontend, t.initQueryFrontend)
	mm.RegisterModule(Compactor, t.initCompactor)
	mm.RegisterModule(Store, t.initStore, modules.UserInvisibleModule)
	mm.RegisterModule(All, nil)

	deps := map[string][]string{
		// Server:       nil,
		// Store:        nil,
		Overrides:     {Server},
		MemberlistKV:  {Server},
		QueryFrontend: {Server},
		Ring:          {Server, MemberlistKV},
		Distributor:   {Ring, Server, Overrides},
		Ingester:      {Store, Server, Overrides, MemberlistKV},
		Querier:       {Store, Ring},
		Compactor:     {Store, Server, Overrides, MemberlistKV},
		All:           {Compactor, QueryFrontend, Querier, Ingester, Distributor},
	}

	for mod, targets := range deps {
		if err := mm.AddDependency(mod, targets...); err != nil {
			return err
		}
	}

	t.ModuleManager = mm

	return nil
}

func addHTTPAPIPrefix(cfg *Config, apiPath string) string {
	return path.Join(cfg.HTTPAPIPrefix, apiPath)
}

func echoHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "echo", http.StatusOK)
	}
}
