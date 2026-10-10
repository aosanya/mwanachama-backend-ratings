package mwanachamaratings_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	ratings "github.com/aosanya/mwanachama-backend-ratings"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

const (
	titlesExamplePath = "spec/examples/titles.ratings.json"
	pantryExamplePath = "spec/examples/pantry.ratings.json"
)

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return db
}

func loadExample(t *testing.T, path string) *spec.Spec {
	t.Helper()
	s, err := ratings.LoadSpec(filepath.Join(".", path))
	if err != nil {
		t.Fatalf("load %s: %v", path, err)
	}
	return s
}

func newManager(t *testing.T) (ratings.RatingsManager, context.Context) {
	t.Helper()
	db := openTestDB(t)
	s := loadExample(t, titlesExamplePath)
	if err := spec.Migrate(db, s); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	rm, err := ratings.NewRatingsManager(db, s)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	return rm, context.Background()
}
