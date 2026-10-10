package mwanachamaratings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/aosanya/mwanachama-backend-ratings/models"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

func openDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return db
}

func examplePaths(t *testing.T) []string {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join("spec", "examples", "*.json"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("glob examples: %v (%d found)", err, len(paths))
	}
	sort.Strings(paths)
	return paths
}

func TestVocabularyMatchesTheBlueprint(t *testing.T) {
	inGo := map[string][]string{
		roleSimilarity + ".kind": {
			string(models.KindRater),
			string(models.KindItem),
		},
		roleSimilarity + ".metric": {
			string(models.MetricEuclidean),
			string(models.MetricCosine),
			string(models.MetricJaccard),
		},
	}

	b, err := Blueprint()
	if err != nil {
		t.Fatalf("blueprint: %v", err)
	}

	for _, o := range b.Objects {
		for _, f := range o.Fields {
			if f.Type != spec.TypeEnum {
				continue
			}
			key := o.Role + "." + f.Name
			want, named := inGo[key]
			if !named {
				t.Errorf("the blueprint declares the enum %s with the values %s, and Go names no constants for them",
					key, strings.Join(f.Values, ", "))
				continue
			}
			delete(inGo, key)
			if !sameValues(want, f.Values) {
				t.Errorf("%s: Go names %s, the blueprint declares %s — a stored value outlives a rename, so these have to agree",
					key, strings.Join(sorted(want), ", "), strings.Join(sorted(f.Values), ", "))
			}
		}
	}

	for key := range inGo {
		t.Errorf("Go names constants for %s, which the blueprint declares no enum for", key)
	}
}

func TestEveryExampleFitsTheTypes(t *testing.T) {
	for _, path := range examplePaths(t) {
		t.Run(filepath.Base(path), func(t *testing.T) {
			s, err := LoadSpec(path)
			if err != nil {
				t.Fatalf("load: %v", err)
			}
			db := openDB(t)
			if err := spec.Migrate(db, s); err != nil {
				t.Fatalf("migrate: %v", err)
			}
			if _, err := NewRatingsManager(db, s); err != nil {
				t.Fatalf("manager: %v", err)
			}
		})
	}
}

func TestAManagerRefusesASpecMissingARole(t *testing.T) {
	s, err := LoadSpec("spec/examples/titles.ratings.json")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	var kept []spec.Object
	for _, o := range s.Objects {
		if o.Role != roleSetting {
			kept = append(kept, o)
		}
	}
	s.Objects = kept
	db := openDB(t)
	if err := spec.Migrate(db, s); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := NewRatingsManager(db, s); err == nil || !strings.Contains(err.Error(), roleSetting) {
		t.Fatalf("a spec filling no %s should fail to build a manager naming it, got %v", roleSetting, err)
	}
}

func TestExamplesDeclareNoFieldsOfTheirOwn(t *testing.T) {
	for _, path := range examplePaths(t) {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		var doc struct {
			Objects []struct {
				Name   string `json:"name"`
				Role   string `json:"role"`
				Fields []struct {
					Name        string   `json:"name"`
					Type        string   `json:"type"`
					Description string   `json:"description"`
					Values      []string `json:"values"`
				} `json:"fields"`
			} `json:"objects"`
		}
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		for _, o := range doc.Objects {
			if o.Role == "" {
				t.Errorf("%s: %s claims no role, and this module's examples only name the module's objects", filepath.Base(path), o.Name)
			}
			for _, f := range o.Fields {
				if f.Type != "" || f.Description != "" || len(f.Values) > 0 {
					t.Errorf("%s: %s restates the declaration of %q — a domain names the object and sets defaults, the module declares the field",
						filepath.Base(path), o.Name, f.Name)
				}
			}
		}
	}
}

func TestRequiredFieldsHaveNoDefault(t *testing.T) {
	b, err := Blueprint()
	if err != nil {
		t.Fatalf("blueprint: %v", err)
	}
	for _, o := range b.Objects {
		for _, f := range o.Fields {
			if f.Required && f.Default != "" {
				t.Errorf("%s.%s is required and defaults to %q", o.Role, f.Name, f.Default)
			}
		}
	}
	for _, path := range examplePaths(t) {
		s, err := LoadSpec(path)
		if err != nil {
			t.Fatalf("load %s: %v", path, err)
		}
		for _, stmt := range s.DDL("postgres") {
			for _, line := range strings.Split(stmt, "\n") {
				line = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(line), ","))
				if strings.Contains(line, "not null") && strings.Contains(line, "default") {
					t.Errorf("%s: column is both required and defaulted: %q", path, line)
				}
			}
		}
	}
}

func sameValues(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	x, y := sorted(a), sorted(b)
	for i := range x {
		if x[i] != y[i] {
			return false
		}
	}
	return true
}

func sorted(in []string) []string {
	out := append([]string{}, in...)
	sort.Strings(out)
	return out
}
