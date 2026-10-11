package mwanachamaratings

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/aosanya/mwanachama-backend-ratings/models"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

const (
	keyA = "actor/main/a"
	keyB = "actor/main/b"
	keyC = "actor/main/c"
	keyD = "actor/main/d"
)

func item(n string) string { return "catalog/cat1/" + n }

func steppingManager(t *testing.T) (*ratingsManager, context.Context) {
	t.Helper()
	db := openDB(t)
	s, err := LoadSpec("spec/examples/titles.ratings.json")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if err := spec.Migrate(db, s); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	rm, err := NewRatingsManager(db, s)
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	m := rm.(*ratingsManager)
	clock := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	m.now = func() time.Time {
		clock = clock.Add(time.Second)
		return clock
	}
	return m, context.Background()
}

func rate(t *testing.T, m *ratingsManager, ctx context.Context, rater, it string, v float64) {
	t.Helper()
	if _, err := m.PutRating(ctx, models.Rating{Rater: rater, Item: it, Value: v}); err != nil {
		t.Fatalf("put %s %s: %v", rater, it, err)
	}
}

func neighbours(t *testing.T, m *ratingsManager, ctx context.Context, key string, kind Kind) map[string]Similarity {
	t.Helper()
	rows, err := m.ListSimilarities(ctx, SimilarityFilter{Key: key, Kind: kind})
	if err != nil {
		t.Fatalf("list %s: %v", key, err)
	}
	out := map[string]Similarity{}
	for _, r := range rows {
		out[r.B] = r
	}
	return out
}

func allSimilarities(t *testing.T, m *ratingsManager, ctx context.Context) map[string]Similarity {
	t.Helper()
	var rows []map[string]any
	if err := m.similarities(ctx).Find(&rows).Error; err != nil {
		t.Fatalf("all: %v", err)
	}
	out := map[string]Similarity{}
	for _, row := range rows {
		var s Similarity
		if err := m.st.Take(m.similarities(ctx).Where("a = ? and kind = ? and b = ?", row["a"], row["kind"], row["b"]),
			roleSimilarity, &s, errNoRow); err != nil {
			t.Fatalf("decode: %v", err)
		}
		out[s.A+"|"+string(s.Kind)+"|"+s.B] = s
	}
	return out
}

func shareThree(t *testing.T, m *ratingsManager, ctx context.Context, a, b string, offset float64) {
	t.Helper()
	for i, it := range []string{"x", "y", "z"} {
		v := 0.2 + 0.2*float64(i)
		rate(t, m, ctx, a, item(it), v)
		rate(t, m, ctx, b, item(it), math.Min(1, v+offset))
	}
}

func TestAWriteLinksEachCoRaterAboveTheFloorBothWays(t *testing.T) {
	m, ctx := steppingManager(t)
	shareThree(t, m, ctx, keyA, keyB, 0.1)
	rate(t, m, ctx, keyC, item("x"), 0.5)
	rate(t, m, ctx, keyC, item("y"), 0.5)

	ab := neighbours(t, m, ctx, keyA, KindRater)
	ba := neighbours(t, m, ctx, keyB, KindRater)
	if len(ab) != 1 || len(ba) != 1 {
		t.Fatalf("a has %v, b has %v; want one neighbour each", ab, ba)
	}
	want := 1 / (1 + math.Sqrt(3*0.1*0.1))
	if got := ab[keyB]; math.Abs(got.Score-want) > 1e-9 || got.Overlap != 3 || got.Metric != MetricEuclidean {
		t.Errorf("a→b = %+v, want score %v over 3", got, want)
	}
	if got := ba[keyA]; math.Abs(got.Score-want) > 1e-9 {
		t.Errorf("b→a = %+v, want the same score", got)
	}
	if _, linked := ab[keyC]; linked {
		t.Errorf("c shares two items, under the floor of three, and was linked anyway")
	}
}

func TestAWriteTouchesOnlyRatersOfThatItem(t *testing.T) {
	m, ctx := steppingManager(t)
	shareThree(t, m, ctx, keyA, keyB, 0.1)
	shareThree(t, m, ctx, keyC, keyD, 0.3)
	rate(t, m, ctx, keyC, item("w"), 0.9)
	before := allSimilarities(t, m, ctx)
	if len(before) != 12 {
		t.Fatalf("expected every pair over x, y, z linked both ways (12 rows), got %d", len(before))
	}

	rate(t, m, ctx, keyA, item("w"), 0.8)

	after := allSimilarities(t, m, ctx)
	ratedW := map[string]bool{keyA: true, keyC: true}
	for id, was := range before {
		now, still := after[id]
		if !still {
			t.Errorf("%s was removed by a write that only added a rating", id)
			continue
		}
		touched := !now.At.Equal(was.At) || now.Score != was.Score
		if ratedW[now.A] && ratedW[now.B] {
			if !touched {
				t.Errorf("%s: both sides rated w and the row was not re-scored", id)
			}
			continue
		}
		if touched {
			t.Errorf("%s: neither side's relation to the written item changed, yet the row was written (%v → %v)", id, was.At, now.At)
		}
	}
}

