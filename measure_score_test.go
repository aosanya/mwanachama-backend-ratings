//go:build measure

package mwanachamaratings

import (
	"fmt"
	"math"
	"strings"
)

type scores struct {
	heldOut int
	n       int
	se      float64
	ae      float64
	ranked  int
	hits    int
	arhr    float64
}

func (s *scores) add(pred, actual float64) {
	s.n++
	d := pred - actual
	s.se += d * d
	s.ae += math.Abs(d)
}

func (s *scores) rank(rank int) {
	s.ranked++
	if rank > 0 {
		s.hits++
		s.arhr += 1 / float64(rank)
	}
}

func (s scores) coverage() float64 {
	if s.heldOut == 0 {
		return math.NaN()
	}
	return float64(s.n) / float64(s.heldOut)
}

func (s scores) rmse() float64 {
	if s.n == 0 {
		return math.NaN()
	}
	return math.Sqrt(s.se / float64(s.n))
}

func (s scores) mae() float64 {
	if s.n == 0 {
		return math.NaN()
	}
	return s.ae / float64(s.n)
}

func (s scores) hitRate() float64 {
	if s.ranked == 0 {
		return math.NaN()
	}
	return float64(s.hits) / float64(s.ranked)
}

func (s scores) averageReciprocalHitRank() float64 {
	if s.ranked == 0 {
		return math.NaN()
	}
	return s.arhr / float64(s.ranked)
}

const likedThreshold = 0.75

func pct(v float64) string {
	if math.IsNaN(v) {
		return "—"
	}
	return fmt.Sprintf("%.1f%%", v*100)
}

func num(v float64, digits int) string {
	if math.IsNaN(v) {
		return "—"
	}
	return fmt.Sprintf("%.*f", digits, v)
}

func row(cells ...string) string { return "| " + strings.Join(cells, " | ") + " |" }

func rule(n int) string { return "|" + strings.Repeat("---|", n) }

func predictFromNeighbours(mx *matrix, item string, neighbours []neighbour) (pred float64, support int, ok bool) {
	var num, den float64
	for _, n := range neighbours {
		v, rated := mx.byRater[n.other][item]
		if !rated {
			continue
		}
		num += n.score * v
		den += n.score
		support++
	}
	if support == 0 || den == 0 {
		return 0, 0, false
	}
	return num / den, support, true
}

func predictCentred(mx *matrix, rater, item string, neighbours []neighbour) (float64, bool) {
	var num, den float64
	for _, n := range neighbours {
		v, rated := mx.byRater[n.other][item]
		if !rated {
			continue
		}
		num += n.score * (v - mx.raterMean(n.other))
		den += n.score
	}
	if den == 0 {
		return 0, false
	}
	return math.Max(0, math.Min(1, mx.raterMean(rater)+num/den)), true
}
