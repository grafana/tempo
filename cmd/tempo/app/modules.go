package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cortexproject/cortex/pkg/ring"
	"github.com/cortexproject/cortex/pkg/ring/kv/codec"
	"github.com/cortexproject/cortex/pkg/ring/kv/memberlist"
	"github.com/cortexproject/cortex/pkg/util"
	"github.com/cortexproject/cortex/pkg/util/services"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/weaveworks/common/middleware"
	"github.com/weaveworks/common/server"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/grafana/tempo/pkg/compactor"
	"github.com/grafana/tempo/pkg/distributor"
	"github.com/grafana/tempo/pkg/ingester"
	"github.com/grafana/tempo/pkg/querier"
	tempo_storage "github.com/grafana/tempo/pkg/storage"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/validation"
)

type moduleName int

// The various modules that make up tempo.
const (
	Ring moduleName = iota
	Overrides
	Server
	Distributor
	Ingester
	Querier
	Compactor
	Store
	MemberlistKV
	All
)

func (m *moduleName) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var val string
	if err := unmarshal(&val); err != nil {
		return err
	}

	return m.Set(val)
}

func (m moduleName) String() string {
	switch m {
	case Ring:
		return "ring"
	case Overrides:
		return "overrides"
	case Server:
		return "server"
	case Distributor:
		return "distributor"
	case Store:
		return "store"
	case Ingester:
		return "ingester"
	case Querier:
		return "querier"
	case Compactor:
		return "compactor"
	case MemberlistKV:
		return "memberlist-kv"
	case All:
		return "all"
	default:
		panic(fmt.Sprintf("unknown module name: %d", m))
	}
}

func (m *moduleName) Set(s string) error {
	switch strings.ToLower(s) {
	case "server":
		*m = Server
		return nil
	case "ring":
		*m = Ring
		return nil
	case "overrides":
		*m = Overrides
		return nil
	case "distributor":
		*m = Distributor
		return nil
	case "store":
		*m = Store
		return nil
	case "ingester":
		*m = Ingester
		return nil
	case "querier":
		*m = Querier
		return nil
	case "compactor":
		*m = Compactor
		return nil
	case "all":
		*m = All
		return nil
	default:
		return fmt.Errorf("unrecognised module name: %s", s)
	}
}

func (t *App) initServer() (err error) {
	t.server, err = server.New(t.cfg.Server)
	return
}

func (t *App) initRing() (err error) {
	t.cfg.Ingester.LifecyclerConfig.RingConfig.KVStore.MemberlistKV = t.memberlistKV.GetMemberlistKV
	t.ring, err = ring.New(t.cfg.Ingester.LifecyclerConfig.RingConfig, "ingester", ring.IngesterRingKey, prometheus.DefaultRegisterer)
	if err != nil {
		return err
	}
	err = services.StartAndAwaitRunning(context.Background(), t.ring)
	if err != nil {
		return err
	}
	prometheus.MustRegister(t.ring)
	t.server.HTTP.Handle("/ring", t.ring)
	return
}

func (t *App) stopRing() (err error) {
	return services.StopAndAwaitTerminated(context.Background(), t.ring)
}

func (t *App) initOverrides() (err error) {
	t.overrides, err = validation.NewOverrides(t.cfg.LimitsConfig)
	return err
}

func (t *App) initDistributor() (err error) {
	t.cfg.Distributor.DistributorRing.KVStore.MemberlistKV = t.memberlistKV.GetMemberlistKV
	t.distributor, err = distributor.New(t.cfg.Distributor, t.cfg.IngesterClient, t.ring, t.overrides, t.cfg.AuthEnabled)
	if err != nil {
		return
	}

	pushHandler := middleware.Merge(
		t.httpAuthMiddleware,
	).Wrap(http.HandlerFunc(t.distributor.PushHandler))

	t.server.HTTP.Path("/ready").Handler(http.HandlerFunc(t.distributor.ReadinessHandler))
	t.server.HTTP.Handle("/api/v0/push", pushHandler)
	return
}

func (t *App) stopDistributor() (err error) {
	t.distributor.Stop()
	return nil
}

func (t *App) initIngester() (err error) {
	t.cfg.Ingester.LifecyclerConfig.ListenPort = t.cfg.Server.GRPCListenPort
	t.cfg.Ingester.LifecyclerConfig.RingConfig.KVStore.MemberlistKV = t.memberlistKV.GetMemberlistKV
	t.ingester, err = ingester.New(t.cfg.Ingester, t.cfg.IngesterClient, t.store, t.overrides)
	if err != nil {
		return
	}

	tempopb.RegisterPusherServer(t.server.GRPC, t.ingester)
	tempopb.RegisterQuerierServer(t.server.GRPC, t.ingester)
	grpc_health_v1.RegisterHealthServer(t.server.GRPC, t.ingester)
	t.server.HTTP.Path("/ready").Handler(http.HandlerFunc(t.ingester.ReadinessHandler))
	t.server.HTTP.Path("/flush").Handler(http.HandlerFunc(t.ingester.FlushHandler))
	return
}

func (t *App) stopIngester() error {
	t.ingester.Shutdown()
	return nil
}

func (t *App) stoppingIngester() error {
	t.ingester.Stopping()
	return nil
}

