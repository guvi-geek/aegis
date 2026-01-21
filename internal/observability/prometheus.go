package observability

/*
// Prometheus setup (commented out, ready for future use)

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

var (
	// RequestCount counts HTTP requests
	RequestCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	// RequestDuration measures HTTP request duration
	RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_request_duration_seconds",
			Help: "HTTP request duration in seconds",
		},
		[]string{"method", "endpoint"},
	)

	// ComputationCount counts plagiarism computations
	ComputationCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plagiarism_computations_total",
			Help: "Total number of plagiarism computations",
		},
		[]string{"status"},
	)

	// ComputationDuration measures computation duration
	ComputationDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name: "plagiarism_computation_duration_seconds",
			Help: "Plagiarism computation duration in seconds",
		},
	)
)

// InitPrometheus initializes Prometheus metrics
func InitPrometheus() {
	prometheus.MustRegister(RequestCount)
	prometheus.MustRegister(RequestDuration)
	prometheus.MustRegister(ComputationCount)
	prometheus.MustRegister(ComputationDuration)
}

// MetricsHandler returns Prometheus metrics handler
func MetricsHandler() http.Handler {
	return promhttp.Handler()
}
*/
