package models

import (
	"math"
	"testing"
)

func TestRatingValueInRangeIsClosedOnBothEnds(t *testing.T) {
	cases := map[string]struct {
		value float64
		want  bool
	}{
		"zero is allowed":         {0, true},
		"one is allowed":          {1, true},
		"middle is allowed":       {0.5, true},
		"just below zero refused": {-0.0001, false},
		"just above one refused":  {1.0001, false},
		"not a number refused":    {math.NaN(), false},
		"positive infinity":       {math.Inf(1), false},
		"negative infinity":       {math.Inf(-1), false},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if got := (Rating{Value: c.value}).ValueInRange(); got != c.want {
				t.Fatalf("ValueInRange(%v) = %v, want %v", c.value, got, c.want)
			}
		})
	}
}

func TestSimilarityScoreInRangeExcludesZeroAndIncludesOne(t *testing.T) {
	cases := map[string]struct {
		score float64
		want  bool
	}{
		"zero refused":         {0, false},
		"tiny positive":        {math.SmallestNonzeroFloat64, true},
		"one allowed":          {1, true},
		"above one refused":    {1.0001, false},
		"negative refused":     {-0.5, false},
		"not a number refused": {math.NaN(), false},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if got := (Similarity{Score: c.score}).ScoreInRange(); got != c.want {
				t.Fatalf("ScoreInRange(%v) = %v, want %v", c.score, got, c.want)
			}
		})
	}
}

func TestSimilarityOverlapIsCountRefusesNegatives(t *testing.T) {
	for overlap, want := range map[int]bool{-1: false, 0: true, 1: true, 1000: true} {
		if got := (Similarity{Overlap: overlap}).OverlapIsCount(); got != want {
			t.Errorf("OverlapIsCount(%d) = %v, want %v", overlap, got, want)
		}
	}
}

func TestListAndRecommendLimitsAreOrderedSoDefaultsFitTheirCaps(t *testing.T) {
	if DefaultRecommendLimit < 1 || DefaultRecommendLimit > MaxRecommendLimit {
		t.Fatalf("default recommend limit %d outside 1..%d", DefaultRecommendLimit, MaxRecommendLimit)
	}
	if MaxRatingListLimit < 1 || MaxSimilarityListLimit < 1 {
		t.Fatalf("list caps must be positive: %d %d", MaxRatingListLimit, MaxSimilarityListLimit)
	}
}

func TestVocabularyValuesAreTheStoredWords(t *testing.T) {
	stored := map[string]string{
		string(KindRater):       "rater",
		string(KindItem):        "item",
		string(MetricEuclidean): "euclidean",
		string(MetricCosine):    "cosine",
		string(MetricJaccard):   "jaccard",
	}
	for got, want := range stored {
		if got != want {
			t.Errorf("stored word %q, want %q", got, want)
		}
	}
}
