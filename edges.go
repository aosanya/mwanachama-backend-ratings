package mwanachamaratings

import (
	"context"
	"fmt"
	"sort"

	"github.com/aosanya/mwanachama-backend-ratings/models"
	"github.com/aosanya/mwanachama-backend-shared/identity"
	"github.com/aosanya/mwanachama-backend-shared/specstore"
)

const PredicateSimilarTaste = "similar_taste"

const edgeKeysPerQuery = 500

type Edge struct {
	From      string  `json:"from"`
	To        string  `json:"to"`
	Predicate string  `json:"predicate"`
	Weight    float64 `json:"weight"`
}

func (m *ratingsManager) Edges(ctx context.Context, partition string, keys []string) ([]Edge, error) {
	if !identity.SegmentPattern.MatchString(partition) {
		return nil, fmt.Errorf("%w: partition %q must match %s", ErrInvalidEdges, partition, identity.SegmentPattern)
	}
	wanted, err := distinctKeys(keys)
	if err != nil {
		return nil, err
	}
	var rows []Similarity
	for start := 0; start < len(wanted); start += edgeKeysPerQuery {
		end := min(start+edgeKeysPerQuery, len(wanted))
		q := m.similarities(ctx).Where("a in ? and kind = ?", wanted[start:end], string(models.KindRater))
		got, err := specstore.List[Similarity](m.st, q, roleSimilarity)
		if err != nil {
			return nil, err
		}
		rows = append(rows, got...)
	}
	return similarTaste(rows), nil
}

func distinctKeys(keys []string) ([]string, error) {
	seen := map[string]bool{}
	out := make([]string, 0, len(keys))
	for i, k := range keys {
		if k == "" {
			return nil, fmt.Errorf("%w: key %d is empty", ErrInvalidEdges, i)
		}
		if !seen[k] {
			seen[k] = true
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out, nil
}

func similarTaste(rows []Similarity) []Edge {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].A != rows[j].A {
			return rows[i].A < rows[j].A
		}
		if rows[i].Score != rows[j].Score {
			return rows[i].Score > rows[j].Score
		}
		return rows[i].B < rows[j].B
	})
	type pair struct{ lo, hi string }
	emitted := map[pair]bool{}
	out := make([]Edge, 0, len(rows))
	for _, r := range rows {
		p := pair{r.A, r.B}
		if p.hi < p.lo {
			p = pair{r.B, r.A}
		}
		if emitted[p] {
			continue
		}
		emitted[p] = true
		out = append(out, Edge{From: r.A, To: r.B, Predicate: PredicateSimilarTaste, Weight: r.Score})
	}
	return out
}
