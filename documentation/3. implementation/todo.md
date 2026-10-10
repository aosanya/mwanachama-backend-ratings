# mwanachama-backend-ratings — open tasks

Open tasks only — 🚀 In Progress · 📋 Not Started · ⏸️ Blocked.

Research first, then the build in dependency order. Every decision the build
rests on is in `requirements.md`; a row names the ones it carries.

## Research

| Task | Title | Notes |
|------|-------|-------|
| RT1 | 📋 **Survey collaborative filtering as it applies here** | User-based vs item-based, the three metrics, neighbourhood size, overlap floors, cold start (RT-Q8); what a Postgres-only implementation costs at 10⁴ raters × 10⁴ items. Output: one page in `2. design/` that gives RT2 its starting numbers. |
| RT2 | 📋 **Define how recommendation quality is measured** | Hold out one rating per rater from a public rating set, predict it, score RMSE and recall@N. The same run sets the metric, K (#13) and the overlap floor (#14) from evidence. Output: `2. design/measurement.md` and the `4. qa/` procedure. |

## Build

| Task | Title | Depends on | Notes |
|------|-------|------------|-------|
| RT3 | 📋 **`ratings.blueprint.json`, two domain specs, `models/`, `NewRatingsManager(db, *spec.Spec)`** | — | Decisions #2, #3, #7, #10–#14. Two roles and their indexes as sketched on the design page; `spec/examples/titles.ratings.json` and `pantry.ratings.json`; where the per-instance `settings` live, against what `spec` lets a domain carry. `TestEveryExampleFitsTheTypes`, `TestVocabularyMatchesTheBlueprint`, `domain_agnostic_test.go` land with it, as in every sibling. |
| RT4 | 📋 **`validate.go` and the [0, 1] bound** | RT3 | `required` and enum `values` read off the spec; the value bound in Go beside the type. `Check(s, role, v)` without a database. |
| RT5 | 📋 **Rating put / list / delete, and the incremental similarity step** | RT4 | #5, #11–#14: one row per `(rater, item)`, replaced on repeat; on every write the three-step computation over the item's raters with the overlap floor and top-K maintenance; item-to-item when the instance declares it. Mutation-checked: a write must touch only raters sharing that item. |
| RT6 | 📋 **`recommend(rater, n)`** | RT5 | #9: the one query on the design page over the K neighbours, excluding the rater's own items. RT-Q8 decides the empty case. |
| RT7 | 📋 **`ratings.operations.json`, `routes/`, MCP tools** | RT6 | #17: five operations through `dispatch`; rater bound from the caller; `AnonymousActions` empty; `TestNoRouteIsAnonymous`. |
| RT8 | 📋 **`Edges(ctx, partition, keys)` — the extractor toward context** | RT5, context's TR13, shared's S59 | #8, #15: `similar_taste` always, `rated` only when the host declared it. Blocked on the write-report shape (S59) and on context existing to receive it. The host calls it, batched (#18). |
| RT9 | 📋 **`Provision`, `cmd/ddl`, `Makefile`, `postgres_integration_test.go`** | RT3 | `spec.Migrate` is the whole storage story; `//go:build integration` gated on `POSTGRES_URL`; `build`/`test`/`test-pg`/`vet`/`clean`. |
| RT10 | 📋 **Measure against a public rating set** | RT2, RT6 | Load the set through `put`, run RT2's procedure, record the numbers in `4. qa/`, and set the defaults for metric, K and overlap from them. |
| RT11 | ⏸️ **Mount in `mwanachama-wakala-api`** | RT7, RT8 | Cross-repo. `RATINGS_INSTANCE` → `SpecFor`, provisioned on boot, mounted beside the other declared domains per that repo's decision #22; the caller→rater mapping; the batched reindex trigger toward context (#18): per-pod pending set, flush at ~30 s or N raters. |