func TestTopKIsMaintainedAndTheWeakestIsDisplaced(t *testing.T) {
	m, ctx := steppingManager(t)
	if _, err := m.PutSetting(ctx, SettingK, "2"); err != nil {
		t.Fatal(err)
	}
	for i, it := range []string{"x", "y", "z"} {
		v := 0.2 + 0.2*float64(i)
		rate(t, m, ctx, keyA, item(it), v)
		rate(t, m, ctx, keyB, item(it), math.Min(1, v+0.3))
		rate(t, m, ctx, keyC, item(it), math.Min(1, v+0.2))
	}
	got := neighbours(t, m, ctx, keyA, KindRater)
	if len(got) != 2 {
		t.Fatalf("a holds %d neighbours, want K = 2", len(got))
	}
	for i, it := range []string{"x", "y", "z"} {
		rate(t, m, ctx, keyD, item(it), 0.2+0.2*float64(i)+0.05)
	}
	got = neighbours(t, m, ctx, keyA, KindRater)
	if len(got) != 2 {
		t.Fatalf("a holds %d neighbours after a closer rater, want still 2", len(got))
	}
	if _, kept := got[keyB]; kept {
		t.Errorf("the weakest neighbour b stayed when d beat it: %v", got)
	}
	if _, in := got[keyD]; !in {
		t.Errorf("the closer rater d did not enter: %v", got)
	}
	if n := neighbours(t, m, ctx, keyB, KindRater); len(n) != 2 {
		t.Errorf("b's own set is its own: %v", n)
	}
}

func TestADeleteBelowTheFloorRemovesThePairBothWays(t *testing.T) {
	m, ctx := steppingManager(t)
	shareThree(t, m, ctx, keyA, keyB, 0.1)
	if err := m.DeleteRating(ctx, keyA, item("z")); err != nil {
		t.Fatal(err)
	}
	if n := neighbours(t, m, ctx, keyA, KindRater); len(n) != 0 {
		t.Errorf("a still has %v after dropping under the floor", n)
	}
	if n := neighbours(t, m, ctx, keyB, KindRater); len(n) != 0 {
		t.Errorf("b still has %v after a dropped under the floor", n)
	}
}

func TestADeleteRefreshesAShrunkSetFromCandidatesNeverStored(t *testing.T) {
	m, ctx := steppingManager(t)
	if _, err := m.PutSetting(ctx, SettingK, "1"); err != nil {
		t.Fatal(err)
	}
	shareThree(t, m, ctx, keyA, keyB, 0.1)
	for i, it := range []string{"x", "y", "z"} {
		rate(t, m, ctx, keyC, item(it), math.Min(1, 0.2+0.2*float64(i)+0.4))
	}
	rate(t, m, ctx, keyB, item("w"), 0.5)
	rate(t, m, ctx, keyA, item("w"), 0.5)
	if n := neighbours(t, m, ctx, keyA, KindRater); len(n) != 1 || n[keyB].A == "" {
		t.Fatalf("a's one slot should hold b: %v", n)
	}

	for _, it := range []string{"z", "w"} {
		if err := m.DeleteRating(ctx, keyB, item(it)); err != nil {
			t.Fatal(err)
		}
	}
	n := neighbours(t, m, ctx, keyA, KindRater)
	if len(n) != 1 {
		t.Fatalf("a's set was not refilled after b left it: %v", n)
	}
	if _, in := n[keyC]; !in {
		t.Errorf("the candidate c, never stored under K = 1, did not enter on refresh: %v", n)
	}
	if nb := neighbours(t, m, ctx, keyB, KindRater); len(nb) != 0 {
		t.Errorf("b shares two items with a and one with c, so holds nothing: %v", nb)
	}
}

func TestItemToItemRunsOnlyWhenTheInstanceDeclaresIt(t *testing.T) {
	m, ctx := steppingManager(t)
	shareThree(t, m, ctx, keyA, keyB, 0.1)
	rate(t, m, ctx, keyC, item("x"), 0.3)
	rate(t, m, ctx, keyC, item("y"), 0.3)
	rate(t, m, ctx, keyC, item("z"), 0.3)
	if n := neighbours(t, m, ctx, item("x"), KindItem); len(n) != 0 {
		t.Fatalf("item rows exist under the default: %v", n)
	}
	if _, err := m.PutSetting(ctx, SettingSimilarities, "rater,item"); err != nil {
		t.Fatal(err)
	}
	rate(t, m, ctx, keyA, item("x"), 0.25)
	n := neighbours(t, m, ctx, item("x"), KindItem)
	if len(n) != 2 {
		t.Fatalf("x should be linked to y and z over three raters, got %v", n)
	}
	for _, other := range []string{"y", "z"} {
		if row := n[item(other)]; row.Kind != KindItem || row.Overlap != 3 {
			t.Errorf("x→%s = %+v", other, row)
		}
		if back := neighbours(t, m, ctx, item(other), KindItem); back[item("x")].A == "" {
			t.Errorf("%s→x is missing: %v", other, back)
		}
	}
}

func TestListSimilaritiesDefaultsToRaterAndRanksByScore(t *testing.T) {
	m, ctx := steppingManager(t)
	shareThree(t, m, ctx, keyA, keyB, 0.1)
	for i, it := range []string{"x", "y", "z"} {
		rate(t, m, ctx, keyC, item(it), math.Min(1, 0.2+0.2*float64(i)+0.3))
	}
	rows, err := m.ListSimilarities(ctx, SimilarityFilter{Key: keyA})
	if err != nil || len(rows) != 2 {
		t.Fatalf("list = %v, %v", rows, err)
	}
	if rows[0].B != keyB || rows[0].Score < rows[1].Score {
		t.Errorf("want the closer rater first: %+v", rows)
	}
	if _, err := m.ListSimilarities(ctx, SimilarityFilter{}); err != ErrInvalidSimilarity {
		t.Errorf("a listing with no key: %v", err)
	}
}
