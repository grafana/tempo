package app

import (
	"fmt"
	"net/http"
	"os"

	"github.com/cortexproject/cortex/pkg/cortex"
	"github.com/cortexproject/cortex/pkg/ring"
	"github.com/cortexproject/cortex/pkg/ring/kv/codec"
	"github.com/cortexproject/cortex/pkg/ring/kv/memberlist"
	"github.com/cortexproject/cortex/pkg/util"
	"github.com/cortexproject/cortex/pkg/util/modules"
	"github.com/cortexproject/cortex/pkg/util/services"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/weaveworks/common/middleware"
	"github.com/weaveworks/common/server"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/grafana/tempo/modules/compactor"
	"github.com/grafana/tempo/modules/distributor"
	"github.com/grafana/tempo/modules/ingester"
	"github.com/grafana/tempo/modules/querier"
	tempo_storage "github.com/grafana/tempo/modules/storage"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/validation"
)

type moduleName int

// The various modules that make up tempo.
const (
	Ring         string = "ring"
	Overrides    string = "overrides"
	Server       string = "server"
	Distributor  string = "distributor"
	Ingester     string = "ingester"
	Querier      string = "querier"
	Compactor    string = "compactor"
	Store        string = "store"
	MemberlistKV string = "memberlist-kv"
	All          string = "all"
)

func (t *App) initServer() (services.Service, error) {
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
	ring, err := ring.New(t.cfg.Ingester.LifecyclerConfig.RingConfig, "ingester", ring.IngesterRingKey, prometheus.DefaultRegisterer)
	if err != nil {
		return nil, fmt.Errorf("failed to create ring %w", err)
	}
	t.ring = ring

	prometheus.MustRegister(t.ring)
	t.server.HTTP.Handle("/ring", t.ring) // jpe - put this someplace else /distributor/ring?  coordinate with below

	return t.ring, nil
}

func (t *App) initOverrides() (services.Service, error) {
	overrides, err := validation.NewOverrides(t.cfg.LimitsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create overrides %w", err)
	}
	t.overrides = overrides
	// overrides don't have operational state, nor do they need to do anything more in starting/stopping phase,
	// so there is no need to return any service.
	return nil, nil
}

func (t *App) initDistributor() (services.Service, error) {
	distributor, err := distributor.New(t.cfg.Distributor, t.cfg.IngesterClient, t.ring, t.overrides, t.cfg.AuthEnabled) // jpe, make ingester client a module?
	if err != nil {
		return nil, fmt.Errorf("failed to create distributor %w", err)
	}
	t.distributor = distributor

	t.server.HTTP.Path("/ready").Handler(http.HandlerFunc(t.distributor.ReadinessHandler)) // jpe use global readiness handler like cortex
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
	grpc_health_v1.RegisterHealthServer(t.server.GRPC, t.ingester)                      // jpe ??
	t.server.HTTP.Path("/ready").Handler(http.HandlerFunc(t.ingester.ReadinessHandler)) // jpe use global readiness handler like cortex
	t.server.HTTP.Path("/flush").Handler(http.HandlerFunc(t.ingester.FlushHandler))
	return t.ingester, nil
}

func (t *App) initQuerier() (services.Service, error) {
	querier, err := querier.New(t.cfg.Querier, t.cfg.IngesterClient, t.ring, t.store, t.overrides) // jpe ingester client as module?
	if err != nil {
		return nil, fmt.Errorf("failed to create querier %w", err)
	}
	t.querier = querier

	tracesHandler := middleware.Merge(
		t.httpAuthMiddleware,
	).Wrap(http.HandlerFunc(t.querier.TraceByIDHandler))

	t.server.HTTP.Path("/ready").Handler(http.HandlerFunc(t.querier.ReadinessHandler)) // jpe global readiness?
	t.server.HTTP.Handle("/api/traces/{traceID}", tracesHandler)

	return t.querier, nil
}

func (t *App) initCompactor() (services.Service, error) {
	t.cfg.Compactor.WaitOnStartup = t.cfg.StorageConfig.Trace.MaintenanceCycle // force the compactor to wait one maintenance cycle to start to prevent collisions
	compactor, err := compactor.New(t.cfg.Compactor, t.store)
	if err != nil {
		return nil, fmt.Errorf("failed to create compactor %w", err)
	}
	t.compactor = compactor

	t.server.HTTP.Handle("/ring-compactor", t.compactor.Ring) // jpe - put this someplace else /compactor/ring?  coordinate with above

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
	t.cfg.MemberlistKV.Codecs = []codec.Codec{
		ring.GetCodec(),
	}

	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("failed to get hostname %w", err)
	}
	t.cfg.MemberlistKV.NodeName = hostname + "-" + uuid.New().String() // jpe i don't think this is needed anymore

	t.cfg.Ingester.LifecyclerConfig.RingConfig.KVStore.MemberlistKV = t.memberlistKV.GetMemberlistKV
	t.cfg.Distributor.DistributorRing.KVStore.MemberlistKV = t.memberlistKV.GetMemberlistKV
	t.cfg.Compactor.ShardingRing.KVStore.MemberlistKV = t.memberlistKV.GetMemberlistKV

	t.memberlistKV = memberlist.NewKVInitService(&t.cfg.MemberlistKV)
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
	mm.RegisterModule(Compactor, t.initCompactor)
	mm.RegisterModule(Store, t.initStore, modules.UserInvisibleModule)

	deps := map[string][]string{
		// Server:       nil,
		// Overrides:    nil,
		// Store:        nil,
		// MemberlistKV: nil,
		Ring:        {Server, MemberlistKV},
		Distributor: {Ring, Server, Overrides},
		Ingester:    {Store, Server, MemberlistKV},
		Querier:     {Store, Ring},
		Compactor:   {Store, Server, MemberlistKV},
		All:         {Compactor, Querier, Ingester, Distributor},
	}

	for mod, targets := range deps {
		if err := mm.AddDependency(mod, targets...); err != nil {
			return err
		}
	}

	t.moduleManager = mm

	return nil
}