func (t *App) initQuerier() (err error) {
	t.querier, err = querier.New(t.cfg.Querier, t.cfg.IngesterClient, t.ring, t.store, t.overrides)
	if err != nil {
		return
	}

	tracesHandler := middleware.Merge(
		t.httpAuthMiddleware,
	).Wrap(http.HandlerFunc(t.querier.TraceByIDHandler))

	t.server.HTTP.Path("/ready").Handler(http.HandlerFunc(t.querier.ReadinessHandler))
	t.server.HTTP.Handle("/api/traces/{traceID}", tracesHandler)

	return
}

func (t *App) initCompactor() (err error) {
	t.cfg.Compactor.ShardingRing.KVStore.MemberlistKV = t.memberlistKV.GetMemberlistKV
	t.compactor, err = compactor.New(t.cfg.Compactor, t.store)
	if err != nil {
		return err
	}

	t.server.HTTP.Handle("/ring-compactor", t.compactor.Ring)

	go func() {
		err := t.compactor.Start(t.cfg.StorageConfig)
		if err != nil {
			log.Fatalf("Error starting compactor: %v", err)
		}
	}()

	return nil
}

func (t *App) stopCompactor() (err error) {
	t.compactor.Shutdown()
	return nil
}

func (t *App) initStore() (err error) {
	t.store, err = tempo_storage.NewStore(t.cfg.StorageConfig, t.overrides, util.Logger)
	return
}

func (t *App) stopStore() error {
	t.store.Shutdown()
	return nil
}

func (t *App) initMemberlistKV() error {
	t.cfg.MemberlistKV.MetricsRegisterer = prometheus.DefaultRegisterer
	t.cfg.MemberlistKV.Codecs = []codec.Codec{
		ring.GetCodec(),
	}

	hostname, err := os.Hostname()
	if err != nil {
		return err
	}
	t.cfg.MemberlistKV.NodeName = hostname + "-" + uuid.New().String()

	t.memberlistKV = memberlist.NewKVInitService(&t.cfg.MemberlistKV)
	return nil
}

func (t *App) stopMemberlistKV() error {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelFunc()

	t.memberlistKV.StopAsync()
	t.memberlistKV.AwaitTerminated(ctx)
	return nil
}

// listDeps recursively gets a list of dependencies for a passed moduleName
func listDeps(m moduleName) []moduleName {
	deps := modules[m].deps
	for _, d := range modules[m].deps {
		deps = append(deps, listDeps(d)...)
	}
	return deps
}

// orderedDeps gets a list of all dependencies ordered so that items are always after any of their dependencies.
func orderedDeps(m moduleName) []moduleName {
	// get a unique list of dependencies and init a map to keep whether they have been added to our result
	deps := uniqueDeps(listDeps(m))
	added := map[moduleName]bool{}

	result := make([]moduleName, 0, len(deps))

	// keep looping through all modules until they have all been added to the result.
	for len(result) < len(deps) {
	OUTER:
		for _, name := range deps {
			if added[name] {
				continue
			}

			for _, dep := range modules[name].deps {
				// stop processing this module if one of its dependencies has
				// not been added to the result yet.
				if !added[dep] {
					continue OUTER
				}
			}

			// if all of the module's dependencies have been added to the result slice,
			// then we can safely add this module to the result slice as well.
			added[name] = true
			result = append(result, name)
		}
	}

	return result
}

// uniqueDeps returns the unique list of input dependencies, guaranteeing input order stability
func uniqueDeps(deps []moduleName) []moduleName {
	result := make([]moduleName, 0, len(deps))
	uniq := map[moduleName]bool{}

	for _, dep := range deps {
		if !uniq[dep] {
			result = append(result, dep)
			uniq[dep] = true
		}
	}

	return result
}

type module struct {
	deps     []moduleName
	init     func(a *App) error
	stopping func(a *App) error
	stop     func(a *App) error
}

var modules = map[moduleName]module{
	Server: {
		init: (*App).initServer,
	},

	Ring: {
		deps: []moduleName{Server, MemberlistKV},
		init: (*App).initRing,
		stop: (*App).stopRing,
	},

	Overrides: {
		init: (*App).initOverrides,
	},

	Distributor: {
		deps: []moduleName{Ring, Server, Overrides},
		init: (*App).initDistributor,
		stop: (*App).stopDistributor,
	},

	Ingester: {
		deps:     []moduleName{Store, Server, MemberlistKV},
		init:     (*App).initIngester,
		stop:     (*App).stopIngester,
		stopping: (*App).stoppingIngester,
	},

	Querier: {
		deps: []moduleName{Store, Ring, Server},
		init: (*App).initQuerier,
	},

	Compactor: {
		deps: []moduleName{Store, Server, MemberlistKV},
		init: (*App).initCompactor,
		stop: (*App).stopCompactor,
	},

	Store: {
		deps: []moduleName{Overrides},
		init: (*App).initStore,
		stop: (*App).stopStore,
	},

	MemberlistKV: {
		init: (*App).initMemberlistKV,
		stop: (*App).stopMemberlistKV,
	},

	All: {
		deps: []moduleName{Compactor, Querier, Ingester, Distributor},
	},
}
