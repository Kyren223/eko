// Eko: A terminal based social media platform
// Copyright (C) 2025 Kyren223
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

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

const (
	Minute = 60
	Hour   = 60 * Minute
	Day    = 24 * Hour
)

var SessionDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Namespace: namespace,
	Name:      "session_duration_seconds",
	Help:      "The duration in seconds of an authenticated session",
	Buckets: []float64{
		1, 30, 5 * Minute, 10 * Minute, 30 * Minute,
		Hour, 2 * Hour, 3 * Hour, 4 * Hour,
		5 * Hour, 6 * Hour, 7 * Hour, 8 * Hour,
		10 * Hour, 12 * Hour, 14 * Hour, 16 * Hour,
		18 * Hour, 20 * Hour, 22 * Hour, Day,
		2 * Day, 7 * Day, 14 * Day, 28 * Day,
	},
	// NativeHistogramBucketFactor:     1.00271,
	// NativeHistogramMaxBucketNumber:  100,
	// NativeHistogramMinResetDuration: time.Hour,
}, []string{"os", "arch", "term", "colorterm"})
