package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const namespace = "eko"

var RequestsProcessed = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: namespace,
	Name:      "requests_processed_total",
	Help:      "The total number of processed requests",
}, []string{"request_type", "dropped"})

var RequestsInProgress = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Namespace: namespace,
	Name:      "requests_in_progress_total",
	Help:      "The total number of in-progress requests",
}, []string{"request_type"})

var RequestProcessingDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Namespace:                   namespace,
	Name:                        "request_processing_duration_seconds",
	Help:                        "The duration in seconds it took to process a request",
	NativeHistogramBucketFactor: 1.00271,
}, []string{"request_type", "dropped"})

var ConnectionsRateLimited = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: namespace,
	Name:      "connections_rate_limited_total",
	Help:      "The total number of rate limited connections",
}, []string{"category"})

var ConnectionsEstablished = promauto.NewCounter(prometheus.CounterOpts{
	Namespace: namespace,
	Name:      "connections_established_total",
	Help:      "The total number of established connections",
})

var ConnectionsClosed = promauto.NewCounter(prometheus.CounterOpts{
	Namespace: namespace,
	Name:      "connections_closed_total",
	Help:      "The total number of closed connections",
})

var ConnectionsActive = promauto.NewGauge(prometheus.GaugeOpts{
	Namespace: namespace,
	Name:      "connections_active_total",
	Help:      "The total number of active connections",
})

var UsersActive = promauto.NewGauge(prometheus.GaugeOpts{
	Namespace: namespace,
	Name:      "users_active_total",
	Help:      "The total number of active users",
})

var SessionDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Namespace:                   namespace,
	Name:                        "session_duration_seconds",
	Help:                        "The duration in seconds of an authenticated session",
	NativeHistogramBucketFactor: 1.00271,
}, []string{"os", "arch", "term", "colorterm"})
