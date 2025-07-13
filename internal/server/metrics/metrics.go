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
}, []string{"request_type"})

// var RequestsInProgress = promauto.NewCounterVec(prometheus.CounterOpts{
// 	Namespace: namespace,
// 	Name:      "requests_in_progress_total",
// 	Help:      "The total number of in-progress requests",
// }, []string{"request_type"})

var RequestProcessingDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Namespace:                   namespace,
	Name:                        "request_processing_duration_seconds",
	Help:                        "The duration in seconds it took to process a request",
	NativeHistogramBucketFactor: 1.00271,
}, []string{"request_type"})

// var RequestProcessingDuration = promauto.NewSummaryVec(prometheus.SummaryOpts{
// 	Namespace: namespace,
// 	Name:      "request_processing_duration_seconds",
// 	Help:      "The duration in seconds it took to process a request",
// 	Objectives: map[float64]float64{
// 		0.01: 0.001,
// 		0.50: 0.005,
// 		0.90: 0.009,
// 		0.95: 0.0095,
// 		0.99: 0.0099,
// 	},
// 	MaxAge: 1 * time.Hour,
// }, []string{"request_type"})

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
