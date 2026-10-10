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
  nothing — `settings_test.go`.

Still to pin, with the rows that build them:

- a rating write recomputes similarity only for raters sharing that item
  (#5) — RT5;
- every score lands in (0, 1] for every metric (#6) — RT5;
- `recommend` never returns an item the rater has rated (#9) — RT6.
