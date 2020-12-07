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

		// GCS latency seems to range from a few ms to a few secs and is
		// important.  So use 6 buckets from 5ms to 5s.
		Buckets: prometheus.ExponentialBuckets(0.005, 4, 6),
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
	if err == nil {
		i.observer.WithLabelValues(req.Method, strconv.Itoa(resp.StatusCode)).Observe(time.Since(start).Seconds())
	}
	return resp, err
}
