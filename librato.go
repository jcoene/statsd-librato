package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"sort"
)

type Measurement struct {
	Counters []*Counter    `json:"counters"`
	Gauges   []interface{} `json:"gauges"`
	Source   string        `json:"source,omitempty"`
}

func (m *Measurement) Count() int {
	return (len(m.Counters) + len(m.Gauges))
}

type Counter struct {
	Name   string  `json:"name"`
	Source string  `json:"source,omitempty"`
	Value  float64 `json:"value"`
}

type Gauge struct {
	Name   string  `json:"name"`
	Source string  `json:"source,omitempty"`
	Value  float64 `json:"value"`
}

type ComplexGauge struct {
	Name       string  `json:"name"`
	Source     string  `json:"source,omitempty"`
	Count      int     `json:"count"`
	Sum        float64 `json:"sum"`
	Min        float64 `json:"min"`
	Max        float64 `json:"max"`
	SumSquares float64 `json:"sum_squares"`
}

func submitLibrato() (err error) {
	m := buildMeasurement()

	if m.Count() == 0 {
		return
	}

	payload, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return
	}

	if *debug {
		log.Printf("sending payload:\n%s\n", string(payload))
	}

	req, err := http.NewRequest("POST", "https://metrics-api.librato.com/v1/metrics", bytes.NewBuffer(payload))
	if err != nil {
		return
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Set("User-Agent", "statsd/1.0")
	req.SetBasicAuth(*libratoUser, *libratoToken)
	req.Close = true

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		raw, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("%s: %s", resp.Status, string(raw))
	}

	log.Printf("%d measurements sent to librato\n", m.Count())

	resetTimers()

	return
}

func buildMeasurement() (m *Measurement) {
	m = &Measurement{}
	if libratoSource != nil {
		m.Source = *libratoSource
	}

	m.Counters = make([]*Counter, len(counters))
	m.Gauges = make([]interface{}, len(gauges))

	n := 0
	for k, v := range counters {
		c := &Counter{}
		c.Name, c.Source = parseSource(k)
		c.Value = v
		m.Counters[n] = c
		n++
	}

	n = 0
	for k, v := range gauges {
		g := &Gauge{}
		g.Name, g.Source = parseSource(k)
		g.Value = v
		m.Gauges[n] = g
		n++
	}

	for k, t := range timers {
		for _, pct := range tiles {
			if g := buildComplexGauge(k, t, pct); g != nil {
				m.Gauges = append(m.Gauges, g)
			}
		}
	}

	return
}

func buildComplexGauge(k string, t []float64, pct float64) *ComplexGauge {
	threshold := ((100.0 - pct) / 100.0) * float64(len(t))
	threshold = math.Floor(threshold + 0.5)

	count := len(t) - int(threshold)
	if count <= 0 {
		return nil
	}

	g := &ComplexGauge{}
	g.Name, g.Source = parseSource(k)
	if pct != 100.0 {
		if float64(int(pct)) != pct {
			rem := int(math.Ceil((pct - float64(int(pct))) * 10))
			g.Name += fmt.Sprintf(".%d_%d", int(pct), rem)
		} else {
			g.Name += fmt.Sprintf(".%d", int(pct))
		}
	}
	g.Count = count

	sort.Float64s(t)
	g.Min = t[0]
	g.Max = t[count-1]
	for i := 0; i < count; i++ {
		g.Sum += t[i]
		g.SumSquares += (t[i] * t[i])
	}

	return g
}
