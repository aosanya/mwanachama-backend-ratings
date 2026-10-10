# mwanachama-backend-ratings — open tasks

Open tasks only — 🚀 In Progress · 📋 Not Started · ⏸️ Blocked.

Research rows first. No build row is filed until the open questions are answered.

| Task | Title | Notes |
|------|-------|-------|
| RT1 | 📋 **Survey collaborative filtering as it applies here** | User-based vs item-based, the three metrics, neighbourhood size, overlap floors, cold start; what a Postgres-only implementation costs at 10⁴ raters × 10⁴ items. Output: one page in `2. design/` with the numbers RT-Q4/RT-Q5 need. |
| RT2 | ⏸️ **Decide the scale and the revision rule** | Blocked on RT-Q1 and RT-Q2. |
| RT3 | ⏸️ **Draft `ratings.blueprint.json` and the first domain spec** | Two roles, `rating` and `similarity`; `metric` as a declared enum; the indexes the incremental computation needs (`rating(item)`, `rating(rater)`, `similarity(a, score)`). Blocked on RT2 and RT-Q3. Two shipped domain specs so neutrality is exercised: a catalog of titles and something far from it. |
| RT4 | ⏸️ **Specify the extractor contract with `mwanachama-backend-context`** | What `reindex(partition, "ratings", keys)` hands over and in what batches; whether `rated` edges travel (RT-Q6); who owns the key ends (RT-Q7). Blocked on context's TR13 shape and shared's S59. |
| RT5 | 📋 **Define how recommendation quality is measured** | Hold out one rating per rater, predict it, score RMSE and recall@N against a public rating set so the metric and K are chosen on evidence. Lives in `4. qa/`. |
| RT6 | ⏸️ **Prototype `recommend` on SQLite and Postgres** | After RT3: load a public rating set, run the incremental computation, answer `recommend` for ten raters, time it. |
