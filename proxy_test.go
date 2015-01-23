package main

import (
	"sort"
	"strings"
	"testing"
)

func TestBuildPayload(t *testing.T) {
	counters = make(map[string]float64)
	gauges = make(map[string]float64)
	timers = make(map[string][]float64)

	readPacket(packet{name: "a", bucket: "c", value: 15})
	readPacket(packet{name: "a", bucket: "c", value: 25})
	readPacket(packet{name: "b", bucket: "c", value: 90})

	readPacket(packet{name: "a", bucket: "g", value: 15.1})
	readPacket(packet{name: "a", bucket: "g", value: 25.1})
	readPacket(packet{name: "b", bucket: "g", value: 90.1})

	readPacket(packet{name: "c", bucket: "ms", value: 15.3})
	readPacket(packet{name: "c", bucket: "ms", value: 25.3})
	readPacket(packet{name: "d", bucket: "ms", value: 90.3})

	expect := sortLines(
		"a:40.000000|c\n" +
			"b:90.000000|c\n" +
			"a:25.100000|g\n" +
			"b:90.100000|g\n" +
			"c:15.300000|ms\n" +
			"c:25.300000|ms\n" +
			"d:90.300000|ms\n")

	buf, num := buildPayload()
	got := sortLines(string(buf))

	if expect != string(got) {
		t.Errorf("got '%s', expected '%s'", string(got), expect)
	}

	if num != 7 {
		t.Errorf("got %d measurements, expected 7", num)
	}
}

func sortLines(s string) string {
	ss := strings.Split(s, "\n")
	sort.Strings(ss)
	return strings.Join(ss, "\n")
}
