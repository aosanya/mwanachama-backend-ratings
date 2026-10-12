# CLAUDE.md

Guidance for Claude Code working in this repository.

## Project: mwanachama-backend-ratings

Ratings and the similarity they imply. A `rating` is `(rater, item, value,
at)`; a `similarity` is `(a, b, metric, score, at)`, computed from co-rated
items by a declared metric. Module path
`github.com/aosanya/mwanachama-backend-ratings`.

**The research rows are done and the build has started.** The decisions are
on
[documentation/1. requirements/requirements.md](documentation/1.%20requirements/requirements.md)
and are settled; the build rows on
[documentation/3. implementation/todo.md](documentation/3.%20implementation/todo.md)
run in dependency order. Do not resolve an open question by guessing; ask.

## What is built (RT3–RT7 and RT9, 2026-10-10 to 2026-10-11)

- `ratings.blueprint.json` declares three roles — `rating`, `similarity`,
  `setting` — embedded and reached through `Blueprint()`, `LoadSpec(path)`
  and `ParseSpec(raw)`. Load a domain spec through those, never `spec.Load`.
- `spec/examples/titles.ratings.json` (a catalogue of titles, instance
  `cat1`) and `pantry.ratings.json` (a household rating recipes, instance
  `home1`) are the same module under two domains far enough apart that a
  domain word in the module would show. `TestShippedExamplesCoexist`
  migrates both into one database; `TestEveryExampleFitsTheTypes` builds a
  manager over each.
- `models/` carries `Rating`, `Similarity`, `Setting`, `Kind` and `Metric`,
  checked against the blueprint in `NewRatingsManager(db, *spec.Spec)` by
  `specstore.New`. `TestVocabularyMatchesTheBlueprint` holds the Go
  constants and the blueprint's enum values to each other.
- **Per-instance settings are rows, not a spec block.** `spec.Spec` decodes
  with unknown fields refused, so the `settings` sketch on the design page
  could not be carried by a domain spec. The third role, `setting`, is a
  key/value table the host writes on boot; `Settings(ctx)` merges its rows
  over `DefaultSettings()` (euclidean, rater-to-rater only, K 30, overlap
  3), and `PutSetting` holds `metric` and `similarities` to the blueprint's
  own enum values and `k`/`overlap` to whole numbers of at least 1. See
  [documentation/2. design/ratings-model.md](documentation/2.%20design/ratings-model.md).
- `validate.go` (RT4) reads `required`, an enum's `values` and `matches` off
  the spec and applies them on the way in, so a domain that widens `kind`
  gets the new value accepted with no Go change
  (`TestVocabularyComesFromTheSpec`). What stays in Go is what a spec cannot
  say — `value` in [0, 1], `score` in (0, 1], `overlap` a count — and each
  lives beside its type in `models/`. `Check(spec, role, v)` is the same
  validation without a database. `patterns.go` holds the `matches` registry,
  empty until a field needs one; a spec naming a pattern nobody supplies is
  an error, not a rule that quietly never runs.
- `rating.go`/`similarity.go`/`sides.go`/`metrics.go` (RT5): `PutRating`
  (one row per `(rater, item)`, a repeat replaces it), `GetRating`,
  `ListRatings` (by rater or item, never unscoped, capped at 500),
  `DeleteRating`, `ListSimilarities`. Every write runs the incremental step
  for each declared kind: one self-join fetches the co-rated value pairs for
  the raters of that item only, the metric runs in Go, and `place` keeps at
  most K rows per `(a, kind)`. A delete re-scores the pairs it can break and
  **refreshes** any key whose set shrank, from scratch, bounded by that key's
  own ratings. Mutation-checked: a write touches no row whose two keys did
  not both rate the written item (`TestAWriteTouchesOnlyRatersOfThatItem`,
  stepping clock). See the "As built" section of
  [ratings-model.md](documentation/2.%20design/ratings-model.md).
