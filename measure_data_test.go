//go:build measure

package mwanachamaratings

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

type observation struct {
	rater string
	item  string
	value float64
	at    int64
}

func readObservations(path string) ([]observation, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []observation
	sc := bufio.NewScanner(f)
	line := 0
	for sc.Scan() {
		line++
		fields := strings.Fields(sc.Text())
		if len(fields) == 0 {
			continue
		}
		if len(fields) < 4 {
			return nil, fmt.Errorf("%s:%d: want rater item value timestamp, got %q", path, line, sc.Text())
		}
		raw, err := strconv.ParseFloat(fields[2], 64)
		if err != nil || raw < 1 || raw > 5 {
			return nil, fmt.Errorf("%s:%d: value %q is not on the five-point scale", path, line, fields[2])
		}
		at, err := strconv.ParseInt(fields[3], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%s:%d: timestamp %q: %v", path, line, fields[3], err)
		}
		out = append(out, observation{rater: fields[0], item: fields[1], value: (raw - 1) / 4, at: at})
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].at < out[j].at })
	return out, nil
}

func holdOutLatest(all []observation) (load []observation, held map[string]observation) {
	held = map[string]observation{}
	for _, o := range all {
		if prev, ok := held[o.rater]; !ok || o.at >= prev.at {
			held[o.rater] = o
		}
	}
	for _, o := range all {
		if h := held[o.rater]; h.item == o.item && h.at == o.at {
			continue
		}
		load = append(load, o)
	}
	return load, held
}

type matrix struct {
	byRater map[string]map[string]float64
	byItem  map[string]map[string]float64
	global  float64
}

func buildMatrix(load []observation) *matrix {
	mx := &matrix{byRater: map[string]map[string]float64{}, byItem: map[string]map[string]float64{}}
	var sum float64
	for _, o := range load {
		if mx.byRater[o.rater] == nil {
			mx.byRater[o.rater] = map[string]float64{}
		}
		if mx.byItem[o.item] == nil {
			mx.byItem[o.item] = map[string]float64{}
		}
		mx.byRater[o.rater][o.item] = o.value
		mx.byItem[o.item][o.rater] = o.value
		sum += o.value
	}
	if len(load) > 0 {
		mx.global = sum / float64(len(load))
	}
	return mx
}

func meanOf(values map[string]float64, fallback float64) float64 {
	if len(values) == 0 {
		return fallback
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (mx *matrix) raterMean(rater string) float64 { return meanOf(mx.byRater[rater], mx.global) }

func (mx *matrix) itemMean(item string) float64 { return meanOf(mx.byItem[item], mx.global) }

func sortedRaters(held map[string]observation) []string {
	out := make([]string, 0, len(held))
	for r := range held {
		out = append(out, r)
	}
	sort.Strings(out)
	return out
}

type baseline struct {
	name    string
	predict func(rater, item string) float64
}

func baselines(mx *matrix) []baseline {
	return []baseline{
		{name: "global mean", predict: func(string, string) float64 { return mx.global }},
		{name: "rater mean", predict: func(r, _ string) float64 { return mx.raterMean(r) }},
		{name: "item mean", predict: func(_, i string) float64 { return mx.itemMean(i) }},
	}
}

func scoreBaseline(b baseline, held map[string]observation) scores {
	var s scores
	for _, rater := range sortedRaters(held) {
		h := held[rater]
		s.add(b.predict(rater, h.item), h.value)
	}
	return s
}
