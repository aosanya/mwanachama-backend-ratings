# Measuring recommendation quality (RT2)

The numbers the build rests on — the metric (#6), K (#13) and the overlap
floor (#14) — are decided as defaults and promised to measurement. This page
is the measurement: what is held out, what is predicted, what is scored, and
how a default is chosen from the scores. RT10 runs it; the procedure itself is
in [`4. qa/measurement-procedure.md`](../4.%20qa/measurement-procedure.md).

Written 2026-10-10 by the backlog routine. It decides nothing in
`requirements.md`; where a measurement could argue for a change, the page
says so and leaves it to the owner.

## The rating set

**MovieLens ml-100k** from GroupLens (University of Minnesota): 100,000
ratings by 943 raters on 1,682 items, integers 1–5, every rater with at least
20 ratings, one file `u.data` of `rater item value timestamp` rows, plus the
set's own five 80/20 splits (`u1`–`u5`) and two splits with exactly ten test
ratings per rater (`ua`, `ub`). It is the set the literature RT1 cites
reports on, so the figures here can be read against published ones.

Why this one:

- **Size fits the harness.** RT10 loads through `put`, so every rating runs
  the incremental similarity step exactly as production will. 100,000 writes
  per grid cell is affordable; the 1M and 25M sets are not, on a first run.
- **Dense enough to have neighbours.** Twenty ratings per rater means the
  overlap floor is reachable; a sparser set would measure cold start instead
  of similarity.
- **`ml-latest-small` is a development set** that GroupLens says changes
  over time and is "not an appropriate dataset for shared research results".
- **A second set is a later run.** A thumbs-only or unary set is what tests
  `jaccard` honestly; none is chosen here.

The five-star value is normalised onto [0, 1] as `(v − 1) / 4` on the way in
(#10), so a 1 is 0.0 and a 5 is 1.0. RMSE is reported on both scales — the
[0, 1] one the module sees and the five-star one the literature uses, which is
the first multiplied by four.

The set's name carries a domain word. It appears in this page, in the QA
procedure and in the harness's data path, never in an identifier, the
blueprint or a stored value, which is what `domain_agnostic_test.go` holds.

### Terms of use

The ml-100k README licenses the set for research use: no implied endorsement,
a citation in anything published from it, no redistribution without
permission, and prior approval for commercial use. The citation GroupLens asks
for is Harper and Konstan, 2015, *The MovieLens Datasets: History and
Context*, ACM Transactions on Interactive Intelligent Systems. For this repo
that means:

- the data is never committed; the harness reads it from a directory named
  by an environment variable and is skipped when that is unset;
- the measurement is internal research on the module's own formulas, and the
  product ships none of the data;
- **the egress policy of the environment the routine runs in blocks
  `files.grouplens.org`** (confirmed on this run), so RT10 needs either the
  archive placed in the environment by hand or the policy widened. Noted on
  RT10.

## The protocol

Leave-one-out, holding out each rater's **latest** rating by timestamp. It is
deterministic, needs no seed, and asks the question production asks — given
everything this rater has done so far, what next — rather than a random
rating from the middle of their history. The set's own `ua`/`ub` splits are
not used: ten held-out ratings per rater is a different question, and they
are not ordered by time.

For one grid cell `(metric, K, overlap)`:

1. **Fresh database.** Migrate the module's blueprint under a measurement
   spec whose settings are the cell's.
2. **Load** every rating except the held-out ones, through `put`, in
   timestamp order. This is the production path, so the similarity table at
   the end is what production would hold. Record wall time and, from the
   store, the number of similarity rows written and deleted.
3. **Predict** each held-out `(rater, item)` with `recommend`'s own formula —
   `Σ score·value / Σ score` over the rater's stored neighbours who rated
   the item. A rater with no neighbour who rated the item gets **no
   prediction**; that is counted, not filled in.
4. **Score**:

   | Measure | Definition | What it tells |
   |---|---|---|
   | **RMSE** | over the predicted held-out values, on [0, 1] and ×4 | how close the formula gets when it answers |
   | **MAE** | the same, absolute | the same, less sensitive to outliers |
   | **coverage** | predicted ÷ held out | how often it answers at all |
   | **hit rate @10** | over raters whose held-out value is ≥ 0.75 (a 4 or a 5): is the held-out item in `recommend(rater, 10)`? | whether a thing the rater liked would have been shown |
   | **ARHR @10** | the same, scored `1 / rank` for a hit | whether it would have been shown near the top |

   Hit rate is recall@N with one relevant item per rater, which is what
   leave-one-out makes it. It is scored only where the held-out rating is
   high, because the question is "would we have recommended something they
   liked", and the threshold is the usual four stars. The list ranked is
   the real `recommend(rater, 10)`: the rater's other items are excluded by
   the query itself (#9), and the held-out item is a candidate because it
   was never loaded.

5. **Baselines**, scored on the same held-out set with the same measures,
   each one SQL:

   | Baseline | Prediction |
   |---|---|
   | global mean | the mean of every loaded value |
   | rater mean | the rater's own mean |
   | item mean | the item's mean, falling back to the global mean |

   A neighbourhood prediction that does not beat the item mean on RMSE has
   not earned its cost, and the grid choice below is moot for that cell.

## The grid

RT1's grid, every cell measured, 60 cells:

| Parameter | Values |
|---|---|
| metric | `euclidean`, `cosine`, `jaccard` |
| K | 10, 20, 30, 50, 100 |
| overlap floor | 2, 3, 5, 10 |

`jaccard` ignores the value and is expected to score worst on RMSE here; it
stays in the grid because its hit rate is the honest comparison for a
thumbs-only host, and because a metric that is in the closed set should have
a figure beside it.

### Choosing the defaults

1. Discard any cell with coverage below 90%: a formula that answers for
   fewer than nine raters in ten is not a default.
2. Among the rest, take the lowest RMSE on the held-out set.
3. Break a tie, or a difference under 0.005 on the [0, 1] scale, toward the
   **smaller K** and the **larger overlap**: both are cheaper per write and
   per read, and a difference that small is below what this set can
   distinguish.
4. Check that the chosen cell's hit rate @10 is not below the best cell's by
   more than one point; if it is, say so and let the owner weigh prediction
   against ranking — they are different products of the same table.

The result is three numbers for `requirements.md` #6, #13, #14, recorded in
`4. qa/measurements.md` beside the whole grid, the baselines, the wall time
per cell and the module version measured.

## What the run can also show, without deciding it

RT1 raised four pressures on decided rows. The harness can measure each as an
extra column at no cost to the protocol, so the owner's question arrives with
a number. None of these changes a default unless the owner says so.

| RT1 point | Extra measure | Decided row it bears on |
|---|---|---|
| `euclidean` grows with overlap | the same cell with `d / sqrt(overlap)` in place of `d` | #6 |
| no centred metric in the set | Pearson over the same co-rated vectors, scored as if it were a fourth metric | #6 |
| `recommend` has no centring and no minimum support | the prediction with each neighbour's value centred on their mean and the rater's mean added back; and the same with ≥ 2 contributing neighbours | #9 |
| hard floor vs soft significance weighting | the cell with score scaled by `min(overlap, 50) / 50` and no floor | #14 |

## What this page does not do

- It does not choose the numbers. RT10 does, from the run.
- It does not measure item-to-item similarity (#12). A "those who rated this
  also rated" list has no held-out truth in this protocol; it is a later
  procedure if a host switches item-to-item on.
- It does not measure the graph. `similar_taste` edges are context's to walk
  and context's TR8 to score.

## Sources

- GroupLens, ml-100k README (terms of use, file layout, counts): <https://files.grouplens.org/datasets/movielens/ml-100k-README.txt> — the read on this run came through search results and a mirror because the host is blocked from this environment; confirm against the file itself.
- GroupLens, ml-latest-small README (why it is not used): <https://files.grouplens.org/datasets/movielens/ml-latest-small-README.html>
- F. Maxwell Harper and Joseph A. Konstan, 2015, *The MovieLens Datasets: History and Context*, ACM TiiS 5(4) — the citation the set asks for.
- The leave-one-out and hit-rate protocol as used across the neighbourhood and top-N literature, e.g. the evaluation chapter of the Cambridge recommender-systems text: <https://www.cambridge.org/core/services/aop-cambridge-core/content/view/EF7B1CD8239DE1211922669BF4192F37/9781009447508c7_225-247.pdf/evaluation-in-recommender-systems.pdf>
- The warning that RMSE and top-N quality do not move together: Cremonesi, Koren and Turrin, 2010, *Performance of Recommender Algorithms on Top-N Recommendation Tasks*, RecSys — cited from memory as RT1's sources were, not re-fetched; it is why both families are scored.
- RT1's survey, [`collaborative-filtering-survey.md`](collaborative-filtering-survey.md), for the grid and the four raised points.