- `recommend.go` (RT6): `Recommend(ctx, rater, n)` is decision #9's one
  query over the K stored neighbours — `Σ score·value / Σ score` per item
  the rater has not rated, with the contributing neighbour count as
  `support`. Empty for a rater with no neighbours (#19); `n` defaults to 10,
  capped at 100. No centring and no minimum support: both would change the
  decided query; RT10 was closed by the owner on 2026-10-11 without a
  public-set run (the module is domain-agnostic, a deployment's own ratings
  are the data), so the decided defaults stand and an instance tunes them
  through its `setting` rows.
- `ratings.operations.json` and `routes/` (RT7): the five operations of
  decision #17 through `mwanachama-backend-shared/dispatch`, REST and MCP
  from one file; `AnonymousActions` is empty and `TestNoRouteIsAnonymous`
  pins it. The rater on `put`, `delete` and `recommend` is bound from the
  mount's `Caller`, never the request, and is absent from the tool schemas.
  Two wire facts: the fifth action is `ratings.recommendation.list`
  (`dispatch.ActionPattern` needs three segments), and a key in an address
  is percent-encoded because it carries slashes. The host mount is RT11.
- `Provision(db, spec)` (RT9) is the whole storage story the host calls on
  boot — `spec.Migrate`, refused for a spec of another module — and
  `cmd/ddl <spec.json> [postgres|sqlite]` prints the same DDL for review or
  for a deployment's own migration tooling. `postgres_integration_test.go`
  (`//go:build integration`, `POSTGRES_URL`, `make test-pg`) provisions a
  random instance, checks `value` lands as `double precision`, and runs the
  self-join, top-K, `recommend` and the delete-below-floor path on real
  Postgres. Written, compiled under the tag, **not run**: the routine's
  environment has no Postgres.
- `measure_*_test.go` (RT10, 2026-10-11) runs RT2's protocol under
  `//go:build measure`: `make measure` with `RATINGS_MEASURE_DIR` naming a
  directory holding `u.data` as `rater item value timestamp` rows on a
  five-point scale loads every rating through `PutRating` per grid cell and
  prints the grid, the baselines and the design page's variant columns.
  Never part of `go test ./...`; proven on a synthetic set only. It is for
  measuring a real instance's own exported ratings — RT10's public-set run
  was closed by the owner as not needed. See
  [measurement-procedure.md](documentation/4.%20qa/measurement-procedure.md).
- **Nothing goes to context** (requirements #22, 2026-10-12): ratings and
  `mwanachama-backend-context` are completely separate. RT8's `Edges` was
  built and deleted the same day.
- `domain_agnostic_test.go` parses `models/`, `routes/` and the root
  package, every stored vocabulary and the blueprint, and fails on a domain
  word. The words include both example domains' nouns (`title`, `review`,
  `recipe`, `verdict`), so the examples are what would catch a leak.

## What is already known

- **Split out of `mwanachama-backend-context` on 2026-10-10.** That module
  holds weighted edges between keys and knows nothing about what it maps; a
  ratings matrix is tabular and similarity is a join plus a sum, which is
  SQL's job. Two repos, and no contract between them (#22).
- **Postgres, declared domain, like every sibling** — `ratings.blueprint.json`
  through `mwanachama-backend-shared/spec` and `specstore`; no row structs,
  no `AutoMigrate`, no ArangoDB. `mwanachama-backend-catalog` is the shape
  to follow.
- **A rater is always an actor**, bound from the caller (requirements #20). **An item is any key.** Both are raw strings in the
  standard's form `<module>/<instance>/<identity>` (developer's
  `architecture-spec-driven-modules.md` §9), owned by other modules. This
  module stores them and never resolves what they mean.
- **Similarity is computed incrementally** — for the rater (or item) whose
  ratings changed, over its co-raters only, never all pairs. Efficiency is a
  requirement, not a nicety.
- **The metric is declared**, from a closed set the module supplies:
  `euclidean` (asked for first), `cosine`, `jaccard`.
- **It feeds nothing.** It never sends anything to
  `mwanachama-backend-context`, never touches ArangoDB and triggers nothing
  (requirements #18, #22). It is mounted with a catalog instance.
- **It names no domain.** Not "movie", not "member", not "user" — `rater`
  and `item`. `domain_agnostic_test.go` enforces it over identifiers, the
  blueprint and every stored enum value, as the siblings do.

## Conventions

- Four-phase `documentation/` layout — see
  [documentation/README.md](documentation/README.md).
- `go test ./...` (sqlite via `glebarez/sqlite`) verifies a change here;
  what sqlite cannot stand in for goes under `//go:build integration`, gated
  on `POSTGRES_URL`, run by `make test-pg`. `make build`, `make vet` and
  `gofmt -l .` are the rest of the pre-push check.
- No Go file over 300 lines; split by responsibility.

## Code comments

Write code with no comments. Not one-liners above a function, not section
banners, not doc comments on exported symbols, not "why" notes next to a
tricky line. A name, a type, or a smaller function carries it instead.

Anything that genuinely needs explaining goes in this repo's `documentation/`
folder, under the phase it belongs to (`1. requirements`, `2. design`,
`3. implementation`, `4. qa`) — never inline.

**Why:** inline prose drifts out of sync with the code, duplicates what
`documentation/` already owns, and buries the explanation where nobody
looking for it will search.

**How to apply:**

- New code ships without comments. If a line seems to need one, rename or
  split until it doesn't.
- Touching code that already has comments: strip the ones in the code you are
  changing. Do not sweep untouched files unless asked.
- If the reasoning matters, add or update the matching `documentation/` page
  in the same change and leave nothing behind in the source.
- Machine-read directives are not comments and stay: build tags, `//go:embed`,
  `//go:generate`, linter pragmas (`//nolint`, `// eslint-disable-next-line`,
  `// ignore:`), license headers, codegen "do not edit" banners, and generated
  files as a whole.
- Commit messages, PR descriptions, and test names carry the narration that
  used to go in comments.

This rule is repeated verbatim in every mwanachama repo's `CLAUDE.md` so that
it reaches sessions that do not load this machine's user-level config —
scheduled cloud routines, other machines, and other agent harnesses.
