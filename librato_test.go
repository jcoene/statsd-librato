package main

import (
	"reflect"
	"testing"
)

func TestBuildMeasurements(t *testing.T) {
	counters = make(map[string]float64)
	gauges = make(map[string]float64)
	timers = make(map[string][]float64)

	readPacket(packet{name: "a", bucket: "c", value: 15})
	readPacket(packet{name: "a", bucket: "c", value: 25})

	readPacket(packet{name: "b", bucket: "g", value: 15.1})
	readPacket(packet{name: "b", bucket: "g", value: 25.1})

	readPacket(packet{name: "c", bucket: "ms", value: 15})
	readPacket(packet{name: "c", bucket: "ms", value: 25})

	libratoSource = nil
	m := buildMeasurement()

	if m.Source != "" {
		t.Errorf("got '%s', exepcted no source", m.Source)
	}

	if m.Count() != 3 {
		t.Errorf("got %d count, expected 3", m.Count())
	}

	if !reflect.DeepEqual(m.Counters[0], &Counter{Name: "a", Value: 40}) {
		t.Errorf("unexpected value for counter 0: %+v", m.Counters[0])
	}

	if !reflect.DeepEqual(m.Gauges[0], &Gauge{Name: "b", Value: 25.1}) {
		t.Errorf("unexpected value for gauge 0: %+v", m.Gauges[0])
	}

	if !reflect.DeepEqual(m.Gauges[1], &ComplexGauge{Name: "c", Count: 2, Sum: 40, Min: 15, Max: 25, SumSquares: (15 * 15) + (25 * 25)}) {
		t.Errorf("unexpected value for gauge 0: %+v", m.Gauges[2])
	}

	s := "app01"
	libratoSource = &s

	m = buildMeasurement()

	if m.Source != "app01" {
		t.Errorf("got '%s', exepcted 'app01'", m.Source)
	}

}

func TestComplexGaugeNoData(t *testing.T) {
	got := buildComplexGauge("name", []float64{}, 100.0)
	if got != nil {
		t.Errorf("got '%+v', expected nil", got)
	}
}

func TestComplexGaugeOnePoint(t *testing.T) {
	got := buildComplexGauge("name", []float64{30}, 100.0)
	expect := &ComplexGauge{
		Name:       "name",
		Count:      1,
		Sum:        30,
		Min:        30,
		Max:        30,
		SumSquares: 30 * 30,
	}

	if !reflect.DeepEqual(got, expect) {
		t.Errorf("got '%+v', expected '%+v'", got, expect)
	}
}

func TestComplexGaugeTwoPoints(t *testing.T) {
	got := buildComplexGauge("name", []float64{30, 60}, 100.0)
	expect := &ComplexGauge{
		Name:       "name",
		Count:      2,
		Sum:        (30 + 60),
		Min:        30,
		Max:        60,
		SumSquares: (30 * 30) + (60 * 60),
	}

	if !reflect.DeepEqual(got, expect) {
		t.Errorf("got '%+v', expected '%+v'", got, expect)
	}
}

func TestComplexGaugeThreePoints(t *testing.T) {
	got := buildComplexGauge("name", []float64{30, 60, 90}, 100.0)
	expect := &ComplexGauge{
		Name:       "name",
		Count:      3,
		Sum:        (30 + 60 + 90),
		Min:        30,
		Max:        90,
		SumSquares: (30 * 30) + (60 * 60) + (90 * 90),
	}

	if !reflect.DeepEqual(got, expect) {
		t.Errorf("got '%+v', expected '%+v'", got, expect)
	}
}

func TestComplexGaugeThreePoints50th(t *testing.T) {
	got := buildComplexGauge("name", []float64{30, 60, 90}, 50.0)
	expect := &ComplexGauge{
		Name:       "name.50",
		Count:      1,
		Sum:        30,
		Min:        30,
		Max:        30,
		SumSquares: (30 * 30),
	}

	if !reflect.DeepEqual(got, expect) {
		t.Errorf("got '%+v', expected '%+v'", got, expect)
	}
}

