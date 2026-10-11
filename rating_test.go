package mwanachamaratings_test

import (
	"context"
	"errors"
	"testing"

	ratings "github.com/aosanya/mwanachama-backend-ratings"
)

const (
	raterA = "actor/main/a"
	raterB = "actor/main/b"
	raterC = "actor/main/c"
	raterD = "actor/main/d"
)

func itemKey(n string) string { return "catalog/cat1/" + n }

func put(t *testing.T, rm ratings.RatingsManager, ctx context.Context, rater, item string, value float64) {
	t.Helper()
	if _, err := rm.PutRating(ctx, ratings.Rating{Rater: rater, Item: item, Value: value}); err != nil {
		t.Fatalf("put %s %s %v: %v", rater, item, value, err)
	}
}

func TestAPutRatingIsReadBackAndARepeatReplacesIt(t *testing.T) {
	rm, ctx := newManager(t)
	put(t, rm, ctx, raterA, itemKey("x"), 0.8)
	got, err := rm.GetRating(ctx, raterA, itemKey("x"))
	if err != nil || got.Value != 0.8 || got.At.IsZero() {
		t.Fatalf("get after put = %+v, %v", got, err)
	}
	first := got.At

	put(t, rm, ctx, raterA, itemKey("x"), 0.2)
	rows, err := rm.ListRatings(ctx, ratings.RatingFilter{Rater: raterA})
	if err != nil || len(rows) != 1 || rows[0].Value != 0.2 {
		t.Fatalf("a repeat replaces the row: %+v, %v", rows, err)
	}
	if rows[0].At.Before(first) {
		t.Errorf("at went backwards: %v then %v", first, rows[0].At)
	}
}

func TestAPutOutsideTheDeclaredRulesOrTheBoundIsRefused(t *testing.T) {
	rm, ctx := newManager(t)
	for _, r := range []ratings.Rating{
		{Rater: raterA, Item: itemKey("x"), Value: 1.5},
		{Rater: raterA, Item: itemKey("x"), Value: -0.1},
		{Rater: "", Item: itemKey("x"), Value: 0.5},
		{Rater: raterA, Item: "", Value: 0.5},
	} {
		if _, err := rm.PutRating(ctx, r); !errors.Is(err, ratings.ErrInvalidRating) {
			t.Errorf("put %+v: want ErrInvalidRating, got %v", r, err)
		}
	}
	if _, err := rm.GetRating(ctx, raterA, itemKey("x")); !errors.Is(err, ratings.ErrRatingNotFound) {
		t.Errorf("a refused put stores nothing, got %v", err)
	}
}

func TestListingIsByRaterOrByItemAndNeverUnscoped(t *testing.T) {
	rm, ctx := newManager(t)
	put(t, rm, ctx, raterA, itemKey("x"), 0.1)
	put(t, rm, ctx, raterA, itemKey("y"), 0.2)
	put(t, rm, ctx, raterB, itemKey("x"), 0.3)

	byRater, err := rm.ListRatings(ctx, ratings.RatingFilter{Rater: raterA})
	if err != nil || len(byRater) != 2 {
		t.Errorf("by rater = %d rows, %v; want 2", len(byRater), err)
	}
	byItem, err := rm.ListRatings(ctx, ratings.RatingFilter{Item: itemKey("x")})
	if err != nil || len(byItem) != 2 {
		t.Errorf("by item = %d rows, %v; want 2", len(byItem), err)
	}
	both, err := rm.ListRatings(ctx, ratings.RatingFilter{Rater: raterB, Item: itemKey("x")})
	if err != nil || len(both) != 1 || both[0].Value != 0.3 {
		t.Errorf("by both = %+v, %v", both, err)
	}
	if _, err := rm.ListRatings(ctx, ratings.RatingFilter{}); !errors.Is(err, ratings.ErrInvalidRating) {
		t.Errorf("an unscoped listing is refused, got %v", err)
	}
	capped, err := rm.ListRatings(ctx, ratings.RatingFilter{Rater: raterA, Limit: 1})
	if err != nil || len(capped) != 1 {
		t.Errorf("limit 1 = %d rows, %v", len(capped), err)
	}
}

func TestADeleteRemovesTheRowAndAMissingOneIsNotFound(t *testing.T) {
	rm, ctx := newManager(t)
	put(t, rm, ctx, raterA, itemKey("x"), 0.5)
	if err := rm.DeleteRating(ctx, raterA, itemKey("x")); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := rm.GetRating(ctx, raterA, itemKey("x")); !errors.Is(err, ratings.ErrRatingNotFound) {
		t.Errorf("after delete: %v", err)
	}
	if err := rm.DeleteRating(ctx, raterA, itemKey("x")); !errors.Is(err, ratings.ErrRatingNotFound) {
		t.Errorf("a second delete: %v", err)
	}
}
