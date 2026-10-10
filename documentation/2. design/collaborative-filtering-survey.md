# Collaborative filtering, as it applies here (RT1)

A survey to give RT2 its starting numbers. It decides nothing: requirements
#5, #6, #9–#14 stand, and where the survey finds a pressure on one it is
listed under [Raised, not decided](#raised-not-decided). Cost figures are
arithmetic over stated assumptions, not measurements; RT10 replaces them.

## Sources

Literature the vocabulary and the standard defaults come from. None was
re-fetched on this run; each is cited by author, year and title so it can be
checked.

- Resnick, Iacovou, Suchak, Bergstrom, Riedl, 1994, *GroupLens: An Open
  Architecture for Collaborative Filtering of Netnews* — user-based
  neighbourhood prediction.
- Breese, Heckerman, Kadie, 1998, *Empirical Analysis of Predictive
  Algorithms for Collaborative Filtering* — memory-based methods compared on
  several rating sets.
- Herlocker, Konstan, Borchers, Riedl, 1999, *An Algorithmic Framework for
  Performing Collaborative Filtering* — neighbourhood size, significance
  weighting by overlap.
- Sarwar, Karypis, Konstan, Riedl, 2001, *Item-Based Collaborative Filtering
  Recommendation Algorithms* — item-to-item similarity, precomputed top-K
  neighbours.
- Linden, Smith, York, 2003, *Amazon.com Recommendations: Item-to-Item
  Collaborative Filtering* — item-to-item at catalog scale.

## User-based and item-based

| | Rater-to-rater (user-based) | Item-to-item (item-based) |
|---|---|---|
| Neighbours of | a rater, by shared items | an item, by shared raters |
| A write by `r` on `i` touches | every rater of `i` | every item `r` has rated |
| Stability | drifts with every rating the rater makes | slower: an item gathers many raters, one rating moves its vectors little |
| Cold start | a rater with fewer than the overlap floor items has no neighbours | an item with fewer raters than the floor has none; a new rater with one rating can already be served |
| Serves | `recommend(rater, n)` as in the model page | "those who rated this also rated", independent of who is asking |

Sarwar 2001 and Linden 2003 prefer item-based where raters outnumber items or
change quickly, because item neighbourhoods are the more stable and can be
read without the asker's history. Here that is requirement #12: rater-to-rater
is the default because `recommend` reads it; item-to-item is the host's
switch. Neither is dropped by this survey.

The cost of the two is symmetric in the incremental step (#5), so the
computation is written once with the sides as parameters.

## The three metrics, over this module's [0, 1] values

All are computed over the items both parties rated.

- **`euclidean`**: `1 / (1 + d)`, `d` the Euclidean distance over co-rated
  items. The sum grows with overlap, so two raters who agree perfectly on 40
  items and differ by 0.1 on each score `d = 0.63`, while a pair who share 3
  items and differ by 0.1 score `d = 0.17`. Sparse pairs look closer than
  dense ones. Normalising by overlap is the common correction.
- **`cosine`**: with values in [0, 1] every vector is non-negative, so every
  cosine is already in [0, 1] and tends to sit high; it discriminates by
  direction, not by level. Herlocker 1999 and Sarwar 2001 both report that
  centring ratings (subtract the rater's mean for user-based, the rater's
  mean for adjusted cosine in item-based) improves accuracy; centring needs a
  per-rater mean, which is a second stored quantity.
- **`jaccard`**: `|both| / |either|`, over the sets of items rated, ignoring
  values. Correct for thumbs or unary signals; on graded values it discards
  the signal.

## Neighbourhood size and overlap floor

Starting grid for RT2, taken from the range the cited work explores rather
than from a claim that any value is right for this data:

| Parameter | Decided default | Grid RT2 sweeps |
|---|---|---|
| K (#13) | 30 | 10, 20, 30, 50, 100 |
| Overlap floor (#14) | 3 | 2, 3, 5, 10 |
| Metric (#6) | `euclidean` | `euclidean`, `cosine`, `jaccard` |

Herlocker 1999 introduces *significance weighting*: scale a similarity by
`min(overlap, n) / n` for a cutoff `n` (the paper's example is around 50) so
small overlaps count less, as an alternative or complement to a hard floor.
The module has a hard floor only (#14); RT2 should report whether a soft
weight beats it.

## What a Postgres-only implementation costs at 10⁴ raters × 10⁴ items

Assumptions, all to be replaced by RT10: 50 ratings per rater on average
(5·10⁵ rating rows, 0.5% dense), item popularity skewed so a randomly chosen
rating lands on an item with many more than the mean 50 raters.

**Storage.** `rating`: 5·10⁵ rows. `similarity` at K = 30: at most 3·10⁵ rows
per kind, 6·10⁵ with both. An all-pairs table would be up to 5·10⁷ rows for
rater-to-rater alone; top-K is a factor of about 170 smaller and every
`recommend` read is bounded by K.

**One write.** Steps 1 and 2 of the model page collapse into one grouped
self-join on the unique `(rater, item)` index plus the `item` index:

```sql
select b.rater, count(*), … from rating a join rating b on b.item = a.item
where a.rater = :r and b.rater <> :r group by b.rater
```

Rows touched equal the sum, over the writer's items, of each item's rater
count: about 50 × 50 = 2,500 at the mean, but size-biased towards popular
items, so a realistic mean is several times that and an item rated by 5,000
raters puts a single item's contribution at 5,000 rows. Step 3 is then
bounded by the candidates and K. This is milliseconds of work in Postgres,
not seconds; a per-write ceiling worth testing is the popular-item case, and
the model page's rule that nothing touches raters outside the item stays the
invariant that keeps it bounded (RT5's mutation check).

**`recommend`.** One join of K similarity rows against those neighbours'
ratings: about 30 × 50 = 1,500 rows before the anti-join to the rater's own
items.

**Item-to-item** doubles the writes' work and, for a rater with `m` ratings,
touches `m` items' neighbour sets per write, so a heavy rater costs `m` times
a light one; that is why #12 keeps it off by default.

## What incremental top-K cannot do alone

Requirement #13 stores only the K best per key. Three consequences for RT5:

1. **A neighbour's score falling cannot promote a non-member.** If a stored
   neighbour drops or is deleted, the K+1-th candidate was never stored, so
   the set shrinks until something refreshes it. RT5 needs a bounded per-key
   recompute (the grouped self-join above, taken for the key alone) for the
   case where a row leaves the set; it is the same query, not a new one.
2. **A rating delete can take a pair below the overlap floor.** The row must
   be deleted, not left at its old score.
3. **Symmetry is two rows.** `(r, s)` and `(s, r)` enter and leave their own
   sets independently, since each side has its own K-th. `s` may keep `r`
   while `r` has dropped `s`; that is correct, not a bug.

## Predicting from neighbours

The model page's `recommend` uses `Σ score·value / Σ score` over neighbours
who rated the item. Standard practice (Resnick 1994, Herlocker 1999) centres
each neighbour's value on that neighbour's mean and adds the target rater's
mean back, and requires several contributing neighbours. Without either, an
item one neighbour rated 1.0 outranks an item twenty neighbours rated 0.9.

## Portability note

The tests run on SQLite and integration on Postgres. `sqrt`, `power` and
similar functions are not guaranteed in the SQLite build the siblings use, so
the metric arithmetic is safest in Go over the fetched co-rated vectors, with
SQL doing only the join and grouping. This costs one fetch of at most the
co-rated items per candidate, bounded by the writer's own rating count.

## Raised, not decided

For the owner or RT2; none changes a decided row.

1. `euclidean` as written grows with overlap. RT2 should measure the plain
   form against an overlap-normalised one before RT5 fixes the formula.
2. The closed metric set has no centred measure (Pearson, adjusted cosine).
   The literature favours one for graded values. Adding a metric is a
   requirements change; RT2 can measure whether it is worth asking for.
3. `recommend` has no minimum-support rule and no centring (#9). RT2 can
   measure both; changing the query is a requirements change.
4. Soft significance weighting, beside or instead of the hard floor (#14).
