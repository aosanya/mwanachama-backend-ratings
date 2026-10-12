# Ratings model

What requirements decisions #1–#17 add up to. RT-Q8 and RT-Q9 are the only
open points and are marked where they bite.

## Objects

```
rating      rater: raw key   item: raw key   value: float in [0, 1]   at: timestamp
            one row per (rater, item); a repeat replaces it                 (#10, #11)
similarity  a: raw key       b: raw key      kind: rater | item   metric: enum
            score: float in (0, 1]   overlap: int   at: timestamp
            at most K rows per (a, kind); overlap ≥ 3                        (#13, #14)
```

Both keys are raw strings in the standard's form, `<module>/<instance>/<identity>`,
owned by other modules. This module never resolves them (#4, #16). The host
normalises the value onto [0, 1] before it arrives; the module refuses anything
outside.

### Blueprint sketch

```json
{
  "module": "ratings",
  "objects": [
    { "role": "rating", "fields": [
      { "name": "rater", "type": "string", "required": true },
      { "name": "item",  "type": "string", "required": true },
      { "name": "value", "type": "float",  "required": true },
      { "name": "at",    "type": "timestamp", "required": true } ],
      "indexes": [ { "fields": ["rater", "item"], "unique": true },
                   { "fields": ["item"] } ] },
    { "role": "similarity", "fields": [
      { "name": "a",       "type": "string", "required": true },
      { "name": "b",       "type": "string", "required": true },
      { "name": "kind",    "type": "enum", "values": ["rater", "item"], "required": true },
      { "name": "metric",  "type": "enum", "values": ["euclidean", "cosine", "jaccard"], "required": true },
      { "name": "score",   "type": "float", "required": true },
      { "name": "overlap", "type": "int",   "required": true },
      { "name": "at",      "type": "timestamp", "required": true } ],
      "indexes": [ { "fields": ["a", "kind", "b"], "unique": true },
                   { "fields": ["a", "kind", "score"] } ] }
  ]
}
```

The `[0, 1]` bound on `value` is a rule the spec cannot state, so it stays in
Go beside the type, the way every sibling keeps what a declaration cannot say
— `Rating.ValueInRange`, `Similarity.ScoreInRange` and `OverlapIsCount` in
`models/`, applied by `validate.go` after the declared rules (RT4).

### A domain spec names, it does not re-declare

```json
{ "module": "ratings", "domain": "titles", "instance": "cat1",
  "objects": [ { "role": "rating",     "name": "review",   "table": "reviews" },
               { "role": "similarity", "name": "affinity", "table": "affinities" },
               { "role": "setting",    "name": "setting",  "table": "settings" } ] }
```

Two ship under `spec/examples/`: a catalog of titles (`cat1`), and a
household rating recipes (`home1`) — far enough apart that a domain word in
the module would show. The shipped blueprint is `ratings.blueprint.json`; the
sketch above it is what it was built from.

### Where the per-instance settings live (settled by RT3, 2026-10-10)

The first sketch put a `settings` block in the domain spec. `spec.Spec`
decodes with unknown fields refused and a domain may set only a default on a
declared field, so a spec cannot carry it, and the one field it could
default — `similarity.metric` — is required at every write and may not
carry a default. So the settings are a third declared role, `setting`: a
key/value table the host writes on boot, the shape catalog already uses for
its run-time knobs.

| Key | Value | Default |
|---|---|---|
| `metric` | one of the blueprint's `similarity.metric` values | `euclidean` (#6) |
| `similarities` | a comma-separated subset of `similarity.kind` | `rater` (#12) |
| `k` | a whole number ≥ 1 | 30 (#13) |
| `overlap` | a whole number ≥ 1 | 3 (#14) |

`Settings(ctx)` merges the rows over the defaults; `PutSetting` refuses a
value outside the blueprint's vocabulary or a count below 1 before anything
is stored, reading the permitted values off the spec rather than from a
second list in Go. A value a measurement changes (RT10) changes the default
in Go and, where an instance differs, its row.

## Similarity, incrementally (decisions #5, #12–#14)

When rater `r` writes a rating for item `i`:

1. find the raters who also rated `i` — the only raters whose similarity to
   `r` can have changed;
2. for each such `s`, take the items both rated; if fewer than the overlap
   floor, skip; else compute the declared metric over the two value vectors;
3. for each of `(r, s)` and `(s, r)`: if a row exists, write the new score;
   else if fewer than K rows exist for that side, insert; else if the score
   beats that side's K-th, insert and delete the K-th.

The work is bounded by one item's raters and by K, never by the catalog.
Item-to-item, when the instance declares it, is the same three steps with
rater and item swapped, keyed on the item's raters.

### As built (RT5, 2026-10-11)

`PutRating` validates through `Check`, inserts or replaces the one
`(rater, item)` row, reads the instance's settings, and for each declared
kind runs the step above. Steps 1 and 2 are one self-join (`sides.go`):

```sql
select b.rater as other, a.value as mine, b.value as theirs
from   rating a join rating b on b.item = a.item
where  a.rater = :r and b.rater <> :r
  and  b.rater in (select rater from rating where item = :i)
```

It returns only co-rated value pairs for raters of `i`, grouped in Go by
`other`; the metric runs in Go over those pairs (SQLite's build has no
`sqrt`), so SQL does the join and nothing else. Jaccard alone needs each
side's rating count, a second grouped query over the candidates. Item-to-item
is the same query with the two columns swapped, which is what `sides` holds.
Step 3 is `place`: an existing `(a, kind, b)` row is updated in place; a new
one is inserted while fewer than K are held, else only if it beats the
weakest, which is then deleted. A tie with the K-th does not displace it.

`DeleteRating` deletes the row and then, per kind, re-scores the pairs the
survey said a delete can break: the candidates are the raters of `i`
(`sharing`); each is re-linked if still above the floor and with a positive
score, else unlinked both ways. Every side a row left is **refreshed** —
`refresh(kind, key)` recomputes that one key's top-K from scratch over
everything it co-rated (the join without the `in` clause) and rewrites its
rows. That is the bounded per-key recompute the survey asked for: it is how
a candidate that was never stored under K enters when a stored neighbour
leaves. A put never refreshes, because adding a rating cannot take a pair
below the floor and a stored score falling is left where it is (the survey's
first consequence, accepted as written).

A cosine of 0 — orthogonal vectors or an all-zero side — is treated as "no
similarity" and writes nothing, so every stored `score` is in (0, 1] for every
metric. Nothing runs in a transaction, matching the siblings: a failure
between the rating write and the similarity step leaves the rating in place
and the next write on either side re-scores the pair.

The invariant from #5 is pinned by `TestAWriteTouchesOnlyRatersOfThatItem`
with a stepping clock: after a write on `w`, a similarity row is re-stamped
only when both its keys rated `w`. A mutant that drops the `in` clause fails
it on three rows.

| Metric | Over co-rated items | Score |
|--------|---------------------|-------|
| `euclidean` | `sqrt(Σ (v_r − v_s)²)` | `1 / (1 + d)` |
| `cosine` | `Σ v_r·v_s / (‖v_r‖·‖v_s‖)` | the cosine itself, clamped to [0, 1] |
| `jaccard` | `|both rated| / |either rated|` — ignores values, for thumbs-only scales | the ratio |

## The contract with `mwanachama-backend-context` (decisions #8, #15)

```
context.reindex(partition, "ratings", [keys])
  → this module, as extractor, appends for each key:
      similar_taste  a ── b   weight = score   source = extractor:ratings      always
      rated          r ──▶ i  weight = value   source = extractor:ratings      only if the host
                                                                             declared `rated`
```

`similar_taste` is declared by the host in its `context.<domain>.json`; this
module supplies the edges, context supplies the walk. The module exposes one
library call for it, `Edges(ctx, partition, keys) ([]Edge, error)`, which the
host's reindex adapter calls; it is not a declared operation. The host batches
rater keys after writes and flushes a reindex every ~30 s or N raters, from
per-pod memory — safe because a reindex is idempotent (#18). This module has
no connection to ArangoDB and no import of context.

### As built (RT8, 2026-10-12)

`Edges(ctx, partition, keys) ([]Edge, error)` is on `RatingsManager`, in
`edges.go`. `Edge{From, To, Predicate, Weight}` (JSON `from`/`to`/
`predicate`/`weight`) is this module's own type, field for field what
context's `Fact` carries, so the host maps one to the other with no import in
either direction (#18).

- **One read, no write.** The stored `kind = rater` similarity rows whose `a`
  is a named key, in one `a in (…)` query per 500 keys — a host flushing a
  large pending set is chunked, never refused. Nothing is recomputed: the
  incremental step on every rating write (RT5) already holds each rater's K
  neighbours, so a reindex costs one indexed read per batch.
- **`similar_taste` only, rater to rater.** An item similarity (when an
  instance turned `similarities` on for items) never travels — #8 names
  taste between raters, and nothing decided an item predicate.
- **A pair is one edge.** `link` writes a pair both ways at one score, and
  context stores a symmetric predicate once per ordered pair (#46 there), so a
  pair named from both sides, or reached from both, is emitted once, from the
  lower key. Edges are ordered by `From`, then strongest first.
- **`Weight` is the stored score**, in (0, 1] by the similarity row's own
  rule, which is the range context's weight accepts. Context uses it on
  insert only (its #33).
- **`partition` is checked, not read.** It is held to
  `identity.SegmentPattern` so a malformed one stops here, but one ratings
  manager is one instance and owns no partition; the host passes the one it
  will hand to context's `Reindex`, the signature #18 decided.
- Duplicate keys collapse; an empty key or a malformed partition is
  `ErrInvalidEdges` (400 on the declared map, though no route calls it —
  `Edges` is a library call, not an operation). A rater with no neighbours,
  or a key nobody rated, is no edges and no error.

**Not built: `rated`.** #15 sends rater ▶ item edges only when the host's
`context.<domain>.json` declares `rated`, but nothing says how the host tells
this module so — ratings cannot read context's declaration (#18). Filed as
RT12 for the owner. The default (#15: off) is what ships.

S59's `identity.Written` (shared, 2026-10-12) is the report a host sends a
reacting module; `Edges` is not one — it is the extractor the host calls
while acting on a `Written{owner: "ratings", …}` of its own, with the raters
it collected (#18's pending set).

## `recommend(rater, n)` without the graph (decision #9)

```sql
select r.item, sum(s.score * r.value) / sum(s.score) as predicted
from   similarity s join rating r on r.rater = s.b
where  s.a = :rater
  and  r.item not in (select item from rating where rater = :rater)
group by r.item
order by predicted desc
limit  :n
```

One query over the K stored neighbours. The graph is not involved. A rater
with no neighbours gets an empty list; a newcomer's fallback is the host's to
compose (#19).

### As built (RT6, 2026-10-11)

`Recommend(ctx, rater, n)` is that query over the stored `kind = rater`
rows, with `count(*) as support` beside the prediction so a reader can see
how many neighbours stood behind each item; the ranking is unchanged by it.
`n` defaults to 10 and is capped at 100. A rater with no neighbours, or no
unrated item among them, gets an empty list and no error (#19). The RT1
survey's points on centring and a minimum support are not built: both would
change the query decision #9 states, so they stay raised for the owner and
RT10's measurement. `TestRecommendNeverReturnsAnItemTheRaterHasRated` pins
the anti-join; the mutant that drops it fails.

## The surface (decision #17)

| Action | Method and path | Who |
|--------|-----------------|-----|
| `ratings.rating.put` | `PUT /ratings/{item}` — rater from the caller, value in the body | any member |
| `ratings.rating.list` | `GET /ratings?rater=…` or `?item=…` | any member |
| `ratings.rating.delete` | `DELETE /ratings/{item}` — the caller's own | any member |
| `ratings.similarity.list` | `GET /similar/{key}?kind=rater\|item` | any member |
| `ratings.recommend` | `GET /recommend?n=…` — for the caller | any member |

All five come from `ratings.operations.json` through `dispatch`, so the MCP
tools (`ratings_rating_put`, …) are the same table. `AnonymousActions` is
empty. The host maps its caller to a rater key; a body never names the rater.

### As built (RT7, 2026-10-11)

`ratings.operations.json` sits beside the blueprint at the root, embedded and
reached through `Operations()`; `routes/` is the adapter over
`mwanachama-backend-shared/dispatch`, in catalog's shape: `RoutesFor`,
`SplitFor`, `OperatorRoutesFor`, `ToolsFor`, each taking a `Mount` with the
host's `Authorize` and `Caller`. Three things the build settled:

- **The `recommend` action is `ratings.recommendation.list`, not
  `ratings.recommend`.** The dispatcher holds every action id to
  `module.object.verb` (`dispatch.ActionPattern`, at least two dots), and
  refuses the two-segment form at load. The operation, path and tool
  (`ratings_recommendation_list`) follow. Decision #17 carries a note; it is
  one line to rename if the owner prefers another third segment.
- **A key in an address is percent-encoded.** Keys carry slashes
  (`catalog/cat1/x`), so `PUT /ratings/{item}` takes `catalog%2Fcat1%2Fx`:
  Go's mux matches the escaped path and hands the wildcard back unescaped,
  and an unencoded key is a 404. An MCP tool argument is a plain string and
  is unaffected. Pinned in `routes_test.go`.
- **The rater is the caller on `put`, `delete` and `recommend`.** `put`
  binds `{from: caller, into: Rater}` over the whole body and
  `{from: path, into: Item}`, so a body naming another rater or item is
  overwritten, not refused; `delete` and `recommend` take the caller as a
  positional argument. None of the three offers `rater` in its tool schema.
  `list` may name a rater in its query, because reading is not acting in
  someone's name. A mount that supplies no `Caller` binds `""`, which
  `PutRating` refuses as a missing required field.

`ListSimilarities` holds `kind` to the blueprint's enum (a `friend` is a 400),
found when the route test asked for one. The five sentinels the module
exports are all mapped to a status in the file's `errors` block, so none is
redacted to a 500; `dispatch.UnmappedSentinels` over the root package is run
by the test.
