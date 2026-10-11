# QA

The first code landed with RT3 (2026-10-10); what it pins is listed below
with the rest.

## Two tiers

```sh
go test ./...        # in-memory SQLite (glebarez/sqlite), no database needed
make test-pg         # -tags=integration, with POSTGRES_URL set
```

## What will be measured (RT2, run by RT10)

Hold out each rater's latest rating from MovieLens ml-100k, predict it through
`recommend`'s formula, and score **RMSE**, **MAE** and **coverage** on the
prediction and **hit rate @10** on the recommended list. The same run chooses
the metric (#6), the neighbourhood size K (#13) and the overlap floor (#14),
so none of those is a guess. The step list is
[measurement-procedure.md](measurement-procedure.md); the reasoning is
[`2. design/measurement.md`](../2.%20design/measurement.md); the numbers will
land in `measurements.md`.

## What is pinned

Since RT3:

- the module names no domain, in identifiers, blueprint and stored values
  (#7) — `domain_agnostic_test.go`;
- both shipped domain specs migrate into one database and fit the Go types —
  `TestShippedExamplesCoexist`, `TestEveryExampleFitsTheTypes`;
- the Go constants and the blueprint's enum values agree —
  `TestVocabularyMatchesTheBlueprint`;
- one row per `(rater, item)` and per `(a, kind, b)` is the primary key
  (#11, #13) — `TestOneRowPerRaterAndItemIsTheKey`;
- a setting is held to the blueprint's vocabulary and a refused put stores
  nothing — `settings_test.go`;
- `value` outside [0, 1], `score` outside (0, 1], a negative `overlap`, an
  undeclared `kind` or `metric` and a missing required field are refused by
  `Check`, and a kind a domain declares is accepted — `validate_test.go`
  (RT4).

- a rating write recomputes similarity only for raters sharing that item
  (#5) — `TestAWriteTouchesOnlyRatersOfThatItem`, with a stepping clock so
  an untouched row's `at` is exact; the mutant that recomputes every pair
  fails it (RT5);
- every metric scores in (0, 1] and a zero cosine writes nothing (#6) —
  `metrics_test.go`, `TestCosineIsDirectionClampedToTheUnitInterval` (RT5);
- a pair under the overlap floor is never linked, and a delete that takes
  one under it removes both rows (#14) —
  `TestAWriteLinksEachCoRaterAboveTheFloorBothWays`,
  `TestADeleteBelowTheFloorRemovesThePairBothWays`; the off-by-one mutant
  fails the second (RT5);
- at most K rows per key and kind, the weakest displaced, and a shrunk set
  refilled from candidates never stored (#13) —
  `TestTopKIsMaintainedAndTheWeakestIsDisplaced`,
  `TestADeleteRefreshesAShrunkSetFromCandidatesNeverStored` (RT5);
- item-to-item runs only when the instance declares it (#12) —
  `TestItemToItemRunsOnlyWhenTheInstanceDeclaresIt` (RT5).

- `recommend` never returns an item the rater has rated (#9) —
  `TestRecommendNeverReturnsAnItemTheRaterHasRated`; the mutant that drops
  the anti-join fails it (RT6);
- a rater with no neighbours gets an empty list and no error (#19) —
  `TestARaterWithNoNeighboursGetsAnEmptyListNotAnError` (RT6).

- no route is anonymous and the five declared operations are the whole
  surface (#17) — `routes/routes_test.go`: `TestNoRouteIsAnonymous`,
  `TestTheFiveDeclaredOperationsAreTheWholeSurface`;
- the rater is the caller, not the body, on `put`, `delete` and `recommend`,
  and no tool offers it (#17) — `TestPutRatingTakesTheRaterFromTheCallerNotTheBody`,
  `TestDeleteAndRecommendAreTheCallersOwn`, `TestTheRaterIsNotAToolArgument`;
- a key in an address is percent-encoded and every sentinel keeps its
  declared status — `TestErrorsKeepTheirDeclaredStatus`,
  `TestListSimilaritiesOverHTTP` (RT7).

- `Provision` migrates idempotently, writes the per-instance registry and
  refuses another module's spec — `provision_test.go` (RT9).

## The Postgres tier (RT9)

`postgres_integration_test.go` runs only under `-tags=integration` with
`POSTGRES_URL` set (`make test-pg`); without the variable it skips. It
provisions the titles example at a random instance, drops its tables and
registry on cleanup, and pins what SQLite cannot: `value` is `double
precision`; the co-rated self-join, the top-K maintenance, `recommend`'s
grouped division and the delete-below-floor path give the same figures as
the sqlite tests. **As of 2026-10-11 it has never been run** — the backlog
routine's environment has no Postgres — so the first `make test-pg` against
a real cluster is still owed, and anything it finds is a bug to file on the
board.

Nothing else is left to pin from the built rows; RT8 (`Edges`) and RT11
(the mount) will add theirs.
