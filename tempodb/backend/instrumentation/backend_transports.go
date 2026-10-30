package instrumentation

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var requestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Namespace:                       "tempodb",
	Name:                            "backend_request_duration_seconds",
	Help:                            "Time spent doing backend storage requests.",
	Buckets:                         prometheus.ExponentialBuckets(0.005, 4, 6),
	NativeHistogramBucketFactor:     1.1,
	NativeHistogramMaxBucketNumber:  100,
	NativeHistogramMinResetDuration: 1 * time.Hour,
}, []string{"operation", "status_code"})

var connectionsDialed = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "tempodb",
	Name:      "backend_connections_dialed_total",
	Help:      "Total number of new connections dialed to the storage backend.",
}, []string{"backend"})

// CountDials counts every connection the transport dials successfully, which shows how often
// requests can't reuse an idle connection.
func CountDials(t *http.Transport, backend string) {
	dial := t.DialContext
	if dial == nil {
		dial = (&net.Dialer{}).DialContext
	}
	counter := connectionsDialed.WithLabelValues(backend)
	t.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		conn, err := dial(ctx, network, addr)
		if err == nil {
			counter.Inc()
		}
		return conn, err
	}
}

type instrumentedTransport struct {
	observer prometheus.ObserverVec
	next     http.RoundTripper
}

func NewTransport(next http.RoundTripper) http.RoundTripper {
	return instrumentedTransport{
		next:     next,
		observer: requestDuration,
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
