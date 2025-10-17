// SPDX-License-Identifier: AGPL-3.0-only

package usagetrackerclient_test

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/flagext"
	"github.com/grafana/dskit/kv/consul"
	"github.com/grafana/dskit/ring"
	ring_client "github.com/grafana/dskit/ring/client"
	"github.com/grafana/dskit/services"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/modules/generator/remoteserieslimiter/usagetrackerclient"
	"github.com/grafana/tempo/modules/generator/remoteserieslimiter/usagetrackerpb"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type Config struct {
	InstanceRing  usagetrackerclient.InstanceRingConfig  `yaml:"instance_ring"`
	PartitionRing usagetrackerclient.PartitionRingConfig `yaml:"partition_ring"`
}

func (cfg *Config) RegisterFlags(f *flag.FlagSet) {
	cfg.InstanceRing.RegisterFlags(f, log.NewNopLogger())
	cfg.PartitionRing.RegisterFlags(f)
}

func TestUsageTrackerClient_TrackSeries(t *testing.T) {
	var (
		ctx    = context.Background()
		logger = log.NewNopLogger()
		userID = "user-1"
	)

	prepareTest := func() (*ring.MultiPartitionInstanceRing, *ring.Ring, prometheus.Registerer) {
		registerer := prometheus.NewPedanticRegistry()

		consulConfig := consul.Config{
			MaxCasRetries: 100,
			CasRetryDelay: 10 * time.Millisecond,
		}

		// Setup the in-memory KV store used for the ring.
		instanceRingStore, instanceRingCloser := consul.NewInMemoryClientWithConfig(ring.GetCodec(), consulConfig, log.NewNopLogger(), nil)
		t.Cleanup(func() { assert.NoError(t, instanceRingCloser.Close()) })

		partitionRingStore, partitionRingCloser := consul.NewInMemoryClientWithConfig(ring.GetPartitionRingCodec(), consulConfig, log.NewNopLogger(), nil)
		t.Cleanup(func() { assert.NoError(t, partitionRingCloser.Close()) })

		// Add few usage-tracker instances to the instance ring.
		require.NoError(t, instanceRingStore.CAS(ctx, usagetrackerclient.InstanceRingKey, func(interface{}) (interface{}, bool, error) {
			d := ring.NewDesc()
			d.AddIngester("usage-tracker-zone-a-1", "1.1.1.1", "zone-a", []uint32{1}, ring.ACTIVE, time.Now(), false, time.Time{})
			d.AddIngester("usage-tracker-zone-a-2", "2.2.2.2", "zone-a", []uint32{2}, ring.ACTIVE, time.Now(), false, time.Time{})
			d.AddIngester("usage-tracker-zone-b-1", "3.3.3.3", "zone-b", []uint32{3}, ring.ACTIVE, time.Now(), false, time.Time{})
			d.AddIngester("usage-tracker-zone-b-2", "4.4.4.4", "zone-b", []uint32{4}, ring.ACTIVE, time.Now(), false, time.Time{})

			return d, true, nil
		}))

		// Add partitions to the partition ring.
		require.NoError(t, partitionRingStore.CAS(ctx, usagetrackerclient.PartitionRingKey, func(interface{}) (interface{}, bool, error) {
			d := ring.NewPartitionRingDesc()
			d.AddPartition(1, ring.PartitionActive, time.Now())
			d.AddPartition(2, ring.PartitionActive, time.Now())
			d.AddOrUpdateOwner("usage-tracker-zone-a-1", ring.OwnerActive, 1, time.Now())
			d.AddOrUpdateOwner("usage-tracker-zone-a-2", ring.OwnerActive, 2, time.Now())
			d.AddOrUpdateOwner("usage-tracker-zone-b-1", ring.OwnerActive, 1, time.Now())
			d.AddOrUpdateOwner("usage-tracker-zone-b-2", ring.OwnerActive, 2, time.Now())

			return d, true, nil
		}))

		serverCfg := createTestServerConfig()
		serverCfg.InstanceRing.KVStore.Mock = instanceRingStore
		serverCfg.PartitionRing.KVStore.Mock = partitionRingStore

		// Create the instance ring.
		instanceRing, err := usagetrackerclient.NewInstanceRingClient(serverCfg.InstanceRing, logger, registerer)
		require.NoError(t, err)
		require.NoError(t, services.StartAndAwaitRunning(ctx, instanceRing))
		t.Cleanup(func() {
			require.NoError(t, services.StopAndAwaitTerminated(ctx, instanceRing))
		})

		// Pre-condition check: all instances should be healthy.
		set, err := instanceRing.GetAllHealthy(usagetrackerclient.TrackSeriesOp)
		require.NoError(t, err)
		require.Len(t, set.Instances, 4)

		// Create the partition ring.
		partitionRingWatcher := usagetrackerclient.NewPartitionRingWatcher(partitionRingStore, logger, registerer)
		require.NoError(t, services.StartAndAwaitRunning(ctx, partitionRingWatcher))
		t.Cleanup(func() {
			require.NoError(t, services.StopAndAwaitTerminated(ctx, partitionRingWatcher))
		})

		partitionRing := ring.NewMultiPartitionInstanceRing(partitionRingWatcher, instanceRing, serverCfg.InstanceRing.HeartbeatTimeout)

		// Pre-condition check: all partitions should be active.
		require.Equal(t, []int32{1, 2}, partitionRingWatcher.PartitionRing().ActivePartitionIDs())

		return partitionRing, instanceRing, registerer
	}

	t.Run("should track series to usage-trackers running in the preferred zone if available (series are sharded to 2 partitions)", func(t *testing.T) {
		t.Parallel()

		partitionRing, instanceRing, registerer := prepareTest()

		// Mock the usage-tracker server.
		instances := map[string]*usageTrackerMock{
			"usage-tracker-zone-a-1": newUsageTrackerMockWithSuccessfulResponse(),
			"usage-tracker-zone-a-2": newUsageTrackerMockWithSuccessfulResponse(),
			"usage-tracker-zone-b-1": newUsageTrackerMockWithSuccessfulResponse(),
			"usage-tracker-zone-b-2": newUsageTrackerMockWithSuccessfulResponse(),
		}

		clientCfg := createTestClientConfig()
		clientCfg.PreferAvailabilityZone = "zone-b"

		clientCfg.ClientFactory = ring_client.PoolInstFunc(func(instance ring.InstanceDesc) (ring_client.PoolClient, error) {
			mock, ok := instances[instance.Id]
			if ok {
				return mock, nil
			}

			return nil, fmt.Errorf("usage-tracker with ID %s not found", instance.Id)
		})

		c := usagetrackerclient.NewUsageTrackerClient("test", clientCfg, partitionRing, instanceRing, logger, registerer)
		require.NoError(t, services.StartAndAwaitRunning(ctx, c))
		t.Cleanup(func() {
			require.NoError(t, services.StopAndAwaitTerminated(ctx, c))
		})

		// Generate the series hashes so that we can predict in which partition they're sharded to.
		partitions := partitionRing.PartitionRing().Partitions()
		require.Len(t, partitions, 2)
		slices.SortFunc(partitions, func(a, b ring.PartitionDesc) int { return int(a.Id - b.Id) })

		require.Equal(t, int32(1), partitions[0].Id)
		require.Equal(t, int32(2), partitions[1].Id)

		series1Partition1 := uint64(partitions[0].Tokens[0] - 1)
		series2Partition1 := uint64(partitions[0].Tokens[1] - 1)
		series3Partition1 := uint64(partitions[0].Tokens[2] - 1)
		series4Partition2 := uint64(partitions[1].Tokens[0] - 1)
		series5Partition2 := uint64(partitions[1].Tokens[1] - 1)

		rejected, err := c.TrackSeries(user.InjectOrgID(ctx, userID), userID, []uint64{series1Partition1, series2Partition1, series3Partition1, series4Partition2, series5Partition2})
		require.NoError(t, err)
		require.Empty(t, rejected)

		// Should have tracked series only to usage-tracker replicas in the preferred zone.
		instances["usage-tracker-zone-a-1"].AssertNumberOfCalls(t, "TrackSeries", 0)
		instances["usage-tracker-zone-a-2"].AssertNumberOfCalls(t, "TrackSeries", 0)
		instances["usage-tracker-zone-b-1"].AssertNumberOfCalls(t, "TrackSeries", 1)
		instances["usage-tracker-zone-b-2"].AssertNumberOfCalls(t, "TrackSeries", 1)

		req := instances["usage-tracker-zone-b-1"].Calls[0].Arguments.Get(1)
		require.ElementsMatch(t, []uint64{series1Partition1, series2Partition1, series3Partition1}, req.(*usagetrackerpb.TrackSeriesRequest).SeriesHashes)
		require.Equal(t, int32(1), req.(*usagetrackerpb.TrackSeriesRequest).Partition)

		req = instances["usage-tracker-zone-b-2"].Calls[0].Arguments.Get(1)
		require.ElementsMatch(t, []uint64{series4Partition2, series5Partition2}, req.(*usagetrackerpb.TrackSeriesRequest).SeriesHashes)
		require.Equal(t, int32(2), req.(*usagetrackerpb.TrackSeriesRequest).Partition)
	})

	t.Run("should track series to usage-trackers running in the preferred zone if available (series are sharded to 1 partition)", func(t *testing.T) {
		t.Parallel()

		partitionRing, instanceRing, registerer := prepareTest()

		// Mock the usage-tracker server.
		instances := map[string]*usageTrackerMock{
			"usage-tracker-zone-a-1": newUsageTrackerMockWithSuccessfulResponse(),
			"usage-tracker-zone-a-2": newUsageTrackerMockWithSuccessfulResponse(),
			"usage-tracker-zone-b-1": newUsageTrackerMockWithSuccessfulResponse(),
			"usage-tracker-zone-b-2": newUsageTrackerMockWithSuccessfulResponse(),
		}

		clientCfg := createTestClientConfig()
		clientCfg.PreferAvailabilityZone = "zone-b"

		clientCfg.ClientFactory = ring_client.PoolInstFunc(func(instance ring.InstanceDesc) (ring_client.PoolClient, error) {
			mock, ok := instances[instance.Id]
			if ok {
				return mock, nil
			}

			return nil, fmt.Errorf("usage-tracker with ID %s not found", instance.Id)
		})

		c := usagetrackerclient.NewUsageTrackerClient("test", clientCfg, partitionRing, instanceRing, logger, registerer)
		require.NoError(t, services.StartAndAwaitRunning(ctx, c))
		t.Cleanup(func() {
			require.NoError(t, services.StopAndAwaitTerminated(ctx, c))
		})

		// Generate the series hashes so that we can predict in which partition they're sharded to.
		partitions := partitionRing.PartitionRing().Partitions()
		require.Len(t, partitions, 2)
		slices.SortFunc(partitions, func(a, b ring.PartitionDesc) int { return int(a.Id - b.Id) })

		require.Equal(t, int32(1), partitions[0].Id)
		require.Equal(t, int32(2), partitions[1].Id)

		series1Partition1 := uint64(partitions[0].Tokens[0] - 1)
		series2Partition1 := uint64(partitions[0].Tokens[1] - 1)
		series3Partition1 := uint64(partitions[0].Tokens[2] - 1)

		rejected, err := c.TrackSeries(user.InjectOrgID(ctx, userID), userID, []uint64{series1Partition1, series2Partition1, series3Partition1})
		require.NoError(t, err)
		require.Empty(t, rejected)

		// Should have tracked series only to usage-tracker replicas in the preferred zone.
		instances["usage-tracker-zone-a-1"].AssertNumberOfCalls(t, "TrackSeries", 0)
		instances["usage-tracker-zone-a-2"].AssertNumberOfCalls(t, "TrackSeries", 0)
		instances["usage-tracker-zone-b-1"].AssertNumberOfCalls(t, "TrackSeries", 1)
		instances["usage-tracker-zone-b-2"].AssertNumberOfCalls(t, "TrackSeries", 0)

		req := instances["usage-tracker-zone-b-1"].Calls[0].Arguments.Get(1)
		require.ElementsMatch(t, []uint64{series1Partition1, series2Partition1, series3Partition1}, req.(*usagetrackerpb.TrackSeriesRequest).SeriesHashes)
		require.Equal(t, int32(1), req.(*usagetrackerpb.TrackSeriesRequest).Partition)
	})

	t.Run("should fallback to the other zone if a usage-tracker instance in the preferred zone is failing", func(t *testing.T) {
		t.Parallel()

		partitionRing, instanceRing, registerer := prepareTest()

		// Mock the usage-tracker server.
		instances := map[string]*usageTrackerMock{
			"usage-tracker-zone-a-1": newUsageTrackerMockWithSuccessfulResponse(),
			"usage-tracker-zone-a-2": newUsageTrackerMockWithSuccessfulResponse(),
			"usage-tracker-zone-b-1": newUsageTrackerMockWithResponse(nil, errors.New("failing instance")),
			"usage-tracker-zone-b-2": newUsageTrackerMockWithSuccessfulResponse(),
		}

		clientCfg := createTestClientConfig()
		clientCfg.PreferAvailabilityZone = "zone-b"

		clientCfg.ClientFactory = ring_client.PoolInstFunc(func(instance ring.InstanceDesc) (ring_client.PoolClient, error) {
			mock, ok := instances[instance.Id]
			if ok {
				return mock, nil
			}

			return nil, fmt.Errorf("usage-tracker with ID %s not found", instance.Id)
		})

		c := usagetrackerclient.NewUsageTrackerClient("test", clientCfg, partitionRing, instanceRing, logger, registerer)
		require.NoError(t, services.StartAndAwaitRunning(ctx, c))
		t.Cleanup(func() {
			require.NoError(t, services.StopAndAwaitTerminated(ctx, c))
		})

		// Generate the series hashes so that we can predict in which partition they're sharded to.
		partitions := partitionRing.PartitionRing().Partitions()
		require.Len(t, partitions, 2)
		slices.SortFunc(partitions, func(a, b ring.PartitionDesc) int { return int(a.Id - b.Id) })

		require.Equal(t, int32(1), partitions[0].Id)
		require.Equal(t, int32(2), partitions[1].Id)

		series1Partition1 := uint64(partitions[0].Tokens[0] - 1)
		series2Partition1 := uint64(partitions[0].Tokens[1] - 1)
		series3Partition1 := uint64(partitions[0].Tokens[2] - 1)
		series4Partition2 := uint64(partitions[1].Tokens[0] - 1)
		series5Partition2 := uint64(partitions[1].Tokens[1] - 1)

		rejected, err := c.TrackSeries(user.InjectOrgID(ctx, userID), userID, []uint64{series1Partition1, series2Partition1, series3Partition1, series4Partition2, series5Partition2})
		require.NoError(t, err)
		require.Empty(t, rejected)

		// Should have tracked series only to usage-tracker replicas in the preferred zone.
		instances["usage-tracker-zone-a-1"].AssertNumberOfCalls(t, "TrackSeries", 1)
		instances["usage-tracker-zone-a-2"].AssertNumberOfCalls(t, "TrackSeries", 0)
		instances["usage-tracker-zone-b-1"].AssertNumberOfCalls(t, "TrackSeries", 1)
		instances["usage-tracker-zone-b-2"].AssertNumberOfCalls(t, "TrackSeries", 1)

		req := instances["usage-tracker-zone-b-1"].Calls[0].Arguments.Get(1)
		require.ElementsMatch(t, []uint64{series1Partition1, series2Partition1, series3Partition1}, req.(*usagetrackerpb.TrackSeriesRequest).SeriesHashes)
		require.Equal(t, int32(1), req.(*usagetrackerpb.TrackSeriesRequest).Partition)

		req = instances["usage-tracker-zone-b-2"].Calls[0].Arguments.Get(1)
		require.ElementsMatch(t, []uint64{series4Partition2, series5Partition2}, req.(*usagetrackerpb.TrackSeriesRequest).SeriesHashes)
		require.Equal(t, int32(2), req.(*usagetrackerpb.TrackSeriesRequest).Partition)

		// Fallback.
		req = instances["usage-tracker-zone-a-1"].Calls[0].Arguments.Get(1)
		require.ElementsMatch(t, []uint64{series1Partition1, series2Partition1, series3Partition1}, req.(*usagetrackerpb.TrackSeriesRequest).SeriesHashes)
		require.Equal(t, int32(1), req.(*usagetrackerpb.TrackSeriesRequest).Partition)
	})

	for _, returnRejectedSeries := range []bool{true, false} {
		testName := map[bool]string{
			true:  "should return rejected series",
			false: "should not return rejected series",
		}[returnRejectedSeries]

		t.Run(testName, func(t *testing.T) {
			t.Parallel()

			partitionRing, instanceRing, registerer := prepareTest()

			// Generate the series hashes so that we can predict in which partition they're sharded to.
			partitions := partitionRing.PartitionRing().Partitions()
			require.Len(t, partitions, 2)
			slices.SortFunc(partitions, func(a, b ring.PartitionDesc) int { return int(a.Id - b.Id) })

			require.Equal(t, int32(1), partitions[0].Id)
			require.Equal(t, int32(2), partitions[1].Id)

			series1Partition1 := uint64(partitions[0].Tokens[0] - 1)
			series2Partition1 := uint64(partitions[0].Tokens[1] - 1)
			series3Partition1 := uint64(partitions[0].Tokens[2] - 1)
			series4Partition2 := uint64(partitions[1].Tokens[0] - 1)
			series5Partition2 := uint64(partitions[1].Tokens[1] - 1)

			// Mock the usage-tracker server.
			instances := map[string]*usageTrackerMock{
				"usage-tracker-zone-a-1": newUsageTrackerMockWithSuccessfulResponse(),
				"usage-tracker-zone-a-2": newUsageTrackerMockWithSuccessfulResponse(),

				// Return rejected series only from zone-b to ensure the response is picked up from this zone.
				"usage-tracker-zone-b-1": newUsageTrackerMockWithResponse(&usagetrackerpb.TrackSeriesResponse{RejectedSeriesHashes: []uint64{series2Partition1}}, nil),
				"usage-tracker-zone-b-2": newUsageTrackerMockWithResponse(&usagetrackerpb.TrackSeriesResponse{RejectedSeriesHashes: []uint64{series4Partition2, series5Partition2}}, nil),
			}

			clientCfg := createTestClientConfig()
			clientCfg.IgnoreRejectedSeries = !returnRejectedSeries
			clientCfg.PreferAvailabilityZone = "zone-b"

			clientCfg.ClientFactory = ring_client.PoolInstFunc(func(instance ring.InstanceDesc) (ring_client.PoolClient, error) {
				mock, ok := instances[instance.Id]
				if ok {
					return mock, nil
				}

				return nil, fmt.Errorf("usage-tracker with ID %s not found", instance.Id)
			})

			c := usagetrackerclient.NewUsageTrackerClient("test", clientCfg, partitionRing, instanceRing, logger, registerer)
			require.NoError(t, services.StartAndAwaitRunning(ctx, c))
			t.Cleanup(func() {
				require.NoError(t, services.StopAndAwaitTerminated(ctx, c))
			})

			rejected, err := c.TrackSeries(user.InjectOrgID(ctx, userID), userID, []uint64{series1Partition1, series2Partition1, series3Partition1, series4Partition2, series5Partition2})
			require.NoError(t, err)
			if returnRejectedSeries {
				require.ElementsMatch(t, []uint64{series2Partition1, series4Partition2, series5Partition2}, rejected)
			} else {
				require.Empty(t, rejected)
			}

			// Should have tracked series only to usage-tracker replicas in the preferred zone.
			instances["usage-tracker-zone-a-1"].AssertNumberOfCalls(t, "TrackSeries", 0)
			instances["usage-tracker-zone-a-2"].AssertNumberOfCalls(t, "TrackSeries", 0)
			instances["usage-tracker-zone-b-1"].AssertNumberOfCalls(t, "TrackSeries", 1)
			instances["usage-tracker-zone-b-2"].AssertNumberOfCalls(t, "TrackSeries", 1)

			req := instances["usage-tracker-zone-b-1"].Calls[0].Arguments.Get(1)
			require.ElementsMatch(t, []uint64{series1Partition1, series2Partition1, series3Partition1}, req.(*usagetrackerpb.TrackSeriesRequest).SeriesHashes)
			require.Equal(t, int32(1), req.(*usagetrackerpb.TrackSeriesRequest).Partition)

			req = instances["usage-tracker-zone-b-2"].Calls[0].Arguments.Get(1)
			require.ElementsMatch(t, []uint64{series4Partition2, series5Partition2}, req.(*usagetrackerpb.TrackSeriesRequest).SeriesHashes)
			require.Equal(t, int32(2), req.(*usagetrackerpb.TrackSeriesRequest).Partition)
		})
	}

	t.Run("should hedge requests to the other zone if a usage-tracker instance in the preferred zone is slow", func(t *testing.T) {
		t.Parallel()

		partitionRing, instanceRing, registerer := prepareTest()

		clientCfg := createTestClientConfig()
		clientCfg.PreferAvailabilityZone = "zone-b"
		clientCfg.RequestsHedgingDelay = 250 * time.Millisecond

		// Generate the series hashes so that we can predict in which partition they're sharded to.
		partitions := partitionRing.PartitionRing().Partitions()
		require.Len(t, partitions, 2)
		slices.SortFunc(partitions, func(a, b ring.PartitionDesc) int { return int(a.Id - b.Id) })

		require.Equal(t, int32(1), partitions[0].Id)
		require.Equal(t, int32(2), partitions[1].Id)

		series1Partition1 := uint64(partitions[0].Tokens[0] - 1)
		series2Partition1 := uint64(partitions[0].Tokens[1] - 1)
		series3Partition1 := uint64(partitions[0].Tokens[2] - 1)
		series4Partition2 := uint64(partitions[1].Tokens[0] - 1)
		series5Partition2 := uint64(partitions[1].Tokens[1] - 1)

		// Mock the usage-tracker server.
		instances := map[string]*usageTrackerMock{
			// Return rejected series only from this instance, to ensure the response comes from here.
			"usage-tracker-zone-a-1": newUsageTrackerMockWithResponse(&usagetrackerpb.TrackSeriesResponse{RejectedSeriesHashes: []uint64{series2Partition1}}, nil),

			"usage-tracker-zone-a-2": newUsageTrackerMockWithSuccessfulResponse(),
			"usage-tracker-zone-b-1": newUsageTrackerMockWithSlowSuccessfulResponse(clientCfg.RequestsHedgingDelay * 2),
			"usage-tracker-zone-b-2": newUsageTrackerMockWithSuccessfulResponse(),
		}

		clientCfg.ClientFactory = ring_client.PoolInstFunc(func(instance ring.InstanceDesc) (ring_client.PoolClient, error) {
			mock, ok := instances[instance.Id]
			if ok {
				return mock, nil
			}

			return nil, fmt.Errorf("usage-tracker with ID %s not found", instance.Id)
		})

		c := usagetrackerclient.NewUsageTrackerClient("test", clientCfg, partitionRing, instanceRing, logger, registerer)
		require.NoError(t, services.StartAndAwaitRunning(ctx, c))
		t.Cleanup(func() {
			require.NoError(t, services.StopAndAwaitTerminated(ctx, c))
		})

		rejected, err := c.TrackSeries(user.InjectOrgID(ctx, userID), userID, []uint64{series1Partition1, series2Partition1, series3Partition1, series4Partition2, series5Partition2})
		require.NoError(t, err)
		require.ElementsMatch(t, []uint64{series2Partition1}, rejected)

		// Should have tracked series only to usage-tracker replicas in the preferred zone.
		instances["usage-tracker-zone-a-1"].AssertNumberOfCalls(t, "TrackSeries", 1)
		instances["usage-tracker-zone-a-2"].AssertNumberOfCalls(t, "TrackSeries", 0)
		instances["usage-tracker-zone-b-1"].AssertNumberOfCalls(t, "TrackSeries", 1)
		instances["usage-tracker-zone-b-2"].AssertNumberOfCalls(t, "TrackSeries", 1)

		req := instances["usage-tracker-zone-b-1"].Calls[0].Arguments.Get(1)
		require.ElementsMatch(t, []uint64{series1Partition1, series2Partition1, series3Partition1}, req.(*usagetrackerpb.TrackSeriesRequest).SeriesHashes)
		require.Equal(t, int32(1), req.(*usagetrackerpb.TrackSeriesRequest).Partition)

		req = instances["usage-tracker-zone-b-2"].Calls[0].Arguments.Get(1)
		require.ElementsMatch(t, []uint64{series4Partition2, series5Partition2}, req.(*usagetrackerpb.TrackSeriesRequest).SeriesHashes)
		require.Equal(t, int32(2), req.(*usagetrackerpb.TrackSeriesRequest).Partition)

		// Hedged request.
		req = instances["usage-tracker-zone-b-1"].Calls[0].Arguments.Get(1)
		require.ElementsMatch(t, []uint64{series1Partition1, series2Partition1, series3Partition1}, req.(*usagetrackerpb.TrackSeriesRequest).SeriesHashes)
		require.Equal(t, int32(1), req.(*usagetrackerpb.TrackSeriesRequest).Partition)
	})

	t.Run("should be a no-op if there are no series to track", func(t *testing.T) {
		t.Parallel()

		partitionRing, instanceRing, registerer := prepareTest()

		clientCfg := createTestClientConfig()
		clientCfg.PreferAvailabilityZone = "zone-b"
		clientCfg.RequestsHedgingDelay = 250 * time.Millisecond

		// Mock the usage-tracker server.
		instances := map[string]*usageTrackerMock{
			"usage-tracker-zone-a-1": newUsageTrackerMockWithSuccessfulResponse(),
			"usage-tracker-zone-a-2": newUsageTrackerMockWithSuccessfulResponse(),
			"usage-tracker-zone-b-1": newUsageTrackerMockWithSuccessfulResponse(),
			"usage-tracker-zone-b-2": newUsageTrackerMockWithSuccessfulResponse(),
		}

		clientCfg.ClientFactory = ring_client.PoolInstFunc(func(instance ring.InstanceDesc) (ring_client.PoolClient, error) {
			mock, ok := instances[instance.Id]
			if ok {
				return mock, nil
			}

			return nil, fmt.Errorf("usage-tracker with ID %s not found", instance.Id)
		})

		c := usagetrackerclient.NewUsageTrackerClient("test", clientCfg, partitionRing, instanceRing, logger, registerer)
		require.NoError(t, services.StartAndAwaitRunning(ctx, c))
		t.Cleanup(func() {
			require.NoError(t, services.StopAndAwaitTerminated(ctx, c))
		})

		rejected, err := c.TrackSeries(user.InjectOrgID(ctx, userID), userID, []uint64{})
		require.NoError(t, err)
		require.Empty(t, rejected)
	})

	t.Run("should ignore errors when IgnoreErrors is enabled", func(t *testing.T) {
		t.Parallel()

		partitionRing, instanceRing, registerer := prepareTest()

		// Mock the usage-tracker server with failing instances.
		instances := map[string]*usageTrackerMock{
			"usage-tracker-zone-a-1": newUsageTrackerMockWithResponse(nil, errors.New("failing instance")),
			"usage-tracker-zone-a-2": newUsageTrackerMockWithResponse(nil, errors.New("failing instance")),
			"usage-tracker-zone-b-1": newUsageTrackerMockWithResponse(nil, errors.New("failing instance")),
			"usage-tracker-zone-b-2": newUsageTrackerMockWithResponse(nil, errors.New("failing instance")),
		}

		clientCfg := createTestClientConfig()
		clientCfg.IgnoreErrors = true
		clientCfg.PreferAvailabilityZone = "zone-b"

		clientCfg.ClientFactory = ring_client.PoolInstFunc(func(instance ring.InstanceDesc) (ring_client.PoolClient, error) {
			mock, ok := instances[instance.Id]
			if ok {
				return mock, nil
			}

			return nil, fmt.Errorf("usage-tracker with ID %s not found", instance.Id)
		})

		c := usagetrackerclient.NewUsageTrackerClient("test", clientCfg, partitionRing, instanceRing, logger, registerer)
		require.NoError(t, services.StartAndAwaitRunning(ctx, c))
		t.Cleanup(func() {
			require.NoError(t, services.StopAndAwaitTerminated(ctx, c))
		})

		// Generate the series hashes so that we can predict in which partition they're sharded to.
		partitions := partitionRing.PartitionRing().Partitions()
		require.Len(t, partitions, 2)
		slices.SortFunc(partitions, func(a, b ring.PartitionDesc) int { return int(a.Id - b.Id) })

		require.Equal(t, int32(1), partitions[0].Id)
		require.Equal(t, int32(2), partitions[1].Id)

		series1Partition1 := uint64(partitions[0].Tokens[0] - 1)
		series2Partition1 := uint64(partitions[0].Tokens[1] - 1)
		series3Partition1 := uint64(partitions[0].Tokens[2] - 1)
		series4Partition2 := uint64(partitions[1].Tokens[0] - 1)
		series5Partition2 := uint64(partitions[1].Tokens[1] - 1)

		// Despite all instances failing, the client should not return an error when IgnoreErrors is enabled.
		rejected, err := c.TrackSeries(user.InjectOrgID(ctx, userID), userID, []uint64{series1Partition1, series2Partition1, series3Partition1, series4Partition2, series5Partition2})
		require.NoError(t, err)
		require.Empty(t, rejected)

		// All instances should have been called despite failing.
		instances["usage-tracker-zone-b-1"].AssertNumberOfCalls(t, "TrackSeries", 1)
		instances["usage-tracker-zone-b-2"].AssertNumberOfCalls(t, "TrackSeries", 1)
	})
}

