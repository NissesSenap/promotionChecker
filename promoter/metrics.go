package promoter

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// NrTagsPromoted metrics
	NrTagsPromoted = promauto.NewCounter(prometheus.CounterOpts{
		Name: "nr_tags_promoted_total",
		Help: "The total number of tags being promoted",
	})
	// HistArtifactory metrics
	HistArtifactory = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "request_from_artifactory",
		Help:    "Historgram get from artifactory",
		Buckets: []float64{0.1, 0.2, 0.5, 1, 2, 5},
	},
		[]string{"repoImage"})

	// HistWebhook metrics
	HistWebhook = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "request_to_webhook",
		Help:    "Historgram post to webhook",
		Buckets: []float64{0.1, 0.2, 0.5, 1, 2, 5},
	},
		[]string{"repoImage"})
)
