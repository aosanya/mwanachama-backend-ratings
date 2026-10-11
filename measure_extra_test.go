//go:build measure

package mwanachamaratings

import (
	"math"
	"sort"
	"strconv"
)

type candidate struct {
	other   string
	overlap int
	pairs   []valuePair
}

func candidatesOf(mx *matrix, rater string) []candidate {
	mine := mx.byRater[rater]
	byOther := map[string][]valuePair{}
	for item, v := range mine {
		for other, theirs := range mx.byItem[item] {
			if other == rater {
				continue
			}
			byOther[other] = append(byOther[other], valuePair{mine: v, theirs: theirs})
		}
	}
	out := make([]candidate, 0, len(byOther))
	for other, pairs := range byOther {
		out = append(out, candidate{other: other, overlap: len(pairs), pairs: pairs})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].other < out[j].other })
	return out
}

type scorer func(c candidate, mx *matrix, rater string) float64

func normalisedEuclidean(c candidate, _ *matrix, _ string) float64 {
	var sum float64
	for _, p := range c.pairs {
		d := p.mine - p.theirs
		sum += d * d
	}
	return 1 / (1 + math.Sqrt(sum)/math.Sqrt(float64(c.overlap)))
}

func pearson(c candidate, _ *matrix, _ string) float64 {
	n := float64(len(c.pairs))
	if n < 2 {
		return 0
	}
	var mx, my float64
	for _, p := range c.pairs {
		mx += p.mine
		my += p.theirs
	}
	mx /= n
	my /= n
	var cov, vx, vy float64
	for _, p := range c.pairs {
		dx, dy := p.mine-mx, p.theirs-my
		cov += dx * dy
		vx += dx * dx
		vy += dy * dy
	}
	if vx == 0 || vy == 0 {
		return 0
	}
	return cov / math.Sqrt(vx*vy)
}

func softWeighted(metric Metric) scorer {
	measure := metricFor(metric)
	return func(c candidate, mx *matrix, rater string) float64 {
		raw := measure(c.pairs, ratedCounts{mine: len(mx.byRater[rater]), theirs: len(mx.byRater[c.other])})
		return raw * math.Min(float64(c.overlap), 50) / 50
	}
}

func topNeighbours(mx *matrix, rater string, cands []candidate, score scorer, k, overlap int) []neighbour {
	var out []neighbour
	for _, c := range cands {
		if c.overlap < overlap {
			continue
		}
		s := score(c, mx, rater)
		if s <= 0 {
			continue
		}
		out = append(out, neighbour{other: c.other, score: s, overlap: c.overlap})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].score != out[j].score {
			return out[i].score > out[j].score
		}
		return out[i].other < out[j].other
	})
	if len(out) > k {
		out = out[:k]
	}
	return out
}

type allCandidates map[string][]candidate

func candidatesForAll(mx *matrix, held map[string]observation) allCandidates {
	out := allCandidates{}
	for _, rater := range sortedRaters(held) {
		out[rater] = candidatesOf(mx, rater)
	}
	return out
}

func scoreInGo(mx *matrix, held map[string]observation, cands allCandidates, score scorer, k, overlap int) scores {
	s := scores{heldOut: len(held)}
	for _, rater := range sortedRaters(held) {
		h := held[rater]
		neighbours := topNeighbours(mx, rater, cands[rater], score, k, overlap)
		if pred, _, ok := predictFromNeighbours(mx, h.item, neighbours); ok {
			s.add(pred, h.value)
		}
	}
	return s
}

type metricVariant struct {
	k, overlap int
	normalised scores
	pearson    scores
}

func metricVariants(mx *matrix, held map[string]observation, cands allCandidates, grid []cell) []metricVariant {
	seen := map[[2]int]bool{}
	var out []metricVariant
	for _, c := range grid {
		key := [2]int{c.k, c.overlap}
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, metricVariant{
			k: c.k, overlap: c.overlap,
			normalised: scoreInGo(mx, held, cands, normalisedEuclidean, c.k, c.overlap),
			pearson:    scoreInGo(mx, held, cands, pearson, c.k, c.overlap),
		})
	}
	return out
}

func metricVariantHeader() string {
	return row("K", "overlap", "euclidean d/√overlap RMSE", "coverage", "pearson RMSE", "coverage") + "\n" + rule(6)
}

func (v metricVariant) row() string {
	return row(strconv.Itoa(v.k), strconv.Itoa(v.overlap),
		num(v.normalised.rmse(), 4), pct(v.normalised.coverage()),
		num(v.pearson.rmse(), 4), pct(v.pearson.coverage()))
}

func predictionVariantHeader() string {
	return row("metric", "K", "overlap", "centred RMSE", "coverage", "support ≥ 2 RMSE", "coverage", "soft-weighted no-floor RMSE", "coverage") +
		"\n" + rule(9)
}

func predictionVariantRow(r cellResult, soft scores) string {
	return row(string(r.metric), strconv.Itoa(r.k), strconv.Itoa(r.overlap),
		num(r.centred.rmse(), 4), pct(r.centred.coverage()),
		num(r.supportTwo.rmse(), 4), pct(r.supportTwo.coverage()),
		num(soft.rmse(), 4), pct(soft.coverage()))
}
