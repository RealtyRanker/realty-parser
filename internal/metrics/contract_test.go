package metrics

import (
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Freeze the existing public metric names, types, labels and bucket boundaries.
// Grafana queries and Prometheus jobs depend on this contract.
func TestMonitoringContract(t *testing.T) {

	expected := map[string]struct {
		kind    string
		labels  []string
		buckets []float64
	}{
		"realty_parser_pages_fetched_total":        {kind: "COUNTER", labels: []string{}, buckets: []float64{}},
		"realty_parser_flats_found_total":          {kind: "COUNTER", labels: []string{}, buckets: []float64{}},
		"realty_parser_flats_inserted_total":       {kind: "COUNTER", labels: []string{}, buckets: []float64{}},
		"realty_parser_parse_errors_total":         {kind: "COUNTER", labels: []string{}, buckets: []float64{}},
		"realty_parser_fetch_errors_total":         {kind: "COUNTER", labels: []string{}, buckets: []float64{}},
		"realty_parser_last_run_timestamp_seconds": {kind: "GAUGE", labels: []string{}, buckets: []float64{}},
		"realty_parser_last_run_duration_seconds":  {kind: "GAUGE", labels: []string{}, buckets: []float64{}},
		"realty_parser_last_run_flats_processed":   {kind: "GAUGE", labels: []string{}, buckets: []float64{}},
		"realty_parser_flat_score":                 {kind: "HISTOGRAM", labels: []string{}, buckets: []float64{400000.0, 450000.0, 500000.0, 550000.0, 600000.0, 650000.0, 700000.0, 750000.0, 800000.0, 850000.0, 900000.0, 950000.0, 1000000.0, 1050000.0, 1100000.0, 1150000.0, 1200000.0, 1250000.0, 1300000.0, 1350000.0}},
		"realty_parser_flat_price_rub":             {kind: "HISTOGRAM", labels: []string{}, buckets: []float64{30000.0, 40000.0, 50000.0, 60000.0, 70000.0, 80000.0, 90000.0, 100000.0, 110000.0, 120000.0, 130000.0, 140000.0, 150000.0, 160000.0, 170000.0, 180000.0, 190000.0, 200000.0, 210000.0, 220000.0}},
	}
	gathered, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, family := range gathered {
		contract, ok := expected[family.GetName()]
		if !ok {
			continue
		}
		if family.GetType().String() != contract.kind {
			t.Errorf("%s type changed", family.GetName())
		}
		for _, metric := range family.Metric {
			labels := []string{}
			for _, label := range metric.Label {
				labels = append(labels, label.GetName())
			}
			sort.Strings(labels)
			if !reflect.DeepEqual(labels, contract.labels) {
				t.Errorf("%s labels changed: %v", family.GetName(), labels)
			}
			if contract.kind == "HISTOGRAM" {
				var bounds []float64
				for _, bucket := range metric.GetHistogram().Bucket {
					bounds = append(bounds, bucket.GetUpperBound())
				}
				if !reflect.DeepEqual(bounds, contract.buckets) {
					t.Errorf("%s buckets changed: %v", family.GetName(), bounds)
				}
			}
		}
		delete(expected, family.GetName())
	}
	if len(expected) > 0 {
		t.Fatalf("missing metrics: %v", expected)
	}
	response := httptest.NewRecorder()
	promhttp.Handler().ServeHTTP(response, httptest.NewRequest("GET", "/metrics", nil))
	if response.Code != 200 || !strings.Contains(response.Body.String(), "# TYPE") {
		t.Fatalf("metrics endpoint: %d", response.Code)
	}
}
