package app

import (
	"testing"

	dskitring "github.com/grafana/dskit/ring"
	"github.com/grafana/tempo/modules/generator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
