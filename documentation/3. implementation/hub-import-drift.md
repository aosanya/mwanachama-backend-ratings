# Hub import drift

What stands between this repository and the hub requiring it as a tagged module in place of the copy under `projects/hub-steward/code/wakala-api/modules/ratings`. Written on 2026-11-19 at this repository's `45fa9e3` (branch `main`, in step with `origin/main`, no tag) against the hub's `dev` branch at `43f22b26`; the last commit that touches the module there is `b4660864`, the copy-in of 4 November. The module has no uncommitted change in the hub.

## How the comparison was run

This repository has no `scripts/hubdiff` and no `hub-copy-diff.md`, so the comparison is the same one by hand: every file under the hub's `modules/ratings`, with `wakalaserver/modules/ratings` rewritten to `github.com/aosanya/mwanachama-backend-ratings` and `wakalaserver/modules/shared` rewritten to `github.com/aosanya/mwanachama-backend-shared`, set against the file of the same path here. Comments and layout do not show; only code does.

## Verdict in three lines

- The module code is the same on both sides. Twenty-three hub files; twenty-two are byte-identical after the import path is rewritten, and the twenty-third (`settings.go`) differs only in where gofmt puts the `gorm.io/gorm` import after the rewrite.
- Nothing is hub only inside the module, so there is nothing to port into this repository. The hub has no test file in the module; every test, the `cmd/ddl` tool and the two shipped example domains are sibling only and stay here.
- What the hub still owns is wiring outside the module (HTTP layer, mounts, the agency-domain declaration). It stays in the hub and changes only its import lines. The import is held up by the shared module, not by this module (see What blocks it).

## The drift table

"Hub only" means the hub has it and this repository does not. "Sibling only" means the reverse. "Both" means both have it and they differ beyond the import path.

| # | Difference | Side | What it does | Disposition |
|---|---|---|---|---|
| 1 | The module's Go and JSON files: `blueprint.go`, `errors.go`, `metrics.go`, `operations.go`, `patterns.go`, `provision.go`, `rating.go`, `ratings.blueprint.json`, `ratings.operations.json`, `ratings_manager.go`, `recommend.go`, `sides.go`, `similarity.go`, `store.go`, `validate.go`, `models/{limits,rating,setting,similarity,vocabulary}.go`, `routes/{routes,tools}.go` | Both, identical | Same behaviour: five operations, three roles, the same blueprint and the same operations file. | None. The hub gets them from the tag. |
| 2 | `settings.go`: the `gorm.io/gorm` import sits in the third-party group here and in the first group in the hub | Both | None. A formatting result of the import rewrite. | Keep this repository's. No change. |
| 3 | Tests: `blueprint_test.go`, `domain_agnostic_test.go`, `examples_test.go`, `measure_*_test.go` (five files), `metrics_test.go`, `postgres_integration_test.go`, `provision_test.go`, `rating_test.go`, `recommend_test.go`, `settings_test.go`, `similarity_test.go`, `testdb_test.go`, `validate_test.go`, `models/rules_cover_test.go`, `routes/routes_test.go` | Sibling only | The hub copy has none. | Stay here. The hub loses nothing by importing. |
| 4 | `cmd/ddl/main.go` | Sibling only | Prints the DDL of a ratings spec for a hand-run migration. | Stays here. |
| 5 | `spec/examples/titles.ratings.json`, `spec/examples/pantry.ratings.json` | Sibling only | The two example domains the domain-agnostic tests migrate side by side. | Stay here. |
| 6 | `Makefile`, `README.md`, `CLAUDE.md`, `go.mod`, `go.sum`, `documentation/` | Sibling only | The repository's own build, notes and design pages. The hub's `go.mod` carries the module's dependencies itself. | Stay here. |
| 7 | `internal/specs/wakala.ratings.json` in the hub (`module` ratings, `domain` agency, `instance` wakala) | Hub only, outside the module | The hub's own declaration of what a ratings mount holds. It is configuration of one deployment, not part of the module. Checked: this repository's `LoadSpec`, `Provision` and `NewRatingsManager` accept it unchanged on in-memory SQLite. | Stays in the hub. |
| 8 | `cmd/server/ratings.go` (`ratingsAttachedTo`, `buildRatings`: reads what the catalog spec attaches, parses the ratings declaration, provisions and builds the manager) | Hub only, outside the module | Server start-up wiring. Uses `ratings.Module`, `ParseSpec`, `Provision`, `NewRatingsManager` and `RatingsManager`, all exported here. | Stays in the hub; the import line changes. |
| 9 | `internal/api/http/catalog_ratings_routes.go`, `serverwide_agency.go`, `deps.go` (HTTP layer: `Deps.Ratings`, the mount, `ratingsroutes.OperatorRoutesFor`, `Mount{Authorize, Caller}`) | Hub only, outside the module | Registers the routes of the module on the hub's mux behind the hub's auth and its fail-closed authorizer. | Stays in the hub; the import lines change. |
| 10 | `internal/accounts/router.go` (`Router.RatingsManager`), `internal/agencies/registry.go`, `record_writers.go`, `record_writers_ratings.go` and their tests | Hub only, outside the module | Per-agency lookup of a ratings manager and the record writers that fill it. Not part of the module. | Stay in the hub; the import lines change. |
| 11 | The hub's permission catalogue and tool filters | Hub only, outside the module | A search of the hub's `internal/` and `cmd/` finds no read of `ratings.operations.json` and no ratings entry in a catalogue file; the tools of the module are listed from `routes/tools.go` at run time. | Nothing to move. Re-check when the hub imports, because the hub's roster test data under `internal/api/http/testdata/mcp_roster` changes only if the tool set does, and it does not here. |
| 12 | Cosmetic: none beyond row 2 | Both | None. | No change. |

