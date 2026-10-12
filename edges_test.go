package mwanachamaratings

import (
	"errors"
	"reflect"
	"testing"
)

func TestEdgesEmitsSimilarTasteForTheNamedRatersAtTheStoredScore(t *testing.T) {
	m, ctx := steppingManager(t)
	shareThree(t, m, ctx, keyA, keyB, 0.1)
	shareThree(t, m, ctx, keyA, keyC, 0.3)
	rate(t, m, ctx, keyD, item("p"), 0.6)

	got, err := m.Edges(ctx, "agy1", []string{keyA})
	if err != nil {
		t.Fatalf("edges: %v", err)
	}
	stored := neighbours(t, m, ctx, keyA, KindRater)
	if len(got) != len(stored) || len(got) == 0 {
		t.Fatalf("edges %+v, stored neighbours %+v", got, stored)
	}
	for _, e := range got {
		if e.Predicate != PredicateSimilarTaste || e.From != keyA {
			t.Errorf("edge %+v: want similar_taste from the named rater", e)
		}
		if s, ok := stored[e.To]; !ok || s.Score != e.Weight {
			t.Errorf("edge %+v: weight must be the stored score %+v", e, s)
		}
		if e.To == keyD {
			t.Errorf("d was not named and shares nothing with a; it must not appear")
		}
	}
	if got[0].Weight < got[len(got)-1].Weight {
		t.Errorf("edges of one rater come strongest first: %+v", got)
	}
}

func TestEdgesEmitsAPairOnceWhenBothSidesAreNamed(t *testing.T) {
	m, ctx := steppingManager(t)
	shareThree(t, m, ctx, keyA, keyB, 0.1)

	got, err := m.Edges(ctx, "agy1", []string{keyB, keyA, keyA})
	if err != nil {
		t.Fatalf("edges: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("a symmetric pair named from both sides is one edge, got %+v", got)
	}
	if got[0].From != keyA || got[0].To != keyB {
		t.Errorf("the pair reads from the lower key: %+v", got[0])
	}
}

func TestEdgesNeverCarriesAnItemSimilarity(t *testing.T) {
	m, ctx := steppingManager(t)
	if _, err := m.PutSetting(ctx, SettingSimilarities, "rater,item"); err != nil {
		t.Fatalf("setting: %v", err)
	}
	shareThree(t, m, ctx, keyA, keyB, 0.1)
	shareThree(t, m, ctx, keyC, keyD, 0.2)
	if len(neighbours(t, m, ctx, item("x"), KindItem)) == 0 {
		t.Fatalf("the fixture must hold item similarities for this test to mean anything")
	}

	got, err := m.Edges(ctx, "agy1", []string{keyA, item("x")})
	if err != nil {
		t.Fatalf("edges: %v", err)
	}
	if len(got) != len(neighbours(t, m, ctx, keyA, KindRater)) {
		t.Fatalf("only a's rater neighbours travel, got %+v", got)
	}
	for _, e := range got {
		if e.From != keyA || e.To == item("y") || e.To == item("z") {
			t.Errorf("only rater-to-rater taste travels, got %+v", e)
		}
	}
}

func TestEdgesForARaterWithNoNeighboursIsEmptyAndWritesNothing(t *testing.T) {
	m, ctx := steppingManager(t)
	shareThree(t, m, ctx, keyA, keyB, 0.1)
	rate(t, m, ctx, keyC, item("q"), 0.4)
	before := allSimilarities(t, m, ctx)

	got, err := m.Edges(ctx, "agy1", []string{keyC, "actor/main/never-rated"})
	if err != nil {
		t.Fatalf("edges: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("no neighbours, no edges: %+v", got)
	}
	if _, err := m.Edges(ctx, "agy1", []string{keyA, keyB}); err != nil {
		t.Fatalf("edges: %v", err)
	}
	if after := allSimilarities(t, m, ctx); !reflect.DeepEqual(before, after) {
		t.Fatalf("Edges is a read; the similarity rows changed")
	}
}

func TestEdgesRefusesAMalformedRequestByName(t *testing.T) {
	m, ctx := steppingManager(t)
	for name, call := range map[string]func() error{
		"empty partition": func() error { _, err := m.Edges(ctx, "", []string{keyA}); return err },
		"bad partition":   func() error { _, err := m.Edges(ctx, "Agy/1", []string{keyA}); return err },
		"empty key":       func() error { _, err := m.Edges(ctx, "agy1", []string{keyA, ""}); return err },
	} {
		if err := call(); !errors.Is(err, ErrInvalidEdges) {
			t.Errorf("%s: %v, want ErrInvalidEdges", name, err)
		}
	}
	if got, err := m.Edges(ctx, "agy1", nil); err != nil || len(got) != 0 {
		t.Errorf("no keys is no edges and no error: %+v %v", got, err)
	}
}

func TestEdgesSpansMoreKeysThanOneQueryHolds(t *testing.T) {
	m, ctx := steppingManager(t)
	shareThree(t, m, ctx, keyA, keyB, 0.1)
	keys := make([]string, 0, edgeKeysPerQuery+2)
	for i := 0; i < edgeKeysPerQuery+1; i++ {
		keys = append(keys, "actor/main/aa"+string(rune('a'+i%26))+string(rune('a'+i/26%26)))
	}
	keys = append(keys, keyB)
	got, err := m.Edges(ctx, "agy1", keys)
	if err != nil {
		t.Fatalf("edges: %v", err)
	}
	if len(got) != 1 || got[0].From != keyB {
		t.Fatalf("a key past the first query's batch must still be read: %+v", got)
	}
}
