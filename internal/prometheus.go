package jenkinsexporter

import "github.com/prometheus/client_golang/prometheus"

const (
	BuildStageMetricName  = "stage_duration_seconds"
	JobDurationMetricName = "job_duration_seconds"
)

type Metrics struct {
	BuildStage  *prometheus.HistogramVec
	JobDuration *prometheus.HistogramVec
	Errors      *prometheus.CounterVec
}

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
