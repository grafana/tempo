package app

import (
	"fmt"
	"net/http"
	"path"

	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/kv/codec"
	"github.com/grafana/dskit/kv/memberlist"
	"github.com/grafana/dskit/modules"
	"github.com/grafana/dskit/ring"
	"github.com/grafana/dskit/services"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/thanos-io/thanos/pkg/discovery/dns"
	"github.com/weaveworks/common/middleware"
	"github.com/weaveworks/common/server"

	"github.com/grafana/tempo/modules/compactor"
	"github.com/grafana/tempo/modules/distributor"
	"github.com/grafana/tempo/modules/frontend"
	frontend_v1pb "github.com/grafana/tempo/modules/frontend/v1/frontendv1pb"
	"github.com/grafana/tempo/modules/generator"
	"github.com/grafana/tempo/modules/ingester"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/modules/querier"
	tempo_storage "github.com/grafana/tempo/modules/storage"
	"github.com/grafana/tempo/pkg/api"
	tempo_ring "github.com/grafana/tempo/pkg/ring"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/log"
)

// The various modules that make up tempo.
const (
	Ring                 string = "ring"
	MetricsGeneratorRing string = "metrics-generator-ring"
	Overrides            string = "overrides"
	Server               string = "server"
	Distributor          string = "distributor"
	Ingester             string = "ingester"
	MetricsGenerator     string = "metrics-generator"
	Querier              string = "querier"
	QueryFrontend        string = "query-frontend"
	Compactor            string = "compactor"
	Store                string = "store"
	MemberlistKV         string = "memberlist-kv"
	SingleBinary         string = "all"
	ScalableSingleBinary string = "scalable-single-binary"
)

func (t *App) initServer() (services.Service, error) {
	t.cfg.Server.MetricsNamespace = metricsNamespace
	t.cfg.Server.ExcludeRequestInLog = true

	DisableSignalHandling(&t.cfg.Server)

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
	s := NewServerService(server, servicesToWaitFor)

	return s, nil
}

func (t *App) initRing() (services.Service, error) {
	ring, err := tempo_ring.New(t.cfg.Ingester.LifecyclerConfig.RingConfig, "ingester", t.cfg.Ingester.OverrideRingKey, prometheus.DefaultRegisterer)
	if err != nil {
		return nil, fmt.Errorf("failed to create ring %w", err)
	}
	t.ring = ring

	t.Server.HTTP.Handle("/ingester/ring", t.ring)

	return t.ring, nil
}

func (t *App) initGeneratorRing() (services.Service, error) {
	generatorRing, err := tempo_ring.New(t.cfg.Generator.Ring.ToRingConfig(), "metrics-generator", generator.RingKey, prometheus.DefaultRegisterer)
	if err != nil {
		return nil, fmt.Errorf("failed to create metrics-generator ring %w", err)
	}
	t.generatorRing = generatorRing

	t.Server.HTTP.Handle("/metrics-generator/ring", t.generatorRing)

	return t.generatorRing, nil
}

func (t *App) initOverrides() (services.Service, error) {
	overrides, err := overrides.NewOverrides(t.cfg.LimitsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create overrides %w", err)
	}
	t.overrides = overrides

	prometheus.MustRegister(&t.cfg.LimitsConfig)

	if t.cfg.LimitsConfig.PerTenantOverrideConfig != "" {
		prometheus.MustRegister(t.overrides)
	}

	return t.overrides, nil
}

func (t *App) initDistributor() (services.Service, error) {
	// todo: make ingester client a module instead of passing the config everywhere
	distributor, err := distributor.New(t.cfg.Distributor, t.cfg.IngesterClient, t.ring, t.cfg.GeneratorClient, t.generatorRing, t.overrides, t.TracesConsumerMiddleware, t.cfg.Server.LogLevel, t.cfg.SearchEnabled, t.cfg.MetricsGeneratorEnabled, prometheus.DefaultRegisterer)
	if err != nil {
		return nil, fmt.Errorf("failed to create distributor %w", err)
	}
	t.distributor = distributor

	if distributor.DistributorRing != nil {
		t.Server.HTTP.Handle("/distributor/ring", distributor.DistributorRing)
	}

	return t.distributor, nil
}

func (t *App) initIngester() (services.Service, error) {
	t.cfg.Ingester.LifecyclerConfig.ListenPort = t.cfg.Server.GRPCListenPort
	ingester, err := ingester.New(t.cfg.Ingester, t.store, t.overrides, prometheus.DefaultRegisterer)
	if err != nil {
		return nil, fmt.Errorf("failed to create ingester: %w", err)
	}
	t.ingester = ingester

	tempopb.RegisterPusherServer(t.Server.GRPC, t.ingester)
	tempopb.RegisterQuerierServer(t.Server.GRPC, t.ingester)
	t.Server.HTTP.Path("/flush").Handler(http.HandlerFunc(t.ingester.FlushHandler))
	t.Server.HTTP.Path("/shutdown").Handler(http.HandlerFunc(t.ingester.ShutdownHandler))
	return t.ingester, nil
}

