package mwanachamaratings_test

import (
	"strings"
	"testing"

	ratings "github.com/aosanya/mwanachama-backend-ratings"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

func TestProvisionMigratesIdempotentlyAndRegistersEveryTable(t *testing.T) {
	db := openTestDB(t)
	s := loadExample(t, titlesExamplePath)
	for i := 0; i < 2; i++ {
		if err := ratings.Provision(db, s); err != nil {
			t.Fatalf("provision %d: %v", i, err)
		}
	}
	if got := tableNames(t, db); len(got) != 3 {
		t.Errorf("tables = %v, want the three declared objects", got)
	}
	var registered int64
	if err := db.Table(s.Instance + "_" + spec.NameRegistrySuffix).Count(&registered).Error; err != nil {
		t.Fatalf("registry: %v", err)
	}
	if registered != 3 {
		t.Errorf("registry holds %d rows, want 3", registered)
	}
	if _, err := ratings.NewRatingsManager(db, s); err != nil {
		t.Errorf("a provisioned database builds a manager: %v", err)
	}
}

func TestProvisionRefusesAnotherModulesSpec(t *testing.T) {
	db := openTestDB(t)
	s := loadExample(t, titlesExamplePath)
	foreign := *s
	foreign.Module = "catalog"
	err := ratings.Provision(db, &foreign)
	if err == nil || !strings.Contains(err.Error(), `"catalog"`) {
		t.Errorf("a foreign spec: %v", err)
	}
	if got := tableNames(t, db); len(got) != 0 {
		t.Errorf("a refused provision creates nothing, got %v", got)
	}
	if err := ratings.Provision(db, nil); err == nil {
		t.Error("a nil spec is refused")
	}
}
