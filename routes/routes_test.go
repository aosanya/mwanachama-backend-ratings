package routes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	ratings "github.com/aosanya/mwanachama-backend-ratings"
	"github.com/aosanya/mwanachama-backend-ratings/routes"
	"github.com/aosanya/mwanachama-backend-shared/dispatch"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

type callerKey struct{}

const raterHeader = "X-Test-Rater"

func callerFromContext(ctx context.Context) string {
	s, _ := ctx.Value(callerKey{}).(string)
	return s
}

func newManager(t *testing.T) (ratings.RatingsManager, context.Context) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	s, err := ratings.LoadSpec(filepath.Join("..", "spec", "examples", "titles.ratings.json"))
	if err != nil {
		t.Fatalf("load spec: %v", err)
	}
	if err := spec.Migrate(db, s); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	rm, err := ratings.NewRatingsManager(db, s)
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	return rm, context.Background()
}

func mount() routes.Mount { return routes.Mount{Caller: callerFromContext} }

func newServer(t *testing.T) (http.Handler, ratings.RatingsManager, context.Context) {
	t.Helper()
	rm, ctx := newManager(t)
	mux := http.NewServeMux()
	for _, r := range routes.OperatorRoutesFor(rm, mount()) {
		handler := r.Handler
		mux.HandleFunc(r.Method+" "+r.Path, func(w http.ResponseWriter, req *http.Request) {
			ctx := context.WithValue(req.Context(), callerKey{}, req.Header.Get(raterHeader))
			handler(w, req.WithContext(ctx))
		})
	}
	return mux, rm, ctx
}