func (t *App) initGenerator() (services.Service, error) {
	t.cfg.Generator.Ring.ListenPort = t.cfg.Server.GRPCListenPort
	generator, err := generator.New(&t.cfg.Generator, t.overrides, prometheus.DefaultRegisterer, log.Logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create metrics-generator %w", err)
	}
	t.generator = generator

	tempopb.RegisterMetricsGeneratorServer(t.Server.GRPC, t.generator)
	return t.generator, nil
}

func (t *App) initQuerier() (services.Service, error) {
	// validate worker config
	// if we're not in single binary mode and worker address is not specified - bail
	if t.cfg.Target != SingleBinary && t.cfg.Querier.Worker.FrontendAddress == "" {
		return nil, fmt.Errorf("frontend worker address not specified")
	} else if t.cfg.Target == SingleBinary {
		// if we're in single binary mode with no worker address specified, register default endpoint
		if t.cfg.Querier.Worker.FrontendAddress == "" {
			t.cfg.Querier.Worker.FrontendAddress = fmt.Sprintf("127.0.0.1:%d", t.cfg.Server.GRPCListenPort)
			level.Warn(log.Logger).Log("msg", "Worker address is empty in single binary mode.  Attempting automatic worker configuration.  If queries are unresponsive consider configuring the worker explicitly.", "address", t.cfg.Querier.Worker.FrontendAddress)
		}
	}

	// do not enable polling if this is the single binary. in that case the compactor will take care of polling
	if t.cfg.Target == Querier {
		t.store.EnablePolling(nil)
	}

	// todo: make ingester client a module instead of passing config everywhere
	querier, err := querier.New(t.cfg.Querier, t.cfg.IngesterClient, t.ring, t.store, t.overrides)
	if err != nil {
		return nil, fmt.Errorf("failed to create querier %w", err)
	}
	t.querier = querier

	middleware := middleware.Merge(
		t.HTTPAuthMiddleware,
	)

	tracesHandler := middleware.Wrap(http.HandlerFunc(t.querier.TraceByIDHandler))
	t.Server.HTTP.Handle(path.Join(api.PathPrefixQuerier, addHTTPAPIPrefix(&t.cfg, api.PathTraces)), tracesHandler)

	if t.cfg.SearchEnabled {
		searchHandler := t.HTTPAuthMiddleware.Wrap(http.HandlerFunc(t.querier.SearchHandler))
		t.Server.HTTP.Handle(path.Join(api.PathPrefixQuerier, addHTTPAPIPrefix(&t.cfg, api.PathSearch)), searchHandler)

		searchTagsHandler := t.HTTPAuthMiddleware.Wrap(http.HandlerFunc(t.querier.SearchTagsHandler))
		t.Server.HTTP.Handle(path.Join(api.PathPrefixQuerier, addHTTPAPIPrefix(&t.cfg, api.PathSearchTags)), searchTagsHandler)

		searchTagValuesHandler := t.HTTPAuthMiddleware.Wrap(http.HandlerFunc(t.querier.SearchTagValuesHandler))
		t.Server.HTTP.Handle(path.Join(api.PathPrefixQuerier, addHTTPAPIPrefix(&t.cfg, api.PathSearchTagValues)), searchTagValuesHandler)
	}

	return t.querier, t.querier.CreateAndRegisterWorker(t.Server.HTTPServer.Handler)
}

