// SPDX-License-Identifier: AGPL-3.0-only

package usagetrackerclient

import (
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/grpcclient"
	"github.com/grafana/dskit/middleware"
	"github.com/grafana/dskit/ring"
	"github.com/grafana/dskit/ring/client"
	"github.com/grafana/tempo/modules/generator/remoteserieslimiter/usagetrackerpb"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func newUsageTrackerClientPool(discovery client.PoolServiceDiscovery, clientName string, clientConfig Config, logger log.Logger, registerer prometheus.Registerer) *client.Pool {
	poolCfg := client.PoolConfig{
		CheckInterval:      10 * time.Second,
		HealthCheckEnabled: true,
		HealthCheckTimeout: 10 * time.Second,
	}

	clientsCount := promauto.With(registerer).NewGauge(prometheus.GaugeOpts{
		Namespace:   "cortex",
		Name:        "usage_tracker_clients",
		Help:        "The current number of usage-tracker clients in the pool.",
		ConstLabels: map[string]string{"client": clientName},
	})

	var factory client.PoolFactory
	if clientConfig.ClientFactory != nil {
		factory = clientConfig.ClientFactory
	} else {
		factory = newUsageTrackerClientFactory(clientName, clientConfig.GRPCClientConfig, registerer, logger)
	}

	return client.NewPool("usage-tracker", poolCfg, discovery, factory, clientsCount, logger)
}

func newUsageTrackerClientFactory(clientName string, clientCfg grpcclient.Config, reg prometheus.Registerer, logger log.Logger) client.PoolFactory {
	requestDuration := promauto.With(reg).NewHistogramVec(prometheus.HistogramOpts{
		Name:                        "cortex_usage_tracker_client_request_duration_seconds",
		Help:                        "Time spent executing  a single request to a usage-tracker instance.",
		NativeHistogramBucketFactor: 1.1,
		ConstLabels:                 prometheus.Labels{"client": clientName},
	}, []string{"operation", "status_code"})

	return client.PoolInstFunc(func(inst ring.InstanceDesc) (client.PoolClient, error) {
		return dialUsageTracker(clientCfg, inst, requestDuration, logger)
	})
}

func dialUsageTracker(clientCfg grpcclient.Config, instance ring.InstanceDesc, requestDuration *prometheus.HistogramVec, logger log.Logger) (*usageTrackerClient, error) {
	unary, stream := grpcclient.Instrument(requestDuration)
	opts, err := clientCfg.DialOption(unary, stream, middleware.NoOpInvalidClusterValidationReporter)
	if err != nil {
		return nil, err
	}

	// nolint:staticcheck // grpc.Dial() has been deprecated; we'll address it before upgrading to gRPC 2.
	conn, err := grpc.Dial(instance.Addr, opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to dial usage-tracker %s %s", instance.Id, instance.Addr)
	}

	return &usageTrackerClient{
		UsageTrackerClient: usagetrackerpb.NewUsageTrackerClient(conn),
		HealthClient:       grpc_health_v1.NewHealthClient(conn),
		conn:               conn,
	}, nil
}

type usageTrackerClient struct {
	usagetrackerpb.UsageTrackerClient
	grpc_health_v1.HealthClient
	conn *grpc.ClientConn
}

func (c *usageTrackerClient) Close() error {
	return c.conn.Close()
}

func (c *usageTrackerClient) String() string {
	return c.RemoteAddress()
}

func (c *usageTrackerClient) RemoteAddress() string {
	return c.conn.Target()
}
