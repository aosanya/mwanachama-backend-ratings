package mwanachamaratings_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ratings "github.com/aosanya/mwanachama-backend-ratings"
)

var domainWords = []string{
	"movie", "film", "member", "user", "customer", "product",
	"title", "review", "reader", "book", "isbn", "librar", "patron",
	"recipe", "dish", "pantry", "household", "verdict",
	"agency", "agencies", "chapter", "cohort", "person",
}

func TestNoDomainWordsInIdentifiers(t *testing.T) {
	for _, dir := range []string{"models", "routes", "."} {
		if _, err := os.Stat(dir); err != nil {
			continue
		}
		fset := token.NewFileSet()
		pkgs, err := parser.ParseDir(fset, dir, func(fi os.FileInfo) bool {
			return !strings.HasSuffix(fi.Name(), "_test.go")
		}, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", dir, err)
		}
		seen := map[string]bool{}
		for _, pkg := range pkgs {
			for path, file := range pkg.Files {
				ast.Inspect(file, func(n ast.Node) bool {
					id, ok := n.(*ast.Ident)
					if !ok {
						return true
					}
					w, bad := carriesDomainWord(id.Name)
					if !bad {
						return true
					}
					key := filepath.Base(path) + ":" + id.Name
					if seen[key] {
						return true
					}
					seen[key] = true
					t.Errorf("%s: identifier %q carries the domain word %q — a rater is any key and so is an item",
						filepath.Base(path), id.Name, w)
					return true
				})
			}
		}
	}
}

func TestNoDomainWordsInStoredVocabularies(t *testing.T) {
	for _, v := range []string{
		string(ratings.KindRater), string(ratings.KindItem),
		string(ratings.MetricEuclidean), string(ratings.MetricCosine), string(ratings.MetricJaccard),
		ratings.SettingMetric, ratings.SettingSimilarities, ratings.SettingK, ratings.SettingOverlap,
	} {
		if w, bad := carriesDomainWord(v); bad {
			t.Errorf("stored value %q carries the domain word %q — a value outlives a rename", v, w)
		}
	}
}

func TestNoDomainWordsInTheBlueprintsRolesFieldsAndStoredValues(t *testing.T) {
	b, err := ratings.Blueprint()
	if err != nil {
		t.Fatalf("blueprint: %v", err)
	}
	report := func(kind, name string) {
		if w, bad := carriesDomainWord(name); bad {
			t.Errorf("blueprint %s %q carries the domain word %q — it belongs in a domain's own spec", kind, name, w)
		}
	}
	for _, o := range b.Objects {
		report("role", o.Role)
		if o.Name != "" || o.Table != "" {
			t.Errorf("blueprint role %q names a table, which is the domain's to choose", o.Role)
		}
		for _, f := range o.Fields {
			report("field", o.Role+"."+f.Name)
			for _, v := range f.Values {
				report("stored value", o.Role+"."+f.Name+"="+v)
			}
		}
		for _, idx := range o.Indexes {
			report("index", o.Role+"."+idx.Name)
		}
	}
}

func carriesDomainWord(s string) (string, bool) {
	low := strings.ToLower(s)
	for _, w := range domainWords {
		if strings.Contains(low, w) {
			return w, true
		}
	}
	return "", false
}
