# Ratings model — working draft

Nothing on this page is decided beyond requirements decisions #1–#9.

## Objects

```
rating      rater: raw key   item: raw key   value: float   at: timestamp
similarity  a: raw key       b: raw key      metric: enum   score: float in (0, 1]   at: timestamp
```

Both keys are raw strings in the standard's form, `<module>/<instance>/<identity>`,
owned by other modules. This module never resolves them.

## Similarity, incrementally (decision #5)

When rater `r` writes a rating for item `i`:

1. find the raters who also rated `i` — the only raters whose similarity to
   `r` can have changed;
2. for each such `s`, take the items both rated and compute the declared
   metric over the two value vectors;
3. upsert `similarity(r, s)` and `similarity(s, r)`.

The work is bounded by one item's raters, not by the catalog. Item-to-item is
the same three steps with rater and item swapped.

| Metric | Over co-rated items | Score |
|--------|---------------------|-------|
| `euclidean` | `sqrt(Σ (v_r − v_s)²)` | `1 / (1 + d)` |
| `cosine` | `Σ v_r·v_s / (‖v_r‖·‖v_s‖)` | the cosine itself, clamped to [0, 1] |
| `jaccard` | `|both rated| / |either rated|` — ignores values, for thumbs-only scales | the ratio |

## The contract with `mwanachama-backend-context` (decision #8)

```
context.reindex(partition, "ratings", [rater keys])
  → this module, as extractor, appends:
      similar_taste  a ── b   weight = score   source = extractor:ratings
```

`similar_taste` is declared by the host in its `context.<domain>.json`; this
module supplies the edges, context supplies the walk. Whether `rated` edges
travel too is RT-Q6.

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

One query over the top-K neighbours (RT-Q4). The graph is not involved.
