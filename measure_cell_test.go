//go:build measure

package mwanachamaratings

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/aosanya/mwanachama-backend-ratings/models"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

const measureSpec = `{
  "module": "ratings",
  "domain": "measure",
  "instance": "measure1",
  "objects": [
    {"role": "rating", "name": "rating", "table": "ratings", "description": "One held value under measurement."},
    {"role": "similarity", "name": "similarity", "table": "similarities", "description": "One neighbour under measurement."},
    {"role": "setting", "name": "setting", "table": "settings", "description": "The cell's knobs."}
  ]
}`

type cell struct {
	metric  Metric
	k       int
	overlap int
}

func (c cell) String() string { return fmt.Sprintf("%s K=%d overlap=%d", c.metric, c.k, c.overlap) }

type writeCounter struct {
	table   string
	written int64
	deleted int64
}

func (w *writeCounter) attach(db *gorm.DB) error {
	count := func(d *gorm.DB) {
		if d.Statement.Table == w.table {
			w.written += d.RowsAffected
		}
	}
	if err := db.Callback().Create().After("gorm:create").Register("measure:create", count); err != nil {
		return err
	}
	if err := db.Callback().Update().After("gorm:update").Register("measure:update", count); err != nil {
		return err
	}
	return db.Callback().Raw().After("gorm:raw").Register("measure:raw", func(d *gorm.DB) {
		if strings.HasPrefix(strings.ToLower(d.Statement.SQL.String()), "delete from "+w.table) {
			w.deleted += d.RowsAffected
		}
	})
}

func openCell(t *testing.T, c cell) (RatingsManager, *writeCounter, context.Context) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	s, err := ParseSpec([]byte(measureSpec))
	if err != nil {
		t.Fatalf("measure spec: %v", err)
	}
	if err := spec.Migrate(db, s); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	rm, err := NewRatingsManager(db, s)
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	ctx := context.Background()
	for key, value := range map[string]string{
		SettingMetric:  string(c.metric),
		SettingK:       strconv.Itoa(c.k),
		SettingOverlap: strconv.Itoa(c.overlap),
	} {
		if _, err := rm.PutSetting(ctx, key, value); err != nil {
			t.Fatalf("%s: setting %s: %v", c, key, err)
		}
	}
	counter := &writeCounter{table: rm.(*ratingsManager).st.Table(roleSimilarity)}
	if err := counter.attach(db); err != nil {
		t.Fatalf("attach counter: %v", err)
	}
	return rm, counter, ctx
}

type cellResult struct {
	cell
	decided     scores
	supportTwo  scores
	centred     scores
	loadSeconds float64
	written     int64
	deleted     int64
}

func runCell(t *testing.T, c cell, load []observation, held map[string]observation, mx *matrix) cellResult {
	t.Helper()
	rm, counter, ctx := openCell(t, c)
	start := time.Now()
	for _, o := range load {
		if _, err := rm.PutRating(ctx, models.Rating{Rater: o.rater, Item: o.item, Value: o.value}); err != nil {
			t.Fatalf("%s: put %s %s: %v", c, o.rater, o.item, err)
		}
	}
	out := cellResult{cell: c, loadSeconds: time.Since(start).Seconds(), written: counter.written, deleted: counter.deleted}
	out.decided.heldOut, out.supportTwo.heldOut, out.centred.heldOut = len(held), len(held), len(held)
	for _, rater := range sortedRaters(held) {
		h := held[rater]
		neighbours := storedNeighbours(t, rm, ctx, c, rater)
		if pred, support, ok := predictFromNeighbours(mx, h.item, neighbours); ok {
			out.decided.add(pred, h.value)
			if support >= 2 {
				out.supportTwo.add(pred, h.value)
			}
		}
		if pred, ok := predictCentred(mx, rater, h.item, neighbours); ok {
			out.centred.add(pred, h.value)
		}
		if h.value >= likedThreshold {
			out.decided.rank(rankOf(t, rm, ctx, c, rater, h.item))
		}
	}
	return out
}

func storedNeighbours(t *testing.T, rm RatingsManager, ctx context.Context, c cell, rater string) []neighbour {
	t.Helper()
	rows, err := rm.ListSimilarities(ctx, SimilarityFilter{Key: rater, Kind: models.KindRater, Limit: c.k})
	if err != nil {
		t.Fatalf("%s: similarities of %s: %v", c, rater, err)
	}
	out := make([]neighbour, 0, len(rows))
	for _, r := range rows {
		out = append(out, neighbour{other: r.B, score: r.Score, overlap: r.Overlap})
	}
	return out
}

func rankOf(t *testing.T, rm RatingsManager, ctx context.Context, c cell, rater, item string) int {
	t.Helper()
	recs, err := rm.Recommend(ctx, rater, models.DefaultRecommendLimit)
	if err != nil {
		t.Fatalf("%s: recommend for %s: %v", c, rater, err)
	}
	for i, r := range recs {
		if r.Item == item {
			return i + 1
		}
	}
	return 0
}

func cellHeader() string {
	return row("metric", "K", "overlap", "coverage", "RMSE [0,1]", "RMSE ×4", "MAE", "hit rate @10", "ARHR @10", "load s", "rows written", "rows deleted") +
		"\n" + rule(12)
}

func (r cellResult) row() string {
	d := r.decided
	return row(string(r.metric), strconv.Itoa(r.k), strconv.Itoa(r.overlap), pct(d.coverage()),
		num(d.rmse(), 4), num(d.rmse()*4, 4), num(d.mae(), 4), pct(d.hitRate()), num(d.averageReciprocalHitRank(), 4),
		num(r.loadSeconds, 1), strconv.FormatInt(r.written, 10), strconv.FormatInt(r.deleted, 10))
}
