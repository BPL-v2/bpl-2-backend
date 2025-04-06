package repository

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var queryDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name: "sql_query_duration_seconds",
	Help: "Duration of sql queries in seconds",
}, []string{"query"})