func (t *App) initQueryFrontend() (services.Service, error) {
	// cortexTripper is a bridge between http and httpgrpc. it does the job of passing data to the cortex
	// frontend code
	cortexTripper, v1, _, err := frontend.InitFrontend(t.cfg.Frontend.Config, frontend.CortexNoQuerierLimits{}, 0, log.Logger, prometheus.DefaultRegisterer)
	if err != nil {
		return nil, err
	}
	t.frontend = v1

	// create query frontend
	queryFrontend, err := frontend.New(t.cfg.Frontend, cortexTripper, t.store, log.Logger, prometheus.DefaultRegisterer)
	if err != nil {
		return nil, err
	}

	// wrap handlers with auth
	middleware := middleware.Merge(
		t.HTTPAuthMiddleware,
		httpGzipMiddleware(),
	)

	traceByIDHandler := middleware.Wrap(queryFrontend.TraceByID)
	searchHandler := middleware.Wrap(queryFrontend.Search)

	// register grpc server for queriers to connect to
	frontend_v1pb.RegisterFrontendServer(t.Server.GRPC, t.frontend)

	// http trace by id endpoint
	t.Server.HTTP.Handle(addHTTPAPIPrefix(&t.cfg, api.PathTraces), traceByIDHandler)

	// http search endpoints
	if t.cfg.SearchEnabled {
		t.Server.HTTP.Handle(addHTTPAPIPrefix(&t.cfg, api.PathSearch), searchHandler)
		t.Server.HTTP.Handle(addHTTPAPIPrefix(&t.cfg, api.PathSearchTags), searchHandler)
		t.Server.HTTP.Handle(addHTTPAPIPrefix(&t.cfg, api.PathSearchTagValues), searchHandler)

		t.store.EnablePolling(nil) // the query frontend does not need to have knowledge of the backend unless it is building jobs for backend search
	}

	// http query echo endpoint
	t.Server.HTTP.Handle(addHTTPAPIPrefix(&t.cfg, api.PathEcho), echoHandler())

	// todo: queryFrontend should implement service.Service and take the cortex frontend a submodule
	return t.frontend, nil
}

func (t *App) initCompactor() (services.Service, error) {
	compactor, err := compactor.New(t.cfg.Compactor, t.store, t.overrides, prometheus.DefaultRegisterer)
	if err != nil {
		return nil, fmt.Errorf("failed to create compactor %w", err)
	}
	t.compactor = compactor

	if t.compactor.Ring != nil {
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
	reg := prometheus.DefaultRegisterer
	t.cfg.MemberlistKV.MetricsRegisterer = reg
	t.cfg.MemberlistKV.MetricsNamespace = metricsNamespace
	t.cfg.MemberlistKV.Codecs = []codec.Codec{
		ring.GetCodec(),
	}

	dnsProviderReg := prometheus.WrapRegistererWithPrefix(
		"tempo_",
		prometheus.WrapRegistererWith(
			prometheus.Labels{"name": "memberlist"},
			reg,
		),
	)

	dnsProvider := dns.NewProvider(log.Logger, dnsProviderReg, dns.GolangResolverType)
	t.MemberlistKV = memberlist.NewKVInitService(&t.cfg.MemberlistKV, log.Logger, dnsProvider, reg)

	t.cfg.Ingester.LifecyclerConfig.RingConfig.KVStore.MemberlistKV = t.MemberlistKV.GetMemberlistKV
	t.cfg.Generator.Ring.KVStore.MemberlistKV = t.MemberlistKV.GetMemberlistKV
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
	mm.RegisterModule(MetricsGeneratorRing, t.initGeneratorRing, modules.UserInvisibleModule)
	mm.RegisterModule(Overrides, t.initOverrides, modules.UserInvisibleModule)
	mm.RegisterModule(Distributor, t.initDistributor)
	mm.RegisterModule(Ingester, t.initIngester)
	mm.RegisterModule(Querier, t.initQuerier)
	mm.RegisterModule(QueryFrontend, t.initQueryFrontend)
	mm.RegisterModule(Compactor, t.initCompactor)
	mm.RegisterModule(MetricsGenerator, t.initGenerator)
	mm.RegisterModule(Store, t.initStore, modules.UserInvisibleModule)
	mm.RegisterModule(SingleBinary, nil)
	mm.RegisterModule(ScalableSingleBinary, nil)

	deps := map[string][]string{
		// Server:       nil,
		// Store:        nil,
		Overrides:            {Server},
		MemberlistKV:         {Server},
		QueryFrontend:        {Store, Server},
		Ring:                 {Server, MemberlistKV},
		MetricsGeneratorRing: {Server, MemberlistKV},
		Distributor:          {Ring, Server, Overrides},
		Ingester:             {Store, Server, Overrides, MemberlistKV},
		MetricsGenerator:     {Server, Overrides, MemberlistKV},
		Querier:              {Store, Ring, Overrides},
		Compactor:            {Store, Server, Overrides, MemberlistKV},
		SingleBinary:         {Compactor, QueryFrontend, Querier, Ingester, Distributor},
		ScalableSingleBinary: {SingleBinary},
	}

	if t.cfg.MetricsGeneratorEnabled {
		// If metrics-generator is enabled, the distributor needs the metrics-generator ring
		deps[Distributor] = append(deps[Distributor], MetricsGeneratorRing)
		// Add the metrics generator as dependency for when target is {,scalable-}single-binary
		deps[SingleBinary] = append(deps[SingleBinary], MetricsGenerator)
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
