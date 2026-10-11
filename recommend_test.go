package mwanachamaratings

import (
	"errors"
	"math"
	"testing"
)

func TestRecommendRanksUnratedItemsByNeighbourWeightedValue(t *testing.T) {
	m, ctx := steppingManager(t)
	shareThree(t, m, ctx, keyA, keyB, 0.1)
	shareThree(t, m, ctx, keyA, keyC, 0.4)
	rate(t, m, ctx, keyB, item("p"), 0.9)
	rate(t, m, ctx, keyC, item("p"), 0.1)
	rate(t, m, ctx, keyB, item("q"), 0.3)
	rate(t, m, ctx, keyC, item("r"), 1.0)

	got, err := m.Recommend(ctx, keyA, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("want p, q and r, got %+v", got)
	}
	ab := neighbours(t, m, ctx, keyA, KindRater)
	wantP := (ab[keyB].Score*0.9 + ab[keyC].Score*0.1) / (ab[keyB].Score + ab[keyC].Score)
	byItem := map[string]Recommendation{}
	for _, r := range got {
		byItem[r.Item] = r
	}
	if p := byItem[item("p")]; math.Abs(p.Predicted-wantP) > 1e-9 || p.Support != 2 {
		t.Errorf("p = %+v, want predicted %v over 2 neighbours", p, wantP)
	}
	if r := byItem[item("r")]; math.Abs(r.Predicted-1.0) > 1e-9 || r.Support != 1 {
		t.Errorf("r = %+v, want 1.0 from one neighbour", r)
	}
	for i := 1; i < len(got); i++ {
		if got[i].Predicted > got[i-1].Predicted {
			t.Errorf("not ranked: %+v", got)
		}
	}
}

func TestRecommendNeverReturnsAnItemTheRaterHasRated(t *testing.T) {
	m, ctx := steppingManager(t)
	shareThree(t, m, ctx, keyA, keyB, 0.1)
	rate(t, m, ctx, keyB, item("p"), 1.0)
	got, err := m.Recommend(ctx, keyA, 10)
	if err != nil || len(got) != 1 || got[0].Item != item("p") {
		t.Fatalf("before rating p: %+v, %v", got, err)
	}
	rate(t, m, ctx, keyA, item("p"), 0.2)
	got, err = m.Recommend(ctx, keyA, 10)
	if err != nil || len(got) != 0 {
		t.Errorf("after rating p it must not come back: %+v, %v", got, err)
	}
}

func TestARaterWithNoNeighboursGetsAnEmptyListNotAnError(t *testing.T) {
	m, ctx := steppingManager(t)
	rate(t, m, ctx, keyB, item("p"), 1.0)
	got, err := m.Recommend(ctx, keyA, 10)
	if err != nil || got == nil || len(got) != 0 {
		t.Errorf("newcomer: %#v, %v", got, err)
	}
	if _, err := m.Recommend(ctx, "", 10); !errors.Is(err, ErrInvalidRating) {
		t.Errorf("no rater: %v", err)
	}
}

func TestRecommendIsBoundedByN(t *testing.T) {
	m, ctx := steppingManager(t)
	shareThree(t, m, ctx, keyA, keyB, 0.1)
	for _, it := range []string{"p", "q", "r", "s"} {
		rate(t, m, ctx, keyB, item(it), 0.7)
	}
	got, err := m.Recommend(ctx, keyA, 2)
	if err != nil || len(got) != 2 {
		t.Errorf("n = 2: %+v, %v", got, err)
	}
	got, err = m.Recommend(ctx, keyA, 0)
	if err != nil || len(got) != 4 {
		t.Errorf("n = 0 means the default of 10: %+v, %v", got, err)
	}
}
