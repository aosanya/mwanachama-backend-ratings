# Requirements — mwanachama-backend-ratings

## Problem

Some things are rated: a rater gives an item a value on a scale. From enough
of those, two raters with alike values are alike in taste, and what one rated
highly is worth showing the other. That is collaborative filtering, and it is
tabular — a matrix of raters by items, queried by joins and sums — which is
why it does not belong inside a graph store.

## Vision

Store ratings as declared objects on Postgres, compute similarity
incrementally by a declared metric, answer "what would this rater rate
highly" directly, and hand `similar_taste` to
`mwanachama-backend-context` so a walk there can blend taste with
co-access, structure and an agent's assertions in one result.

## Decided (2026-10-10, the day it split from context)

| # | Decision |
|---|----------|
| 1 | **A separate repo from `mwanachama-backend-context`.** That module knows almost nothing about what it maps (its #18); a rating is a value and a metric is maths over values, which would have put knowledge into the graph. Its decisions #59 and #60 were withdrawn into this repo. |
| 2 | **Postgres, as a declared domain.** `ratings.blueprint.json` through `mwanachama-backend-shared/spec` and `specstore`, the shape every sibling takes. No row structs, no `AutoMigrate`, no second database. |
| 3 | **Two objects.** `rating(rater, item, value, at)` and `similarity(a, b, metric, score, at)`. |
| 4 | **A rater is any key.** A person, a chapter, an agency, a cohort — a raw string in the standard's `<module>/<instance>/<identity>` form, owned elsewhere. An item is the same. The module stores the strings and never resolves them. |
| 5 | **Similarity is computed incrementally**, for the rater whose ratings changed, over the raters who share an item with it, never all pairs. The same rule, with the sides flipped, gives item-to-item similarity from who rated both. |
| 6 | **The metric is declared** from a closed set the module supplies — `euclidean` (asked for first), `cosine`, `jaccard` — the way a sibling supplies `patterns.go`'s `e164`. Euclidean distance over co-rated items punishes two raters who agree but rate on different scales; cosine is the declarable alternative. Similarity is `1 / (1 + distance)` for a distance metric, so every score lands in (0, 1]. |
| 7 | **It names no domain.** `rater`, `item`, `value`, `score` — never movie, member, user, product. Enforced by a `domain_agnostic_test.go` over identifiers, the blueprint and stored enum values. |
| 8 | **Toward `mwanachama-backend-context` it is an extractor.** On that module's `reindex(partition, "ratings", [rater])` it emits `similar_taste` edges between rater keys with `weight = score`. The graph gets taste; this module keeps the numbers. |
| 9 | **It answers on its own too.** `recommend(rater, n)` — items rated highly by the most similar raters that this rater has not rated — is one SQL query here, no graph needed. The graph is for blending taste with what it alone holds. |

## Decided on the owner's "document it fully" (2026-10-10)

The seven questions below were each taken on the recommended option so the
repo could be documented through to its board. Each is one line to reverse:
say so and the row is struck through.

| # | Decision |
|---|----------|
| 10 | **`value` is a float in [0, 1]; the host normalises on the way in.** A 1–5 star arrives as 0.0–1.0, a thumb as 0 or 1; anything outside the range is refused. Every metric works unchanged, `recommend`'s "rated highly" means one thing, and the module never learns what the host's scale was. (RT-Q1) |
| 11 | **A repeated rating replaces the earlier one.** One row per `(rater, item)`, `value` and `at` written in place. The module is a matrix, not a log; if taste drift ever needs measuring, a `rating_event` object is one blueprint addition away, and nothing here forecloses it. (RT-Q2) |
| 12 | **Which similarities run is declared per instance**, `similarities: ["rater"]`, `["item"]` or both; the default is rater-to-rater, which is what `recommend` reads. Item-to-item is switched on by the host that needs "those who rated this also rated", and costs a second computation per write. (RT-Q3) |
| 13 | **Top-K neighbours per key, K = 30 by default.** The incremental step maintains the set: a new score enters if it beats the K-th, and the one it displaces is deleted. Storage and every later query are bounded by K, not by the catalog. K is RT5's to set from measurement. (RT-Q4) |
| 14 | **An overlap floor of 3 co-rated items** before a similarity is written at all; two raters sharing one item have a perfect, meaningless score. Also RT5's number to set. (RT-Q5) |
| 15 | **Only `similar_taste` travels to context by default.** `rated` edges — one per rating, the matrix duplicated into the graph — travel only when the host's `context.<domain>.json` declares a `rated` predicate, which is the host saying it wants a walk to cross rater → item. Efficiency and context's "knows nothing" rule both argue for the default. (RT-Q6) |
| 16 | **The rater's and item's owners implement `StableIdentity`/`Resolve` for their ends; this module implements neither for them.** It stores raw strings and passes them through. A `similar_taste` edge's ends are fetchable in a walk because accounts and catalog answer for them (shared's S56), not because this module does. Its own rows are never referenced from outside, so it exposes nothing of its own under S56 until something needs it. (RT-Q7) |
| 17 | **Five declared operations, `AnonymousActions` empty.** `ratings.rating.put`, `ratings.rating.list`, `ratings.rating.delete`, `ratings.similarity.list`, `ratings.recommend` — all in `ratings.operations.json`, REST and MCP through `dispatch`, gated by the host's authorizer. The rater on a `put` is a key the host supplies from its caller, never a body field the caller names (`from: caller` through the host's own mapping), so nobody rates in someone else's name. The extractor hook toward context is a library call, not an operation. |

