package app

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cortexproject/cortex/pkg/chunk"
	"github.com/cortexproject/cortex/pkg/chunk/storage"
	"github.com/cortexproject/cortex/pkg/querier/frontend"
	"github.com/cortexproject/cortex/pkg/ring"
	"github.com/cortexproject/cortex/pkg/util"

	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	httpgrpc_server "github.com/weaveworks/common/httpgrpc/server"
	"github.com/weaveworks/common/middleware"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/joe-elliott/frigg/pkg/distributor"
	"github.com/joe-elliott/frigg/pkg/ingester"
	loki_storage "github.com/joe-elliott/frigg/pkg/storage"
	"github.com/joe-elliott/frigg/pkg/util/validation"
)

const maxChunkAgeForTableManager = 12 * time.Hour

type moduleName int

// The various modules that make up Loki.
const (
	Ring moduleName = iota
	Overrides
	Server
	Distributor
	Ingester
	Store
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
	case "all":
		*m = All
		return nil
	default:
		return fmt.Errorf("unrecognised module name: %s", s)
	}
}

func (t *Loki) initServer() (err error) {
	t.server, err = server.New(t.cfg.Server)
	return
}

func (a *App) initRing() (err error) {
	if t.cfg.Ingester.LifecyclerConfig.RingConfig.ReplicationFactor != 1 {
		return fmt.Errorf("frigg only supports a replication factor of 1")
	}

	t.ring, err = ring.New(t.cfg.Ingester.LifecyclerConfig.RingConfig, "ingester", ring.IngesterRingKey)
	if err != nil {
		return
	}
	prometheus.MustRegister(t.ring)
	t.server.HTTP.Handle("/ring", t.ring)
	return
}

func (t *Loki) initOverrides() (err error) {
	t.overrides, err = validation.NewOverrides(t.cfg.LimitsConfig)
	return err
}

func (a *App) initDistributor() (err error) {
	t.distributor, err = distributor.New(t.cfg.Distributor, t.cfg.IngesterClient, t.ring, t.overrides)
	if err != nil {
		return
	}

	pushHandler := middleware.Merge(
		t.httpAuthMiddleware,
	).Wrap(http.HandlerFunc(t.distributor.PushHandler))

	friggpb.RegisterPusherServer(t.server.GRPC, t.distributor)
	t.server.HTTP.Path("/ready").Handler(http.HandlerFunc(t.distributor.ReadinessHandler))
	return
}

func (a *App) stopDistributor() (err error) {
	t.distributor.Stop()
	return nil
}

func (a *App) initIngester() (err error) {
	t.cfg.Ingester.LifecyclerConfig.ListenPort = &t.cfg.Server.GRPCListenPort
	t.ingester, err = ingester.New(t.cfg.Ingester, t.cfg.IngesterClient, t.store, t.overrides)
	if err != nil {
		return
	}

	friggpb.RegisterPusherServer(t.server.GRPC, t.ingester)
	grpc_health_v1.RegisterHealthServer(t.server.GRPC, t.ingester)
	t.server.HTTP.Path("/ready").Handler(http.HandlerFunc(t.ingester.ReadinessHandler))
	t.server.HTTP.Path("/flush").Handler(http.HandlerFunc(t.ingester.FlushHandler))
	return
}

func (a *App) stopIngester() error {
	t.ingester.Shutdown()
	return nil
}

func (a *App) stoppingIngester() error {
	t.ingester.Stopping()
	return nil
}

func (a *App) initStore() (err error) {
	t.store, err = loki_storage.NewStore(t.cfg.StorageConfig, t.cfg.ChunkStoreConfig, t.cfg.SchemaConfig, t.overrides)
	return
}

func (a *App) stopStore() error {
	t.store.Stop()
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
		deps: []moduleName{Server},
		init: (*App).initRing,
	},

	Overrides: {
		init: (*App).initOverrides,
	},

	Distributor: {
		deps: []moduleName{Ring, Server, Overrides},
		init: (*App).initDistributor,
		stop: (*App).stopDistributor,
	},

	Store: {
		deps: []moduleName{Overrides},
		init: (*App).initStore,
		stop: (*App).stopStore,
	},

	Ingester: {
		deps:     []moduleName{Store, Server},
		init:     (*App).initIngester,
		stop:     (*App).stopIngester,
		stopping: (*App).stoppingIngester,
	},

	All: {
		deps: []moduleName{Querier, Ingester, Distributor, TableManager},
	},
}
