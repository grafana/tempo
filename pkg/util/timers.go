package util

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var metricCompactionSeconds = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "tempodb",
	Name:      "activity_seconds_total",
	Help:      "Total time spend on stuff",
}, []string{"activity"})

var activityStartTime time.Time

func StartActivity() {
	activityStartTime = time.Now()
}

func CompleteActivity(activity string) {
	now := time.Now()
	elasped := now.Sub(activityStartTime)

	metricCompactionSeconds.WithLabelValues(activity).Add(elasped.Seconds())

	activityStartTime = now
}
