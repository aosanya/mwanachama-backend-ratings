package mwanachamaratings

import (
	"math"
	"testing"

	"github.com/aosanya/mwanachama-backend-ratings/models"
)

func close(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestEuclideanIsOneOverOnePlusDistance(t *testing.T) {
	same := []valuePair{{0.2, 0.2}, {0.9, 0.9}, {0.5, 0.5}}
	if got := euclidean(same, ratedCounts{}); !close(got, 1) {
		t.Errorf("identical vectors score %v, want 1", got)
	}
	apart := []valuePair{{0, 1}, {1, 0}}
	if got := euclidean(apart, ratedCounts{}); !close(got, 1/(1+math.Sqrt2)) {
		t.Errorf("opposite corners score %v, want %v", got, 1/(1+math.Sqrt2))
	}
}

func TestCosineIsDirectionClampedToTheUnitInterval(t *testing.T) {
	scaled := []valuePair{{0.2, 0.4}, {0.4, 0.8}, {0.1, 0.2}}
	if got := cosine(scaled, ratedCounts{}); !close(got, 1) {
		t.Errorf("a scaled copy scores %v, want 1", got)
	}
	orthogonal := []valuePair{{1, 0}, {0, 1}, {0, 0}}
	if got := cosine(orthogonal, ratedCounts{}); got != 0 {
		t.Errorf("orthogonal vectors score %v, want 0 so no row is written", got)
	}
	zero := []valuePair{{0, 0.5}, {0, 0.7}, {0, 0.1}}
	if got := cosine(zero, ratedCounts{}); got != 0 {
		t.Errorf("an all-zero side scores %v, want 0 rather than NaN", got)
	}
}

func TestJaccardIgnoresValuesAndCountsSets(t *testing.T) {
	shared := []valuePair{{0, 1}, {1, 0}, {0.5, 0.5}}
	if got := jaccard(shared, ratedCounts{mine: 5, theirs: 4}); !close(got, 3.0/6.0) {
		t.Errorf("3 shared of 5 and 4 scores %v, want 0.5", got)
	}
	if got := jaccard(nil, ratedCounts{}); got != 0 {
		t.Errorf("nothing shared scores %v, want 0", got)
	}
}

func TestEveryMetricIsResolvedByName(t *testing.T) {
	shared := []valuePair{{0.3, 0.6}, {0.9, 0.8}, {0.1, 0.2}}
	for _, m := range []Metric{models.MetricEuclidean, models.MetricCosine, models.MetricJaccard} {
		got := metricFor(m)(shared, ratedCounts{mine: 3, theirs: 3})
		if !(got > 0 && got <= 1) {
			t.Errorf("%s scores %v, outside (0, 1]", m, got)
		}
	}
}
