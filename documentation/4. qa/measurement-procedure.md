# Measurement procedure (RT2)

> **2026-10-11.** RT10 is closed without a public-set run (owner: the module
> is domain-agnostic; a deployment's own ratings are the data). This
> procedure stands for whoever wants to measure a real instance: export its
> ratings as `rater item value timestamp` rows on a five-point scale into a
> directory, and the steps below apply unchanged. Steps that name ml-100k
> describe one example set.

The step list for one measurement run. The reasoning is on
[`2. design/measurement.md`](../2.%20design/measurement.md); this page is
what to do.

## Before the run

1. Obtain `ml-100k.zip` from GroupLens and extract it somewhere outside the
   repository. The data is never committed. The environment the backlog
   routine runs in cannot reach `files.grouplens.org`; a person places the
   archive, or the egress policy is widened. On 2026-10-11 the routine also
   tried every other host that carries a copy — `web.archive.org`,
   `huggingface.co`, `zenodo.org`, `openml.org`, `figshare.com`,
   `data.mendeley.com` — and every one answers 403 at `CONNECT`. `pypi.org`
   is open but no package there bundles the set; the ones that name it
   download from GroupLens at run time.
2. Export `RATINGS_MEASURE_DIR=/path/to/ml-100k`. The harness is a test under
   the build tag `measure`, skipped unless that variable is set, so
   `go test ./...` never needs the data.
3. Note the module commit being measured.

## The harness

Built 2026-10-11 as five files in the root package under `//go:build
measure`, so they are invisible to `go build`, `go vet` and `go test ./...`:

| File | Holds |
|---|---|
| `measure_test.go` | `TestMeasure`, the grid, and the three narrowing variables below |
| `measure_data_test.go` | reading `u.data`, the hold-out, the in-memory matrix, the baselines |
| `measure_cell_test.go` | one cell: a fresh SQLite database, the settings as rows, the load through `PutRating`, prediction from `ListSimilarities`, the rank from `Recommend` |
| `measure_score_test.go` | RMSE, MAE, coverage, hit rate, ARHR, and the two prediction formulas |
| `measure_extra_test.go` | the variants computed in Go over the loaded matrix |

Three variables narrow the grid for a shorter run, each a comma-separated
list; unset, the grid is the full 60 cells:
`RATINGS_MEASURE_METRICS=euclidean,cosine`, `RATINGS_MEASURE_K=10,30`,
`RATINGS_MEASURE_OVERLAP=3`. A narrowed run is a rehearsal, not the
measurement; only a full grid sets a default.

What the figures are:

- **rows written** counts every insert *and* every in-place update of a
  similarity row, and **rows deleted** every row the top-K displacement or a
  refresh removed. Both are read off GORM callbacks on the similarity table,
  so they are the writes production would make for the same load.
- The prediction for a held-out item is `recommend`'s formula applied to
  the rater's stored neighbours (`ListSimilarities`, the cell's K) and the
  loaded matrix. The rank is the real `recommend(rater, 10)`.
- The three **baselines** are computed from the loaded matrix in Go. They
  do not depend on the cell and are printed once.
- The **metric variants** (euclidean over `d / √overlap`, Pearson) do not go
  through `put`: the harness scores every candidate neighbour in Go over the
  loaded matrix, applies the cell's floor and K, and predicts with the
  decided formula. They depend only on K and overlap, so they are printed
  once per `(K, overlap)` pair. A Pearson score of zero or below is not a
  neighbour.
- The **prediction variants** use the stored neighbours of the cell:
  *centred* is each neighbour's value less that neighbour's mean, weighted
  as before, plus the rater's mean, clamped to [0, 1]; *support ≥ 2* is the
  decided prediction counted only where two or more neighbours contributed.
  *Soft-weighted, no floor* is computed in Go: the cell's metric scaled by
  `min(overlap, 50) / 50` with an overlap floor of 1.
- Coverage of a variant is over the same held-out set as the grid's.

The harness was proven on a synthetic set of 1,500 ratings by 60 raters
over 40 items (not committed; minted by the routine) with a narrowed grid of
12 cells: every column filled, every cell at 100% coverage, `jaccard`
worst on RMSE as expected, 56 s for the twelve loads. Those figures measure
nothing about the module and are not recorded on `measurements.md`.

## The run

```sh
make measure       # go test -tags=measure -run TestMeasure -v -timeout 0 ./...
```

Budget the time before starting: the synthetic rehearsal ran at about 3 ms
per `put` on a 60-rater matrix, and a put's cost grows with the number of
raters who share the item, so on ml-100k expect a cell to take from tens of
minutes to a few hours, and the full grid to need a run that is left alone.

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
   mean, on the same held-out set;
7. prints the metric variants once per `(K, overlap)` and the prediction
   variants once per cell — the extra columns the design page asks for,
   so an owner question arrives with a number.

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
