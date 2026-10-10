package mwanachamaratings_test

import (
	"errors"
	"testing"

	ratings "github.com/aosanya/mwanachama-backend-ratings"
)

func TestSettingsDefaultToTheDecidedNumbers(t *testing.T) {
	rm, ctx := newManager(t)
	got, err := rm.Settings(ctx)
	if err != nil {
		t.Fatalf("settings: %v", err)
	}
	if got.Metric != ratings.MetricEuclidean || got.K != 30 || got.Overlap != 3 {
		t.Errorf("defaults = %+v, want euclidean, K 30, overlap 3", got)
	}
	if !got.Runs(ratings.KindRater) || got.Runs(ratings.KindItem) {
		t.Errorf("by default only rater-to-rater runs, got %v", got.Similarities)
	}
}

func TestAPutSettingIsReadBack(t *testing.T) {
	rm, ctx := newManager(t)
	for key, value := range map[string]string{
		ratings.SettingMetric:       "jaccard",
		ratings.SettingSimilarities: "rater, item",
		ratings.SettingK:            "50",
		ratings.SettingOverlap:      "5",
	} {
		if _, err := rm.PutSetting(ctx, key, value); err != nil {
			t.Fatalf("put %s: %v", key, err)
		}
	}
	got, err := rm.Settings(ctx)
	if err != nil {
		t.Fatalf("settings: %v", err)
	}
	if got.Metric != ratings.MetricJaccard || got.K != 50 || got.Overlap != 5 || !got.Runs(ratings.KindItem) || !got.Runs(ratings.KindRater) {
		t.Errorf("settings = %+v", got)
	}

	if _, err := rm.PutSetting(ctx, ratings.SettingK, "10"); err != nil {
		t.Fatalf("put again: %v", err)
	}
	s, err := rm.GetSetting(ctx, ratings.SettingK)
	if err != nil || s.Value != "10" {
		t.Fatalf("a second put replaces the row: got %+v, %v", s, err)
	}
	if got, _ = rm.Settings(ctx); got.K != 10 {
		t.Errorf("K = %d after replacing, want 10", got.K)
	}
}

func TestASettingIsHeldToTheBlueprintsVocabulary(t *testing.T) {
	rm, ctx := newManager(t)
	for key, value := range map[string]string{
		ratings.SettingMetric:       "pearson",
		ratings.SettingSimilarities: "rater,movie",
		ratings.SettingK:            "0",
		ratings.SettingOverlap:      "three",
		"half_life":                 "7d",
	} {
		_, err := rm.PutSetting(ctx, key, value)
		if !errors.Is(err, ratings.ErrInvalidSetting) {
			t.Errorf("put %s=%q: want ErrInvalidSetting, got %v", key, value, err)
		}
	}
	if _, err := rm.GetSetting(ctx, ratings.SettingMetric); !errors.Is(err, ratings.ErrSettingNotFound) {
		t.Errorf("a refused put stores nothing, got %v", err)
	}
}