## Open questions for the owner

Answered — struck through — or still open.

| # | Question | Why it matters |
|---|----------|----------------|
| ~~RT-Q1~~ | **Where is the scale declared** (1–5, 0–10, thumbs)? A domain spec may set only a default on a declared field, not a rule, so the scale is either a module-level `min`/`max` on `value` with the domain's own values normalised on write, or a per-instance setting the module reads. | Cosine does not care about the scale; euclidean does, and `recommend`'s "rated highly" needs to know what high is. |
| ~~RT-Q2~~ | **Does a repeated rating overwrite or append?** The same rater rating the same item again: latest wins (one row, edited), or a history (append, read the latest). | Context's "knowledge is never edited" rule does not reach this repo, but a history is what lets taste drift be measured later. |
| ~~RT-Q3~~ | **Which similarities are computed** — rater-to-rater, item-to-item, or both, declared per instance? | Both is twice the writes; item-to-item is what "people who liked this also liked" needs and does not depend on the current rater having rated anything. |
| ~~RT-Q4~~ | **How many neighbours are kept per rater** — the top K by score (K ≈ 20–50), or every co-rater? | All pairs over co-raters is still quadratic in a dense catalog; top-K bounds storage and every later query. |
| ~~RT-Q5~~ | **Minimum overlap** — how many co-rated items before a similarity counts? Two raters sharing one item have a perfect, meaningless score. | A floor of 2–3 is the usual answer; it is a number for measurement, not a guess. |
| ~~RT-Q6~~ | **Does it also emit `rated` edges to context**, rater → item with the value as weight, or only `similar_taste`? | With `rated` edges the graph can walk rater → similar rater → item; without them the graph knows taste but not what anyone rated, and `recommend` stays SQL-only. |
| ~~RT-Q7~~ | **Who owns the rater and item keys toward context** — must the raters' owner (accounts, actor) and the items' owner (catalog) implement `StableIdentity`/`Resolve` for a `similar_taste` edge's ends to be fetchable, and does this module implement them for `rating` rows at all? | Context fills `object`/`id` into walk results through the owners (its #56); this module is not the owner of either end. |

Still open:

| # | Question | Why it matters |
|---|----------|----------------|
| RT-Q8 | **Cold start.** A rater with no ratings, or an item nobody has rated, has no neighbours. Does `recommend` fall back to the instance's most-rated items, return nothing, or is that the host's problem? | Every new rater hits this on day one; a fallback is a popularity list the module could compute cheaply, but it is a second notion of "good" beside similarity. |
| RT-Q9 | **Who triggers the context reindex** after a rating write — this module, by calling the host's hook, or the host, which already triggers `context.reindex` after writes it knows about (context's #25)? | Ratings writes are many and small; a reindex per write may be too chatty, and a batch rule (every N writes, or every few minutes) is a decision. |
