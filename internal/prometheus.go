// Package jenkinsexporter is the root internal package.
package jenkinsexporter

import "github.com/prometheus/client_golang/prometheus"

const (
	// BuildStageMetricName is the name of the build stage duration metric.
	BuildStageMetricName = "stage_duration_seconds"
	// JobDurationMetricName is the name of the job duration metric.
	JobDurationMetricName = "job_duration_seconds"
)

// Metrics holds all prometheus metric vectors.
type Metrics struct {
	BuildStage  *prometheus.HistogramVec
	JobDuration *prometheus.HistogramVec
	Errors      *prometheus.CounterVec
}

// MustNewMetrics initializes and registers all prometheus metrics. Panics on error.
func MustNewMetrics(namespace string, buckets []float64) *Metrics {
	result := Metrics{
		BuildStage: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      BuildStageMetricName,
				Buckets:   buckets,
			},
			[]string{
				"branch",
				"jenkins_job",
				"result",
				"stage",
				"type",
			},
		),
		JobDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      JobDurationMetricName,
				Buckets:   buckets,
			},
			[]string{
				"jenkins_job",
				"type",
				"result",
				"branch",
			},
		),
		Errors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "errors",
				Help:      "jenkins api fetch errors",
			},
			[]string{"type"},
		),
	}

	prometheus.MustRegister(result.BuildStage)
	prometheus.MustRegister(result.JobDuration)
	prometheus.MustRegister(result.Errors)

	return &result
}
