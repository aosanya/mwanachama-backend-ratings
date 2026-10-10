package mwanachamaratings_test

import (
	"strings"
	"testing"
	"time"

	ratings "github.com/aosanya/mwanachama-backend-ratings"
)

var now = time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)

func TestARatingIsHeldToTheDeclaredRulesAndTheBound(t *testing.T) {
	s := loadExample(t, titlesExamplePath)
	good := ratings.Rating{Rater: "acct/a1/u-1", Item: "cat/c1/t-1", Value: 0.75, At: now}
	if err := ratings.Check(s, ratings.RoleRating, good); err != nil {
		t.Fatalf("a well-formed rating is refused: %v", err)
	}
	for _, edge := range []float64{0, 1} {
		r := good
		r.Value = edge
		if err := ratings.Check(s, ratings.RoleRating, r); err != nil {
			t.Errorf("value %v is inside the closed range and was refused: %v", edge, err)
		}
	}
	for name, bad := range map[string]ratings.Rating{
		"no rater":      {Item: good.Item, Value: 0.5, At: now},
		"blank item":    {Rater: good.Rater, Item: "  ", Value: 0.5, At: now},
		"no instant":    {Rater: good.Rater, Item: good.Item, Value: 0.5},
		"value above 1": {Rater: good.Rater, Item: good.Item, Value: 1.0001, At: now},
		"value below 0": {Rater: good.Rater, Item: good.Item, Value: -0.0001, At: now},
	} {
		if err := ratings.Check(s, ratings.RoleRating, bad); err == nil {
			t.Errorf("%s: accepted %+v", name, bad)
		}
	}
}

func TestASimilarityIsHeldToTheDeclaredRulesAndTheBounds(t *testing.T) {
	s := loadExample(t, titlesExamplePath)
	good := ratings.Similarity{A: "acct/a1/u-1", Kind: ratings.KindRater, B: "acct/a1/u-2",
		Metric: ratings.MetricCosine, Score: 1, Overlap: 3, At: now}
	if err := ratings.Check(s, ratings.RoleSimilarity, &good); err != nil {
		t.Fatalf("a well-formed similarity is refused: %v", err)
	}
	for name, mutate := range map[string]func(*ratings.Similarity){
		"undeclared kind":   func(x *ratings.Similarity) { x.Kind = "movie" },
		"undeclared metric": func(x *ratings.Similarity) { x.Metric = "pearson" },
		"score of zero":     func(x *ratings.Similarity) { x.Score = 0 },
		"score above one":   func(x *ratings.Similarity) { x.Score = 1.5 },
		"negative overlap":  func(x *ratings.Similarity) { x.Overlap = -1 },
		"no neighbour":      func(x *ratings.Similarity) { x.B = "" },
	} {
		bad := good
		mutate(&bad)
		if err := ratings.Check(s, ratings.RoleSimilarity, bad); err == nil {
			t.Errorf("%s: accepted %+v", name, bad)
		}
	}
}

func TestVocabularyComesFromTheSpec(t *testing.T) {
	shipped := loadExample(t, titlesExamplePath)
	widened, err := ratings.ParseSpec([]byte(`{
	  "module": "ratings", "domain": "titles", "instance": "cat2",
	  "objects": [
	    {"role": "rating", "name": "review", "table": "reviews", "description": "A reader's star rating of one title."},
	    {"role": "similarity", "name": "affinity", "table": "affinities",
	     "description": "Two readers with alike taste, or two reading groups.",
	     "fields": [{"name": "kind", "values": ["group"]}]},
	    {"role": "setting", "name": "setting", "table": "settings", "description": "The catalogue's knobs."}
	  ]}`))
	if err != nil {
		t.Fatalf("a domain may widen an enum: %v", err)
	}
	sim := ratings.Similarity{A: "x", Kind: "group", B: "y", Metric: ratings.MetricJaccard, Score: 0.5, Overlap: 4, At: now}
	if err := ratings.Check(shipped, ratings.RoleSimilarity, sim); err == nil || !strings.Contains(err.Error(), "group") {
		t.Errorf("the shipped spec should refuse the kind %q by name, got %v", sim.Kind, err)
	}
	if err := ratings.Check(widened, ratings.RoleSimilarity, sim); err != nil {
		t.Errorf("a spec that declares the kind should accept it: %v", err)
	}
}

func TestCheckNamesARoleTheDomainDoesNotFill(t *testing.T) {
	s := loadExample(t, titlesExamplePath)
	if err := ratings.Check(s, "ledger", ratings.Rating{}); err == nil || !strings.Contains(err.Error(), "ledger") {
		t.Errorf("want an error naming the role, got %v", err)
	}
}
