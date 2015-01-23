package main

import (
	"reflect"
	"testing"
)

func TestReadPackets(t *testing.T) {
	counters = make(map[string]float64)
	gauges = make(map[string]float64)
	timers = make(map[string][]float64)

	readPacket(packet{name: "a", bucket: "c", value: 15})
	readPacket(packet{name: "a", bucket: "c", value: 25})
	readPacket(packet{name: "b", bucket: "c", value: 90})

	if len(counters) != 2 {
		t.Errorf("got %d counters, expected 2", len(counters))
	}

	if counters["a"] != 40 {
		t.Errorf("got %d for counter a, expected 40", counters["a"])
	}

	if counters["b"] != 90 {
		t.Errorf("got %d for counter b, expected 90", counters["b"])
	}

	readPacket(packet{name: "a", bucket: "g", value: 15.1})
	readPacket(packet{name: "a", bucket: "g", value: 25.1})
	readPacket(packet{name: "b", bucket: "g", value: 90.1})

	if len(gauges) != 2 {
		t.Errorf("got %d gauges, expected 2", len(gauges))
	}

	if gauges["a"] != 25.1 {
		t.Errorf("got %f for gauge a, expected 25.1", gauges["a"])
	}

	if gauges["b"] != 90.1 {
		t.Errorf("got %f for gauge b, expected 90.1", gauges["b"])
	}

	readPacket(packet{name: "c", bucket: "ms", value: 15.3})
	readPacket(packet{name: "c", bucket: "ms", value: 25.3})
	readPacket(packet{name: "d", bucket: "ms", value: 90.3})

	if len(timers) != 2 {
		t.Errorf("got %d timers, expected 2", len(timers))
	}

	if !reflect.DeepEqual(timers["c"], []float64{15.3, 25.3}) {
		t.Errorf("got %+v for timer c, expected {15.3, 25.3}", timers["c"])
	}

	if !reflect.DeepEqual(timers["d"], []float64{90.3}) {
		t.Errorf("got %+v for timer d, expected {90.3}", timers["d"])
	}
}
