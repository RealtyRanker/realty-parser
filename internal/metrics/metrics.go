package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Counters

	PagesFetched = promauto.NewCounter(prometheus.CounterOpts{
		Name: "realty_parser_pages_fetched_total",
		Help: "Total CIAN search pages fetched",
	})
	FlatsFound = promauto.NewCounter(prometheus.CounterOpts{
		Name: "realty_parser_flats_found_total",
		Help: "Total flat listing URLs found on search pages",
	})
	FlatsInserted = promauto.NewCounter(prometheus.CounterOpts{
		Name: "realty_parser_flats_inserted_total",
		Help: "Total flats successfully inserted into the database",
	})
	ParseErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "realty_parser_parse_errors_total",
		Help: "Total flat page parse failures",
	})
	FetchErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "realty_parser_fetch_errors_total",
		Help: "Total HTTP fetch failures",
	})

	// Gauges

	LastRunTimestamp = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "realty_parser_last_run_timestamp_seconds",
		Help: "Unix timestamp of the last worker run start",
	})
	LastRunDuration = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "realty_parser_last_run_duration_seconds",
		Help: "Duration of the last worker run in seconds",
	})
	LastRunFlatsProcessed = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "realty_parser_last_run_flats_processed",
		Help: "Number of flats processed in the last worker run",
	})

	// Histograms

	FlatScores = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "realty_parser_flat_score",
		Help:    "Distribution of calculated flat scores",
		Buckets: prometheus.LinearBuckets(400000, 50000, 20),
	})
	FlatPrices = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "realty_parser_flat_price_rub",
		Help:    "Distribution of flat prices in rubles",
		Buckets: prometheus.LinearBuckets(30000, 10000, 20),
	})
)
