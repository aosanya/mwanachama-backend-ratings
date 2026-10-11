//go:build integration

package mwanachamaratings_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"math"
	"os"
	"testing"

	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	ratings "github.com/aosanya/mwanachama-backend-ratings"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

func randomInstance(t *testing.T) string {
	t.Helper()
	b := make([]byte, 5)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("random instance: %v", err)
	}
	return "pg" + hex.EncodeToString(b)
}

func newPostgresManager(t *testing.T) (ratings.RatingsManager, *gorm.DB, *spec.Spec, context.Context) {
	t.Helper()
	dsn := os.Getenv("POSTGRES_URL")
	if dsn == "" {
		t.Skip("POSTGRES_URL not set; skipping Postgres integration test")
	}
	db, err := gorm.Open(gormpostgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	s := loadExample(t, titlesExamplePath)
	s.Instance = randomInstance(t)
	if err := s.Validate(); err != nil {
		t.Fatalf("spec at instance %q: %v", s.Instance, err)
	}
	if err := ratings.Provision(db, s); err != nil {
		t.Fatalf("provision: %v", err)
	}
	t.Cleanup(func() {
		for _, o := range s.Objects {
			db.Exec("drop table if exists " + s.TableFor(o) + " cascade")
		}
		db.Exec("drop table if exists " + s.Instance + "_" + spec.NameRegistrySuffix + " cascade")
	})
	rm, err := ratings.NewRatingsManager(db, s)
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	return rm, db, s, context.Background()
}

func TestPostgresProvisionIsIdempotentAndTypesTheNumbers(t *testing.T) {
	_, db, s, _ := newPostgresManager(t)
	if err := ratings.Provision(db, s); err != nil {
		t.Fatalf("second provision: %v", err)
	}
	o, _ := s.ByRole(ratings.RoleRating)
	var dataType string
	err := db.Raw(`select data_type from information_schema.columns
		where table_schema = current_schema() and table_name = ? and column_name = 'value'`, s.TableFor(o)).
		Scan(&dataType).Error
	if err != nil || dataType != "double precision" {
		t.Errorf("value is %q on Postgres (%v), want double precision", dataType, err)
	}
}

func TestPostgresWriteLinksCoRatersAndRecommends(t *testing.T) {
	rm, _, _, ctx := newPostgresManager(t)
	for i, it := range []string{"x", "y", "z"} {
		v := 0.2 + 0.2*float64(i)
		for _, r := range []ratings.Rating{
			{Rater: raterA, Item: itemKey(it), Value: v},
			{Rater: raterB, Item: itemKey(it), Value: v + 0.1},
			{Rater: raterC, Item: itemKey(it), Value: math.Min(1, v+0.4)},
		} {
			if _, err := rm.PutRating(ctx, r); err != nil {
				t.Fatalf("put: %v", err)
			}
		}
	}
	put(t, rm, ctx, raterB, itemKey("p"), 0.9)
	put(t, rm, ctx, raterC, itemKey("q"), 0.2)

	sims, err := rm.ListSimilarities(ctx, ratings.SimilarityFilter{Key: raterA})
	if err != nil || len(sims) != 2 || sims[0].B != raterB {
		t.Fatalf("a's neighbours on Postgres: %+v, %v", sims, err)
	}
	want := 1 / (1 + math.Sqrt(3*0.1*0.1))
	if math.Abs(sims[0].Score-want) > 1e-9 || sims[0].Overlap != 3 {
		t.Errorf("a→b = %+v, want %v over 3", sims[0], want)
	}

	recs, err := rm.Recommend(ctx, raterA, 10)
	if err != nil || len(recs) != 2 || recs[0].Item != itemKey("p") {
		t.Fatalf("recommend on Postgres: %+v, %v", recs, err)
	}
	if recs[0].Support != 1 || math.Abs(recs[0].Predicted-0.9) > 1e-9 {
		t.Errorf("p = %+v, want 0.9 from one neighbour", recs[0])
	}

	if err := rm.DeleteRating(ctx, raterA, itemKey("z")); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if sims, _ := rm.ListSimilarities(ctx, ratings.SimilarityFilter{Key: raterA}); len(sims) != 0 {
		t.Errorf("under the floor after a delete, a still holds %+v", sims)
	}
}