func do(t *testing.T, h http.Handler, rater, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, path, nil)
	} else {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
	}
	if rater != "" {
		r.Header.Set(raterHeader, rater)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

const (
	me    = "actor/main/me"
	peer  = "actor/main/peer"
	other = "actor/main/other"
)

func item(n string) string { return "catalog/cat1/" + n }

func at(key string) string { return url.PathEscape(key) }

func seedTaste(t *testing.T, rm ratings.RatingsManager, ctx context.Context) {
	t.Helper()
	for i, it := range []string{"x", "y", "z"} {
		v := 0.2 + 0.2*float64(i)
		for _, r := range []ratings.Rating{{Rater: me, Item: item(it), Value: v}, {Rater: peer, Item: item(it), Value: v + 0.1}} {
			if _, err := rm.PutRating(ctx, r); err != nil {
				t.Fatalf("seed: %v", err)
			}
		}
	}
	if _, err := rm.PutRating(ctx, ratings.Rating{Rater: peer, Item: item("p"), Value: 0.9}); err != nil {
		t.Fatalf("seed: %v", err)
	}
}

func TestNoRouteIsAnonymous(t *testing.T) {
	rm, _ := newManager(t)
	split := routes.SplitFor(rm, mount())
	if len(split.Anonymous) != 0 {
		t.Errorf("anonymous routes = %v, want none", split.Anonymous)
	}
	if len(split.Gated) != 5 {
		t.Errorf("gated routes = %d, want the five declared operations", len(split.Gated))
	}
	if len(routes.AnonymousActions) != 0 {
		t.Errorf("AnonymousActions = %v, want empty", routes.AnonymousActions)
	}
}

func TestTheFiveDeclaredOperationsAreTheWholeSurface(t *testing.T) {
	rm, _ := newManager(t)
	want := map[string]string{
		"PUT /ratings/{item}":    "ratings.rating.put",
		"GET /ratings":           "ratings.rating.list",
		"DELETE /ratings/{item}": "ratings.rating.delete",
		"GET /similar/{key}":     "ratings.similarity.list",
		"GET /recommend":         "ratings.recommendation.list",
	}
	for _, r := range routes.Routes(rm) {
		action, declared := want[r.Method+" "+r.Path]
		if !declared || action != r.Action {
			t.Errorf("unexpected route %s %s → %s", r.Method, r.Path, r.Action)
		}
		delete(want, r.Method+" "+r.Path)
	}
	for addr := range want {
		t.Errorf("%s is declared but not routed", addr)
	}
	tools := routes.Tools(rm)
	if len(tools) != 5 {
		t.Errorf("tools = %d, want 5", len(tools))
	}
	s, err := routes.Spec()
	if err != nil {
		t.Fatal(err)
	}
	if problems := dispatch.UnmappedSentinels(s, map[string]error{}, ".."); len(problems) > 0 {
		t.Errorf("sentinels the spec maps to no status: %v", problems)
	}
}

func TestPutRatingTakesTheRaterFromTheCallerNotTheBody(t *testing.T) {
	h, rm, ctx := newServer(t)
	rec := do(t, h, me, http.MethodPut, "/ratings/"+at(item("x")), `{"rater":"`+other+`","item":"catalog/cat1/elsewhere","value":0.7}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("put: %d %s", rec.Code, rec.Body.String())
	}
	var got ratings.Rating
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Rater != me || got.Item != item("x") || got.Value != 0.7 {
		t.Errorf("the address and the caller outrank the body: %+v", got)
	}
	if _, err := rm.GetRating(ctx, other, item("x")); err == nil {
		t.Error("a rating was stored in the body's name")
	}
	if _, err := rm.GetRating(ctx, me, "catalog/cat1/elsewhere"); err == nil {
		t.Error("a rating was stored at the body's item")
	}

	list := do(t, h, me, http.MethodGet, "/ratings?rater="+me, "")
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), item("x")) {
		t.Errorf("list: %d %s", list.Code, list.Body.String())
	}
	if raw := do(t, h, me, http.MethodPut, "/ratings/"+item("y"), `{"value":0.1}`); raw.Code != http.StatusNotFound {
		t.Errorf("a key's slashes must be percent-encoded in the address; unencoded answered %d", raw.Code)
	}
	if list.Code != http.StatusOK {
		t.Errorf("list: %d %s", list.Code, list.Body.String())
	}
}

func TestDeleteAndRecommendAreTheCallersOwn(t *testing.T) {
	h, rm, ctx := newServer(t)
	seedTaste(t, rm, ctx)

	rec := do(t, h, me, http.MethodGet, "/recommend?n=5", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), item("p")) {
		t.Fatalf("recommend for me: %d %s", rec.Code, rec.Body.String())
	}
	if rec := do(t, h, other, http.MethodGet, "/recommend", ""); rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Errorf("a stranger with no neighbours gets an empty list: %d %q", rec.Code, rec.Body.String())
	}

	if rec := do(t, h, other, http.MethodDelete, "/ratings/"+at(item("x")), ""); rec.Code != http.StatusNotFound {
		t.Errorf("a stranger deleting my rating: %d, want 404 because it is not theirs", rec.Code)
	}
	if rec := do(t, h, me, http.MethodDelete, "/ratings/"+at(item("x")), ""); rec.Code != http.StatusNoContent {
		t.Errorf("deleting my own: %d", rec.Code)
	}
	if _, err := rm.GetRating(ctx, me, item("x")); err == nil {
		t.Error("my rating is still there")
	}
}

func TestListSimilaritiesOverHTTP(t *testing.T) {
	h, rm, ctx := newServer(t)
	seedTaste(t, rm, ctx)
	rec := do(t, h, me, http.MethodGet, "/similar/"+at(item("ignored"))+"?kind=item", "")
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Errorf("item kind is not declared, so nothing: %d %s", rec.Code, rec.Body.String())
	}
	rec = do(t, h, me, http.MethodGet, "/similar/"+at(me), "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), peer) {
		t.Errorf("my neighbours: %d %s", rec.Code, rec.Body.String())
	}
	if rec := do(t, h, me, http.MethodGet, "/similar/"+at(me)+"?kind=friend", ""); rec.Code != http.StatusBadRequest {
		t.Errorf("an undeclared kind: %d %s", rec.Code, rec.Body.String())
	}
}

func TestErrorsKeepTheirDeclaredStatus(t *testing.T) {
	h, _, _ := newServer(t)
	for _, tc := range []struct {
		method, path, body string
		want               int
	}{
		{http.MethodPut, "/ratings/" + at(item("x")), `{"value":1.5}`, http.StatusBadRequest},
		{http.MethodPut, "/ratings/" + at(item("x")), `{"value":0.5,"stars":5}`, http.StatusBadRequest},
		{http.MethodGet, "/ratings", "", http.StatusBadRequest},
		{http.MethodDelete, "/ratings/" + at(item("never")), "", http.StatusNotFound},
	} {
		if rec := do(t, h, me, tc.method, tc.path, tc.body); rec.Code != tc.want {
			t.Errorf("%s %s %s: %d, want %d (%s)", tc.method, tc.path, tc.body, rec.Code, tc.want, rec.Body.String())
		}
	}
}

func TestTheRaterIsNotAToolArgument(t *testing.T) {
	rm, _ := newManager(t)
	for _, tool := range routes.Tools(rm) {
		var schema struct {
			Properties map[string]any `json:"properties"`
			Required   []string       `json:"required"`
		}
		if err := json.Unmarshal(tool.InputSchema, &schema); err != nil {
			t.Fatalf("%s: %v", tool.Name, err)
		}
		if _, offered := schema.Properties["rater"]; offered && tool.Action != "ratings.rating.list" {
			t.Errorf("%s offers a rater argument, which only the mount may supply", tool.Name)
		}
		if _, offered := schema.Properties["at"]; offered {
			t.Errorf("%s offers the store-owned at", tool.Name)
		}
		if tool.Description == "" {
			t.Errorf("%s has no description", tool.Name)
		}
	}
}
