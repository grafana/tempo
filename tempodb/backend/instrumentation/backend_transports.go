package instrumentation

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	gcsRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "tempodb",
		Name:      "gcs_request_duration_seconds",
		Help:      "Time spent doing GCS requests. (DEPRECATED: See tempodb_backend_request_duration_seconds)",
		Buckets:   prometheus.ExponentialBuckets(0.005, 4, 6),
	}, []string{"operation", "status_code"})

	s3RequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "tempodb",
		Name:      "s3_request_duration_seconds",
		Help:      "Time spent doing AWS S3 requests. (DEPRECATED: See tempodb_backend_request_duration_seconds)",
		Buckets:   prometheus.ExponentialBuckets(0.005, 4, 6),
	}, []string{"operation", "status_code"})

	azureRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "tempodb",
		Name:      "azure_request_duration_seconds",
		Help:      "Time spent doing Azure requests. (DEPRECATED: See tempodb_backend_request_duration_seconds)",
		Buckets:   prometheus.ExponentialBuckets(0.005, 4, 6),
	}, []string{"operation", "status_code"})

	requestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "tempodb",
		Name:      "backend_request_duration_seconds",
		Help:      "Time spent doing backend storage requests.",
		Buckets:   prometheus.ExponentialBuckets(0.005, 4, 6),
	}, []string{"operation", "status_code"})
)

type instrumentedTransport struct {
	legacyObserver prometheus.ObserverVec
	observer       prometheus.ObserverVec
	next           http.RoundTripper
}

func NewGCSTransport(next http.RoundTripper) http.RoundTripper {
	return instrumentedTransport{
		next:           next,
		observer:       requestDuration,
		legacyObserver: gcsRequestDuration,
	}
}

func NewS3Transport(next http.RoundTripper) http.RoundTripper {
	return instrumentedTransport{
		next:           next,
		observer:       requestDuration,
		legacyObserver: s3RequestDuration,
	}
}

func NewAzureTransport(next http.RoundTripper) http.RoundTripper {
	return instrumentedTransport{
		next:           next,
		observer:       requestDuration,
		legacyObserver: azureRequestDuration,
	}
}

func (i instrumentedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	resp, err := i.next.RoundTrip(req)
	if err == nil {
		i.legacyObserver.WithLabelValues(req.Method, strconv.Itoa(resp.StatusCode)).Observe(time.Since(start).Seconds())
		i.observer.WithLabelValues(req.Method, strconv.Itoa(resp.StatusCode)).Observe(time.Since(start).Seconds())
	}
	return resp, err
}