func TestComplexGaugeThreePoints67th(t *testing.T) {
	got := buildComplexGauge("name", []float64{30, 60, 90}, 67.0)
	expect := &ComplexGauge{
		Name:       "name.67",
		Count:      2,
		Sum:        (30 + 60),
		Min:        30,
		Max:        60,
		SumSquares: (30 * 30) + (60 * 60),
	}

	if !reflect.DeepEqual(got, expect) {
		t.Errorf("got '%+v', expected '%+v'", got, expect)
	}
}

func TestComplexGaugeFourPoints50th(t *testing.T) {
	got := buildComplexGauge("name", []float64{10, 20, 30, 40}, 50.0)
	expect := &ComplexGauge{
		Name:       "name.50",
		Count:      2,
		Sum:        (10 + 20),
		Min:        10,
		Max:        20,
		SumSquares: (10 * 10) + (20 * 20),
	}

	if !reflect.DeepEqual(got, expect) {
		t.Errorf("got '%+v', expected '%+v'", got, expect)
	}
}

func TestComplexGaugeFourPoints75th(t *testing.T) {
	got := buildComplexGauge("name", []float64{10, 20, 30, 40}, 75.0)
	expect := &ComplexGauge{
		Name:       "name.75",
		Count:      3,
		Sum:        (10 + 20 + 30),
		Min:        10,
		Max:        30,
		SumSquares: (10 * 10) + (20 * 20) + (30 * 30),
	}

	if !reflect.DeepEqual(got, expect) {
		t.Errorf("got '%+v', expected '%+v'", got, expect)
	}
}

func TestComplexGaugeFourPoints99th(t *testing.T) {
	got := buildComplexGauge("name", []float64{10, 20, 30, 40}, 99.0)
	expect := &ComplexGauge{
		Name:       "name.99",
		Count:      4,
		Sum:        (10 + 20 + 30 + 40),
		Min:        10,
		Max:        40,
		SumSquares: (10 * 10) + (20 * 20) + (30 * 30) + (40 * 40),
	}

	if !reflect.DeepEqual(got, expect) {
		t.Errorf("got '%+v', expected '%+v'", got, expect)
	}
}

func TestComplexGaugeFourPoints75p3th(t *testing.T) {
	got := buildComplexGauge("name", []float64{10, 20, 30, 40}, 75.3)
	expect := &ComplexGauge{
		Name:       "name.75_3",
		Count:      3,
		Sum:        (10 + 20 + 30),
		Min:        10,
		Max:        30,
		SumSquares: (10 * 10) + (20 * 20) + (30 * 30),
	}

	if !reflect.DeepEqual(got, expect) {
		t.Errorf("got '%+v', expected '%+v'", got, expect)
	}
}

func TestComplexGaugeFourPoints75p8th(t *testing.T) {
	got := buildComplexGauge("name", []float64{10, 20, 30, 40}, 75.8)
	expect := &ComplexGauge{
		Name:       "name.75_8",
		Count:      3,
		Sum:        (10 + 20 + 30),
		Min:        10,
		Max:        30,
		SumSquares: (10 * 10) + (20 * 20) + (30 * 30),
	}

	if !reflect.DeepEqual(got, expect) {
		t.Errorf("got '%+v', expected '%+v'", got, expect)
	}
}

func TestComplexGaugeFourPoints99p5th(t *testing.T) {
	got := buildComplexGauge("name", []float64{10, 20, 30, 40}, 99.5)
	expect := &ComplexGauge{
		Name:       "name.99_5",
		Count:      4,
		Sum:        (10 + 20 + 30 + 40),
		Min:        10,
		Max:        40,
		SumSquares: (10 * 10) + (20 * 20) + (30 * 30) + (40 * 40),
	}

	if !reflect.DeepEqual(got, expect) {
		t.Errorf("got '%+v', expected '%+v'", got, expect)
	}
}
