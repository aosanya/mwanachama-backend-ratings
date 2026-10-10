# Measurement procedure (RT2, run by RT10)

The step list for one measurement run. The reasoning is on
[`2. design/measurement.md`](../2.%20design/measurement.md); this page is
what to do.

## Before the run

1. Obtain `ml-100k.zip` from GroupLens and extract it somewhere outside the
   repository. The data is never committed. The environment the backlog
   routine runs in cannot reach `files.grouplens.org`; a person places the
   archive, or the egress policy is widened.
2. Export `RATINGS_MEASURE_DIR=/path/to/ml-100k`. The harness is a test under
   the build tag `measure`, skipped unless that variable is set, so
   `go test ./...` never needs the data.
3. Note the module commit being measured.

## The run

```sh
make measure       # go test -tags=measure -run TestMeasure -v -timeout 0 ./...
```

For every cell of the grid — metric × K × overlap, 60 cells — the harness:

1. opens a fresh SQLite database and migrates the blueprint under a spec
   carrying the cell's settings;
2. reads `u.data`, normalises each value as `(v − 1) / 4`, finds each
   rater's latest rating by timestamp and sets it aside;
3. writes every other rating through `put` in timestamp order, timing the
   load and counting similarity rows written and deleted;
4. predicts each held-out value through `recommend`'s formula, and ranks
   `recommend(rater, 10)` for each rater whose held-out value is ≥ 0.75;
5. prints one row: cell, coverage, RMSE on [0, 1] and ×4, MAE, hit rate
   @10, ARHR @10, load seconds, rows written, rows deleted;
6. prints the three baselines once per run: global mean, rater mean, item
   mean, on the same held-out set.

## After the run

1. Paste the table into `4. qa/measurements.md` under the date and the
   commit.
2. Apply the choice rule from the design page: drop cells under 90%
   coverage, take the lowest RMSE, break near-ties toward smaller K and
   larger overlap, check hit rate @10 against the best cell.
3. Record the three chosen numbers on `requirements.md` #6, #13 and #14 as
   measured values with the date, and change the defaults in the module in
   the same commit.
4. Where an extra column (overlap-normalised euclidean, Pearson, centred
   prediction, soft weighting) beats the decided formula by more than the
   near-tie margin, add a "Needs the owner:" line to the board naming the
   figure. The decided row stays until the owner moves it.

## What a green run looks like

- Every cell answers for at least nine raters in ten at K ≥ 30 and overlap
  ≤ 3; a cell that does not has a bug or a floor that is too high for the
  set.
- The chosen cell beats the item-mean baseline on RMSE. If none does, the
  formula is not earning its table, and that is the finding.
- `jaccard` scores worst on RMSE and is not chosen; if it is chosen, the
  value is not reaching the metric.
