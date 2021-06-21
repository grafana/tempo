package gcs

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
		Help:      "Time spent doing GCS requests.",

		// We often write large blocks to GCS, so use buckets from 5ms to 80s.
		Buckets: prometheus.ExponentialBuckets(0.005, 4, 8),
	}, []string{"operation", "status_code"})
)

type instrumentedTransport struct {
	observer prometheus.ObserverVec
	next     http.RoundTripper
}

func newInstrumentedTransport(next http.RoundTripper) http.RoundTripper {
	return instrumentedTransport{
		observer: gcsRequestDuration,
		next:     next,
	}
}

func (i instrumentedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	resp, err := i.next.RoundTrip(req)
	var status string
	if err == nil {
		status = strconv.Itoa(resp.StatusCode)
	} else {
		status = "500"
	}
	i.observer.WithLabelValues(req.Method, status).Observe(time.Since(start).Seconds())
	return resp, err
}
