package app

import (
	"fmt"
	"net/http"
	"os"

	"github.com/cortexproject/cortex/pkg/cortex"
	cortex_frontend "github.com/cortexproject/cortex/pkg/querier/frontend"
	"github.com/cortexproject/cortex/pkg/ring"
	"github.com/cortexproject/cortex/pkg/ring/kv/codec"
	"github.com/cortexproject/cortex/pkg/ring/kv/memberlist"
	"github.com/cortexproject/cortex/pkg/util"
	"github.com/cortexproject/cortex/pkg/util/modules"
	"github.com/cortexproject/cortex/pkg/util/services"
	"github.com/go-kit/kit/log/level"
	"github.com/google/uuid"
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
	Ring         string = "ring"
	Overrides    string = "overrides"
	Server       string = "server"
	Distributor  string = "distributor"
	Ingester     string = "ingester"
	Querier      string = "querier"
	Frontend     string = "frontend"
	Compactor    string = "compactor"
	Store        string = "store"
	MemberlistKV string = "memberlist-kv"
	All          string = "all"
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

	t.server = server
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
	t.server.HTTP.Handle("/ingester/ring", t.ring)

	return t.ring, nil
}

func (t *App) initOverrides() (services.Service, error) {
	overrides, err := overrides.NewOverrides(t.cfg.LimitsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create overrides %w", err)
	}
	t.overrides = overrides

	return t.overrides, nil
}

func (t *App) initDistributor() (services.Service, error) {
	// todo: make ingester client a module instead of passing the config everywhere
	distributor, err := distributor.New(t.cfg.Distributor, t.cfg.IngesterClient, t.ring, t.overrides, t.cfg.AuthEnabled, t.cfg.Server.LogLevel)
	if err != nil {
		return nil, fmt.Errorf("failed to create distributor %w", err)
	}
	t.distributor = distributor

	if distributor.DistributorRing != nil {
		prometheus.MustRegister(distributor.DistributorRing)
		t.server.HTTP.Handle("/distributor/ring", distributor.DistributorRing)
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

	tempopb.RegisterPusherServer(t.server.GRPC, t.ingester)
	tempopb.RegisterQuerierServer(t.server.GRPC, t.ingester)
	t.server.HTTP.Path("/flush").Handler(http.HandlerFunc(t.ingester.FlushHandler))
	return t.ingester, nil
}

func (t *App) initQuerier() (services.Service, error) {
	// validate worker config
	// if we're not in single binary mode and worker address is not specified - bail
	if t.cfg.Target != All && t.cfg.Querier.Worker.Address == "" {
		return nil, fmt.Errorf("frontend worker address not specified")
	} else if t.cfg.Target == All {
		// if we're in single binary mode with no worker address specified, register default endpoint
		if t.cfg.Querier.Worker.Address == "" {
			t.cfg.Querier.Worker.Address = fmt.Sprintf("127.0.0.1:%d", t.cfg.Server.GRPCListenPort)
			level.Warn(util.Logger).Log("msg", "Worker address is empty in single binary mode.  Attempting automatic worker configuration.  If queries are unresponsive consider configuring the worker explicitly.", "address", t.cfg.Querier.Worker.Address)
		}
	}

	// todo: make ingester client a module instead of passing config everywhere
	querier, err := querier.New(t.cfg.Querier, t.cfg.IngesterClient, t.ring, t.store, t.overrides)
	if err != nil {
		return nil, fmt.Errorf("failed to create querier %w", err)
	}
	t.querier = querier

	tracesHandler := middleware.Merge(
		t.httpAuthMiddleware,
	).Wrap(http.HandlerFunc(t.querier.TraceByIDHandler))

	t.server.HTTP.Handle("/querier/api/traces/{traceID}", tracesHandler)
	return t.querier, t.querier.CreateAndRegisterWorker(t.server.HTTP)
}

func (t *App) initQueryFrontend() (services.Service, error) {
	var err error
	t.frontend, err = cortex_frontend.New(t.cfg.Frontend.Config, util.Logger, prometheus.DefaultRegisterer)
	if err != nil {
		return nil, err
	}

	// custom tripperware that splits requests
	tripperware, err := frontend.NewTripperware(t.cfg.Frontend, util.Logger, prometheus.DefaultRegisterer)
	if err != nil {
		return nil, err
	}
	// tripperware will be called before f.roundTripper (which calls roundtripgrpc)
	t.frontend.Wrap(tripperware)

	tracesHandler := middleware.Merge(
		t.httpAuthMiddleware,
	).Wrap(t.frontend.Handler())

	cortex_frontend.RegisterFrontendServer(t.server.GRPC, t.frontend)
	t.server.HTTP.Handle("/api/traces/{traceID}", tracesHandler)

	return services.NewIdleService(nil, func(_ error) error {
		t.frontend.Close()
		return nil
	}), nil
}

func (t *App) initCompactor() (services.Service, error) {
	compactor, err := compactor.New(t.cfg.Compactor, t.store)
	if err != nil {
		return nil, fmt.Errorf("failed to create compactor %w", err)
	}
	t.compactor = compactor

	if t.compactor.Ring != nil {
		prometheus.MustRegister(t.compactor.Ring)
		t.server.HTTP.Handle("/compactor/ring", t.compactor.Ring)
	}

	return t.compactor, nil
}

func (t *App) initStore() (services.Service, error) {
	store, err := tempo_storage.NewStore(t.cfg.StorageConfig, util.Logger)
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

	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("failed to get hostname %w", err)
	}
	// todo: do we still need this?  does the package do this by default now?
	t.cfg.MemberlistKV.NodeName = hostname + "-" + uuid.New().String()

	t.memberlistKV = memberlist.NewKVInitService(&t.cfg.MemberlistKV, util.Logger)

	t.cfg.Ingester.LifecyclerConfig.RingConfig.KVStore.MemberlistKV = t.memberlistKV.GetMemberlistKV
	t.cfg.Distributor.DistributorRing.KVStore.MemberlistKV = t.memberlistKV.GetMemberlistKV
	t.cfg.Compactor.ShardingRing.KVStore.MemberlistKV = t.memberlistKV.GetMemberlistKV

	return t.memberlistKV, nil
}

