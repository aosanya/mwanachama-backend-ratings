# QA

Nothing is built, so nothing is tested yet.

## Two tiers

```sh
go test ./...        # in-memory SQLite (glebarez/sqlite), no database needed
make test-pg         # -tags=integration, with POSTGRES_URL set
```

## What will be measured (RT5)

Hold out one rating per rater from a public rating set, predict it through
`recommend`'s formula, and score **RMSE** on the prediction and **recall@N**
on the recommended list. The same run chooses the metric, the neighbourhood
size K (RT-Q4) and the overlap floor (RT-Q5), so none of those is a guess.

## What will be pinned, when the code exists

- a rating write recomputes similarity only for raters sharing that item (#5);
- every score lands in (0, 1] for every metric (#6);
- `recommend` never returns an item the rater has rated (#9);
- the module names no domain, in identifiers, blueprint and stored values (#7);
- both shipped domain specs migrate into one database and fit the Go types.
