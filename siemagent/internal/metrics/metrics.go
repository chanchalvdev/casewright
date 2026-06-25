package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	EventsClassifiedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "events_classified_total",
			Help: "Total number of classified events by severity and attack type.",
		},
		[]string{"severity", "attack_type"},
	)

	ClassificationDurationSeconds = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "classification_duration_seconds",
			Help:    "Duration of log event classification in seconds.",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10},
		},
	)

	LLMStreamErrorsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "llm_stream_errors_total",
			Help: "Total number of errors from the LLM streaming API.",
		},
	)
)