## What the hub must do to require the tagged module

1. In `code/wakala-api/go.mod`, add `require github.com/aosanya/mwanachama-backend-ratings vX.Y.Z` (the hub has no require of it today, and no `replace`).
2. Change the import `wakalaserver/modules/ratings` to `github.com/aosanya/mwanachama-backend-ratings`, and its `/routes` the same way. That is ten files outside the module: `cmd/server/ratings.go`, `internal/accounts/router.go`, `internal/agencies/record_writers.go`, `record_writers_ratings.go`, `record_writers_ratings_test.go`, `registry.go`, `internal/api/http/catalog_ratings_routes.go`, `deps.go`, `serverwide_agency.go` and `serverwide_per_agency_tools_test.go`. Most alias the root package as `mwanachamaratings` or `ratings`, so the alias stays. The package is `mwanachamaratings`; `routes` is named `routes`, so it needs the `ratingsroutes` alias the hub already uses.
3. Delete `modules/ratings`.
4. Build with the shared module resolved one way (blocker 1).
5. The module path in this repository's `go.mod`, `github.com/aosanya/mwanachama-backend-ratings`, equals the path of its remote. The module is at the repository root, so the tag is plain `vX.Y.Z`.

## What blocks it

1. **Two copies of the shared module.** The hub's copy of the module imports `wakalaserver/modules/shared/...`; this repository imports `github.com/aosanya/mwanachama-backend-shared/...` (`dispatch`, `httpwire`, `spec`, `specstore`). They are different Go packages with the same shape, so a `spec.Spec` or a manager passed between the hub's packages and this module does not compile until the hub requires the shared repository too and drops its own copy. The shared repository carries uncommitted work of its own at the time of writing and has to settle first.
2. **No usable shared version in this repository's `go.mod`.** It requires `mwanachama-backend-shared v0.0.0` and a `replace` to `../mwanachama-backend-shared`. A `replace` in a dependency is ignored by whoever requires it, so a consumer cannot resolve `v0.0.0`. The shared repository has tags `v0.1.0` and `v0.2.0`. So: a shared tag, then here `require ... vA.B.C` and the `replace` removed.
3. **Three tests here fail against the shared repository as committed.** `go test ./...` on in-memory SQLite passes `models` and `routes` and fails three tests of the root package, none of them touched by this work: `TestShippedExamplesCoexist`, `TestShippedExampleMigratesIdempotently` and `TestProvisionMigratesIdempotentlyAndRegistersEveryTable`. They pin the table count of a migrated spec to the number of declared objects (three, or six for two domains). Since 2026-10-29 shared's `spec.Migrate` also creates a `_test` copy of each object's table (`spec.TestSuffix`), so the database holds twice as many tables. The hub's own shared copy has the same suffix. The pins are stale, not the code. Not fixed here: updating a pinned count is held until it is decided whether the pins should count live tables only or live and `_test` tables together.
4. **Unpushed and untagged.** This repository is level with `origin/main` and has no tag. Nothing is pushed or tagged without the owner's yes.

Not a blocker: unexported helpers (the hub code outside the module uses only exported names), package names (the alias already exists), `replace` in the hub (not needed once both modules are tagged), the Dockerfile (it downloads from the public module proxy and both repositories are public).

## Questions for the owner

None from the comparison: there is no operation for which the two sides behave differently. The one open choice is the stale table-count pins of blocker 3.
