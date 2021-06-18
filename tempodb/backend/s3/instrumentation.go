package s3

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	s3RequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "tempodb",
		Name:      "s3_request_duration_seconds",
		Help:      "Time spent doing AWS S3 requests.",

		Buckets: prometheus.ExponentialBuckets(0.005, 4, 6),
	}, []string{"operation", "status_code"})
)

type instrumentedTransport struct {
	observer prometheus.ObserverVec
	next     http.RoundTripper
}

func newInstrumentedTransport(next http.RoundTripper) http.RoundTripper {
	return instrumentedTransport{
		observer: s3RequestDuration,
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