func (t *App) setupModuleManager() error {
	mm := modules.NewManager()

	mm.RegisterModule(Server, t.initServer, modules.UserInvisibleModule)
	mm.RegisterModule(MemberlistKV, t.initMemberlistKV, modules.UserInvisibleModule)
	mm.RegisterModule(Ring, t.initRing, modules.UserInvisibleModule)
	mm.RegisterModule(Overrides, t.initOverrides, modules.UserInvisibleModule)
	mm.RegisterModule(Distributor, t.initDistributor)
	mm.RegisterModule(Ingester, t.initIngester)
	mm.RegisterModule(Querier, t.initQuerier)
	mm.RegisterModule(Frontend, t.initQueryFrontend)
	mm.RegisterModule(Compactor, t.initCompactor)
	mm.RegisterModule(Store, t.initStore, modules.UserInvisibleModule)
	mm.RegisterModule(All, nil)

	deps := map[string][]string{
		// Server:       nil,
		// Overrides:    nil,
		// Store:        nil,
		// MemberlistKV: nil,
		Frontend:    {Server},
		Ring:        {Server, MemberlistKV},
		Distributor: {Ring, Server, Overrides},
		Ingester:    {Store, Server, Overrides, MemberlistKV},
		Querier:     {Store, Ring},
		Compactor:   {Store, Server, MemberlistKV},
		All:         {Compactor, Frontend, Querier, Ingester, Distributor},
	}

	for mod, targets := range deps {
		if err := mm.AddDependency(mod, targets...); err != nil {
			return err
		}
	}

	t.moduleManager = mm

	return nil
}