func createTestClientConfig() usagetrackerclient.Config {
	cfg := usagetrackerclient.Config{}
	flagext.DefaultValues(&cfg)

	// No hedging in tests by default.
	cfg.RequestsHedgingDelay = time.Hour

	return cfg
}

func createTestServerConfig() Config {
	cfg := Config{}
	flagext.DefaultValues(&cfg)

	return cfg
}

type usageTrackerMock struct {
	mock.Mock

	usagetrackerpb.UsageTrackerClient
	grpc_health_v1.HealthClient
}

func newUsageTrackerMockWithSuccessfulResponse() *usageTrackerMock {
	return newUsageTrackerMockWithResponse(&usagetrackerpb.TrackSeriesResponse{}, nil)
}

func newUsageTrackerMockWithSlowSuccessfulResponse(delay time.Duration) *usageTrackerMock {
	m := &usageTrackerMock{}
	m.On("TrackSeries", mock.Anything, mock.Anything).Run(func(_ mock.Arguments) {
		time.Sleep(delay)
	}).Return(&usagetrackerpb.TrackSeriesResponse{}, nil)

	return m
}

func newUsageTrackerMockWithResponse(res *usagetrackerpb.TrackSeriesResponse, err error) *usageTrackerMock {
	m := &usageTrackerMock{}
	m.On("TrackSeries", mock.Anything, mock.Anything).Return(res, err)

	return m
}

func (m *usageTrackerMock) TrackSeries(ctx context.Context, req *usagetrackerpb.TrackSeriesRequest, _ ...grpc.CallOption) (*usagetrackerpb.TrackSeriesResponse, error) {
	args := m.Called(ctx, req)

	if args.Get(0) != nil {
		return args.Get(0).(*usagetrackerpb.TrackSeriesResponse), args.Error(1)
	}

	return nil, args.Error(1)
}

func (m *usageTrackerMock) Close() error {
	return nil
}
