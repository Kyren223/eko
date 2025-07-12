package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	reg = prometheus.DefaultRegisterer

	namespace = "eko"

	RequestsProcessed = promauto.With(reg).NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "processed_requests_total",
		Help:      "The total number of processed requests",
	})
)
