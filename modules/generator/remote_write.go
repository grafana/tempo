package generator

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/prometheus/storage/remote"

	"github.com/grafana/tempo/cmd/tempo/build"
	"github.com/grafana/tempo/pkg/util"
)

var UserAgent = fmt.Sprintf("tempo-remote-write/%s", build.Version)

type RemoteWriter interface {
	remote.WriteClient
}

type RemoteWriteClient struct {
	remote.WriteClient
}

func NewRemoteWriter(cfg RemoteWriteConfig, userID string) (RemoteWriter, error) {
	writeClient, err := remote.NewWriteClient(
		"metrics_generator",
		&remote.ClientConfig{
			URL:              cfg.Client.URL,
			Timeout:          cfg.Client.RemoteTimeout,
			HTTPClientConfig: cfg.Client.HTTPClientConfig,
			Headers: util.MergeMaps(cfg.Client.Headers, map[string]string{
				"X-Scope-OrgID": userID,
				"User-Agent":    UserAgent,
			}),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("could not create remote-write client for tenant: %s", userID)
	}

	return &RemoteWriteClient{
		WriteClient: writeClient,
	}, nil
}

type remoteWriteMetrics struct {
	samplesSent       *prometheus.GaugeVec
	remoteWriteErrors *prometheus.CounterVec
	remoteWriteTotal  *prometheus.CounterVec
}

func newRemoteWriteMetrics(reg prometheus.Registerer) *remoteWriteMetrics {
	return &remoteWriteMetrics{
		samplesSent: promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "tempo",
			Name:      "metrics_generator_samples_sent_total",
			Help:      "Number of samples sent per remote write",
		}, []string{"tenant"}),
		remoteWriteErrors: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Namespace: "tempo",
			Name:      "metrics_generator_remote_write_errors",
			Help:      "Number of remote-write requests that failed due to error.",
		}, []string{"tenant"}),
		remoteWriteTotal: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Namespace: "tempo",
			Name:      "metrics_generator_remote_write_total",
			Help:      "Number of remote-write requests.",
		}, []string{"tenant"}),
	}
}
