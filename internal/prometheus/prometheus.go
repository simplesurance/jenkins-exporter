package prometheus

// The methods in this package are not concurrency-safe.

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

type Collector struct {
	namespace   string
	summaries   map[string]*prometheus.SummaryVec
	histograms  map[string]*prometheus.HistogramVec
	counters    map[string]*prometheus.CounterVec
	constLabels prometheus.Labels
}

func NewCollector(namespace string, constLabels map[string]string) *Collector {
	return &Collector{
		namespace:   namespace,
		summaries:   map[string]*prometheus.SummaryVec{},
		histograms:  map[string]*prometheus.HistogramVec{},
		counters:    map[string]*prometheus.CounterVec{},
		constLabels: constLabels,
	}
}

func (c *Collector) CounterAdd(key string, cnt float64, help string, labels map[string]string) {
	s, exist := c.counters[key]
	if !exist {
		s = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace:   c.namespace,
				Name:        sanitize(key),
				ConstLabels: c.constLabels,
				Help:        help,
			},
			labelNames(labels),
		)

		prometheus.MustRegister(s)

		c.counters[key] = s
	}

	s.With(labels).Add(cnt)
}

func (c *Collector) Summary(key string, val float64, help string, labels map[string]string) {
	s, exist := c.summaries[key]
	if !exist {
		s = prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Namespace:   c.namespace,
				Name:        sanitize(key),
				ConstLabels: c.constLabels,
				Help:        help,
			},
			labelNames(labels),
		)

		prometheus.MustRegister(s)

		c.summaries[key] = s
	}

	s.With(labels).Observe(val)
}

func (c *Collector) Histogram(key string, val float64, help string, buckets []float64, labels map[string]string) {
	h, exist := c.histograms[key]
	if !exist {
		h = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace:   c.namespace,
				Name:        sanitize(key),
				ConstLabels: c.constLabels,
				Buckets:     buckets,
				Help:        help,
			},
			labelNames(labels),
		)

		prometheus.MustRegister(h)

		c.histograms[key] = h
	}

	h.With(labels).Observe(val)
}

func labelNames(lbs prometheus.Labels) []string {
	names := make([]string, 0, len(lbs))
	for k := range lbs {
		names = append(names, k)
	}

	return names
}

func sanitize(key string) string {
	rep := strings.NewReplacer(
		" ", "_",
		"-", "_",
		"/", "_",
		"\\", "_",
		".", "_",
		",", "_",
	)
	return rep.Replace(strings.ToLower(strings.TrimSpace(key)))
}
