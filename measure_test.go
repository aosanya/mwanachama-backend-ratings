//go:build measure

package mwanachamaratings

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-ratings/models"
)

const (
	measureDirVar     = "RATINGS_MEASURE_DIR"
	measureMetricsVar = "RATINGS_MEASURE_METRICS"
	measureKVar       = "RATINGS_MEASURE_K"
	measureOverlapVar = "RATINGS_MEASURE_OVERLAP"
)

var fullGrid = struct {
	metrics  []Metric
	ks       []int
	overlaps []int
}{
	metrics:  []Metric{models.MetricEuclidean, models.MetricCosine, models.MetricJaccard},
	ks:       []int{10, 20, 30, 50, 100},
	overlaps: []int{2, 3, 5, 10},
}

func gridFromEnv(t *testing.T) []cell {
	t.Helper()
	metrics := fullGrid.metrics
	if v := os.Getenv(measureMetricsVar); v != "" {
		metrics = nil
		for _, m := range strings.Split(v, ",") {
			metrics = append(metrics, Metric(strings.TrimSpace(m)))
		}
	}
	ks := intsFromEnv(t, measureKVar, fullGrid.ks)
	overlaps := intsFromEnv(t, measureOverlapVar, fullGrid.overlaps)
	var out []cell
	for _, m := range metrics {
		for _, k := range ks {
			for _, o := range overlaps {
				out = append(out, cell{metric: m, k: k, overlap: o})
			}
		}
	}
	return out
}

func intsFromEnv(t *testing.T, name string, fallback []int) []int {
	t.Helper()
	v := os.Getenv(name)
	if v == "" {
		return fallback
	}
	var out []int
	for _, part := range strings.Split(v, ",") {
		n, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || n < 1 {
			t.Fatalf("%s: %q is not a list of whole numbers", name, v)
		}
		out = append(out, n)
	}
	return out
}

func TestMeasure(t *testing.T) {
	dir := os.Getenv(measureDirVar)
	if dir == "" {
		t.Skipf("%s is unset; place the rating set and export its directory", measureDirVar)
	}
	all, err := readObservations(filepath.Join(dir, "u.data"))
	if err != nil {
		t.Fatal(err)
	}
	load, held := holdOutLatest(all)
	mx := buildMatrix(load)
	grid := gridFromEnv(t)
	out := os.Stdout

	fmt.Fprintf(out, "\n%d observations, %d raters, %d items, %d held out, %d loaded, %d cells\n\n",
		len(all), len(mx.byRater), len(mx.byItem), len(held), len(load), len(grid))

	fmt.Fprintln(out, "## Grid")
	fmt.Fprintln(out)
	fmt.Fprintln(out, cellHeader())
	results := make([]cellResult, 0, len(grid))
	for _, c := range grid {
		r := runCell(t, c, load, held, mx)
		results = append(results, r)
		fmt.Fprintln(out, r.row())
	}

	fmt.Fprintln(out)
	fmt.Fprintln(out, "## Baselines")
	fmt.Fprintln(out)
	fmt.Fprintln(out, row("baseline", "coverage", "RMSE [0,1]", "RMSE ×4", "MAE"))
	fmt.Fprintln(out, rule(5))
	for _, b := range baselines(mx) {
		s := scoreBaseline(b, held)
		s.heldOut = len(held)
		fmt.Fprintln(out, row(b.name, pct(s.coverage()), num(s.rmse(), 4), num(s.rmse()*4, 4), num(s.mae(), 4)))
	}

	cands := candidatesForAll(mx, held)
	fmt.Fprintln(out)
	fmt.Fprintln(out, "## Metric variants (computed in Go over the loaded matrix, not through put)")
	fmt.Fprintln(out)
	fmt.Fprintln(out, metricVariantHeader())
	for _, v := range metricVariants(mx, held, cands, grid) {
		fmt.Fprintln(out, v.row())
	}

	fmt.Fprintln(out)
	fmt.Fprintln(out, "## Prediction variants (centred and support ≥ 2 over the stored neighbours; soft weighting computed in Go)")
	fmt.Fprintln(out)
	fmt.Fprintln(out, predictionVariantHeader())
	for _, r := range results {
		soft := scoreInGo(mx, held, cands, softWeighted(r.metric), r.k, 1)
		fmt.Fprintln(out, predictionVariantRow(r, soft))
	}
	fmt.Fprintln(out)
}
