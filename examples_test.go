package mwanachamaratings_test

import (
	"sort"
	"strings"
	"testing"

	"gorm.io/gorm"

	ratings "github.com/aosanya/mwanachama-backend-ratings"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

func tableNames(t *testing.T, db *gorm.DB) []string {
	t.Helper()
	var names []string
	if err := db.Raw(`select name from sqlite_master where type='table' and name not like 'sqlite_%'
		and name not like '%_' || ?`, spec.NameRegistrySuffix).
		Scan(&names).Error; err != nil {
		t.Fatalf("list tables: %v", err)
	}
	sort.Strings(names)
	return names
}

func declaredTables(specs ...*spec.Spec) []string {
	var out []string
	for _, s := range specs {
		for _, o := range s.Objects {
			out = append(out, s.TableFor(o))
		}
	}
	sort.Strings(out)
	return out
}

func TestShippedExamplesCoexist(t *testing.T) {
	db := openTestDB(t)
	titles := loadExample(t, titlesExamplePath)
	pantry := loadExample(t, pantryExamplePath)

	for _, s := range []*spec.Spec{titles, pantry} {
		if err := spec.Migrate(db, s); err != nil {
			t.Fatalf("migrate %s: %v", s.Domain, err)
		}
		if _, err := ratings.NewRatingsManager(db, s); err != nil {
			t.Fatalf("manager %s: %v", s.Domain, err)
		}
	}

	want := declaredTables(titles, pantry)
	if got := tableNames(t, db); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("tables =\n  %v\nwant\n  %v", got, want)
	}

	for _, tc := range []struct {
		s        *spec.Spec
		wantName string
		wantRaw  string
	}{
		{titles, "review", "ratings_main_reviews"},
		{pantry, "verdict", "ratings_main_verdicts"},
	} {
		o, ok := tc.s.ByRole(ratings.RoleRating)
		if !ok {
			t.Fatalf("%s: no object fills the rating role", tc.s.Domain)
		}
		if o.Name != tc.wantName || tc.s.RawNameFor(o) != tc.wantRaw {
			t.Errorf("%s: rating is %q raw-named %q, want %q named %q",
				tc.s.Domain, o.Name, tc.s.RawNameFor(o), tc.wantName, tc.wantRaw)
		}
	}
}

func TestShippedExampleMigratesIdempotently(t *testing.T) {
	db := openTestDB(t)
	s := loadExample(t, titlesExamplePath)
	for i := 0; i < 3; i++ {
		if err := spec.Migrate(db, s); err != nil {
			t.Fatalf("migrate %d: %v", i, err)
		}
	}
	if n := len(tableNames(t, db)); n != 3 {
		t.Errorf("tables = %d, want 3", n)
	}
}

func TestOneRowPerRaterAndItemIsTheKey(t *testing.T) {
	ddl := strings.Join(loadExample(t, titlesExamplePath).DDL("sqlite"), "\n")
	for _, want := range []string{"primary key (rater, item)", "primary key (a, kind, b)"} {
		if !strings.Contains(ddl, want) {
			t.Errorf("DDL is missing %q", want)
		}
	}
}

func TestShippedExamplesAreDescribed(t *testing.T) {
	sentence := func(s string) bool { return len(strings.Fields(s)) >= 3 }
	restates := func(desc, name string) bool {
		d := strings.ToLower(strings.Trim(desc, " ."))
		return d == strings.ReplaceAll(name, "_", " ") || d == name
	}
	for _, path := range []string{titlesExamplePath, pantryExamplePath} {
		s := loadExample(t, path)
		for _, o := range s.Objects {
			if !sentence(o.Description) || restates(o.Description, o.Name) {
				t.Errorf("%s: object %q is not described: %q", path, o.Name, o.Description)
			}
			for _, f := range o.Fields {
				if !sentence(f.Description) || restates(f.Description, f.Name) {
					t.Errorf("%s: %s.%s is not described: %q", path, o.Name, f.Name, f.Description)
				}
			}
		}
	}
}
