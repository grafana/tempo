package app

import (
	"net/http"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/gorilla/mux"
	"github.com/grafana/dskit/kv/consul"
	dskitring "github.com/grafana/dskit/ring"
	"github.com/grafana/dskit/server"
	"github.com/grafana/dskit/services"
	"github.com/grafana/tempo/modules/generator"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/modules/storage"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/wal"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

type fakeTempoServer struct {
	router *mux.Router
	grpc   *grpc.Server
}

func (f *fakeTempoServer) HTTPRouter() *mux.Router   { return f.router }
func (f *fakeTempoServer) HTTPHandler() http.Handler { return f.router }
func (f *fakeTempoServer) GRPC() *grpc.Server        { return f.grpc }
func (f *fakeTempoServer) Log() log.Logger           { return log.NewNopLogger() }
func (f *fakeTempoServer) EnableHTTP2()              {}
func (f *fakeTempoServer) SetKeepAlivesEnabled(bool) {}
func (f *fakeTempoServer) StartAndReturnService(server.Config, bool, func() []services.Service) (services.Service, error) {
	return services.NewIdleService(nil, nil), nil
}

func newTestStore(t *testing.T, tmpDir string) storage.Store {
	t.Helper()

	s, err := storage.NewStore(storage.Config{
		Trace: tempodb.Config{
			Backend: backend.Local,
			Local: &local.Config{
				Path: tmpDir + "/traces",
			},
			Block: &common.BlockConfig{
				BloomFP:             0.01,
				BloomShardSizeBytes: 100_000,
				Version:             encoding.LatestEncoding().Version(),
			},
			WAL: &wal.Config{
				Filepath: tmpDir + "/wal",
			},
			BlocklistPoll: 100 * time.Millisecond,
		},
	}, nil, log.NewNopLogger())
	require.NoError(t, err)

	return s
}

func TestConfigureGenerator(t *testing.T) {
	t.Run("single binary does not consume from kafka", func(t *testing.T) {
		cfg := NewDefaultConfig()
		cfg.Target = SingleBinary
		cfg.Ingest.Kafka.Topic = "tempo"
		cfg.Ingest.Kafka.ConsumerGroup = "custom"

		app := &App{cfg: *cfg}
		app.configureGenerator()

		assert.False(t, app.cfg.Generator.ConsumeFromKafka)
		assert.Equal(t, "tempo", app.cfg.Generator.Ingest.Kafka.Topic)
		assert.Equal(t, "custom", app.cfg.Generator.Ingest.Kafka.ConsumerGroup)
	})

	t.Run("partition ring mode consumes from kafka with fixed consumer group", func(t *testing.T) {
		cfg := NewDefaultConfig()
		cfg.Target = MetricsGenerator
		cfg.Generator.RingMode = generator.RingModePartition
		cfg.Ingest.Kafka.Topic = "tempo"
		cfg.Ingest.Kafka.ConsumerGroup = "custom"

		app := &App{cfg: *cfg}
		app.configureGenerator()

		assert.True(t, app.cfg.Generator.ConsumeFromKafka)
		assert.Equal(t, "tempo", app.cfg.Generator.Ingest.Kafka.Topic)
		assert.Equal(t, generator.ConsumerGroup, app.cfg.Generator.Ingest.Kafka.ConsumerGroup)
	})

	t.Run("generator ring mode consumes from kafka without overriding consumer group", func(t *testing.T) {
		cfg := NewDefaultConfig()
		cfg.Target = MetricsGenerator
		cfg.Generator.RingMode = generator.RingModeGenerator
		cfg.Ingest.Kafka.Topic = "tempo"
		cfg.Ingest.Kafka.ConsumerGroup = "custom"

		app := &App{cfg: *cfg}
		app.configureGenerator()

		assert.True(t, app.cfg.Generator.ConsumeFromKafka)
		assert.Equal(t, "custom", app.cfg.Generator.Ingest.Kafka.ConsumerGroup)
	})
}

func TestGeneratorRingReader(t *testing.T) {
	partitionRing := &dskitring.PartitionInstanceRing{}
	generatorRingWatcher := &dskitring.PartitionRingWatcher{}

	t.Run("single binary does not need a ring reader", func(t *testing.T) {
		cfg := NewDefaultConfig()
		cfg.Target = SingleBinary
		cfg.Generator.RingMode = generator.RingModeGenerator

		app := &App{cfg: *cfg}
		app.configureGenerator()

		reader, err := app.generatorRingReader()
		require.NoError(t, err)
		assert.Nil(t, reader)
	})

	t.Run("distributed partition mode uses partition ring", func(t *testing.T) {
		cfg := NewDefaultConfig()
		cfg.Target = MetricsGenerator
		cfg.Generator.RingMode = generator.RingModePartition

		app := &App{
			cfg:           *cfg,
			partitionRing: partitionRing,
		}
		app.configureGenerator()

		reader, err := app.generatorRingReader()
		require.NoError(t, err)
		assert.Same(t, partitionRing, reader)
	})

	t.Run("distributed generator mode uses generator ring watcher", func(t *testing.T) {
		cfg := NewDefaultConfig()
		cfg.Target = MetricsGenerator
		cfg.Generator.RingMode = generator.RingModeGenerator

		app := &App{
			cfg:                  *cfg,
			partitionRing:        partitionRing,
			generatorRingWatcher: generatorRingWatcher,
		}
		app.configureGenerator()

		reader, err := app.generatorRingReader()
		require.NoError(t, err)
		assert.Same(t, generatorRingWatcher, reader)
	})
}

func TestInitGeneratorNoLocalBlocks_forcesGeneratorRingMode(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.Target = MetricsGeneratorNoLocalBlocks
	cfg.Generator.RingMode = generator.RingModePartition
	cfg.Generator.Storage.Path = t.TempDir()
	cfg.Ingest.Kafka.Topic = "tempo"
	cfg.Ingest.Kafka.ConsumerGroup = "custom"

	app := &App{
		cfg:                  *cfg,
		generatorRingWatcher: &dskitring.PartitionRingWatcher{},
	}

	svc, err := app.initGeneratorNoLocalBlocks()
	require.NoError(t, err)
	require.NotNil(t, svc)
	assert.Equal(t, generator.RingModeGenerator, app.cfg.Generator.RingMode)
	assert.True(t, app.cfg.Generator.ConsumeFromKafka)
	assert.Equal(t, "custom", app.cfg.Generator.Ingest.Kafka.ConsumerGroup)
}

func TestInitLiveStoreSingleBinaryUsesLocalIngest(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.Target = SingleBinary
	cfg.LiveStore.WAL.Filepath = t.TempDir()
	cfg.LiveStore.ShutdownMarkerDir = t.TempDir()
	cfg.LiveStore.Ring.InstanceID = "single-binary"

	partitionRingStore, partitionRingCloser := consul.NewInMemoryClient(dskitring.GetPartitionRingCodec(), log.NewNopLogger(), nil)
	defer partitionRingCloser.Close()
	readRingStore, readRingCloser := consul.NewInMemoryClient(dskitring.GetCodec(), log.NewNopLogger(), nil)
	defer readRingCloser.Close()
	cfg.LiveStore.PartitionRing.KVStore.Mock = partitionRingStore
	cfg.LiveStore.Ring.KVStore.Mock = readRingStore

	overridesSvc, err := overrides.NewOverrides(cfg.Overrides, nil, prometheus.NewRegistry())
	require.NoError(t, err)

	app := &App{
		cfg:       *cfg,
		Server:    &fakeTempoServer{router: mux.NewRouter(), grpc: grpc.NewServer()},
		Overrides: overridesSvc,
		store:     newTestStore(t, t.TempDir()),
	}

	svc, err := app.initLiveStore()
	require.NoError(t, err)
	require.NotNil(t, svc)
	require.NotNil(t, app.liveStore)
	assert.False(t, app.cfg.LiveStore.ConsumeFromKafka)
}

func TestSetupModuleManagerSingleBinaryUsesLocalDistributorAndGeneratorDeps(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.Target = SingleBinary

	app, err := New(*cfg)
	require.NoError(t, err)

	distributorDeps := app.deps[Distributor]
	assert.Contains(t, distributorDeps, LiveStore)
	assert.Contains(t, distributorDeps, MetricsGenerator)
	assert.NotContains(t, distributorDeps, PartitionRing)
	assert.NotContains(t, distributorDeps, LiveStoreRing)

	generatorDeps := app.deps[MetricsGenerator]
	assert.Contains(t, generatorDeps, "common")
	assert.NotContains(t, generatorDeps, MemberlistKV)
	assert.NotContains(t, generatorDeps, PartitionRing)
	assert.NotContains(t, generatorDeps, GeneratorRingWatcher)
}

func TestSetupModuleManagerDistributedUsesKafkaDistributorAndGeneratorDeps(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.Target = Distributor

	app, err := New(*cfg)
	require.NoError(t, err)

	distributorDeps := app.deps[Distributor]
	assert.Contains(t, distributorDeps, PartitionRing)
	assert.Contains(t, distributorDeps, LiveStoreRing)
	assert.NotContains(t, distributorDeps, LiveStore)
	assert.NotContains(t, distributorDeps, MetricsGenerator)

	generatorDeps := app.deps[MetricsGenerator]
	assert.Contains(t, generatorDeps, MemberlistKV)
	assert.Contains(t, generatorDeps, PartitionRing)
	assert.Contains(t, generatorDeps, GeneratorRingWatcher)
}
