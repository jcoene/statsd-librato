package main

import (
	"testing"
)

var parsePacketTests = []struct {
	msg    string
	length int
	name   string
	bucket string
	value  float64
}{
	{"invalid", 0, "", "", 0},
	{"wrong.type:1|z", 0, "", "", 0},
	{"prefix.type:2|cg", 1, "prefix.type", "c", 2},
	{"my.name:1|c", 1, "my.name", "c", 1},
	{"first.name:6|c\nlast.name:7|c", 2, "first.name", "c", 6},
	{"some.gauge:234.6|g", 1, "some.gauge", "g", 234.6},
	{"sampled.counter:4|c|@0.5", 1, "sampled.counter", "c", 8},
	{"sampled.counter:4|c|@0.33", 1, "sampled.counter", "c", 12},
	{"first.timer:123.4567|ms\nsecond.timer:456.7890|ms", 2, "first.timer", "ms", 123.4567},
}

func TestParsePacket(t *testing.T) {
	for _, s := range parsePacketTests {
		ps := parsePacket(s.msg)
		if len(ps) != s.length {
			t.Errorf("%s: got %d packets, expected %d", s.msg, len(ps), s.length)
		}
		if len(ps) > 0 {
			if ps[0].name != s.name {
				t.Errorf("%s: got name '%s', expected '%s'", s.msg, ps[0].name, s.name)
			}
			if ps[0].bucket != s.bucket {
				t.Errorf("%s: got bucket '%s', expected '%s'", s.msg, ps[0].bucket, s.bucket)
			}
			if ps[0].value != s.value {
				t.Errorf("%s: got value '%f', expected '%f'", s.msg, ps[0].value, s.value)
			}
		}
	}
}

var parseSourceTests = []struct {
	in     string
	name   string
	source string
}{
	{"some_metric", "some_metric", ""},
	{"some_source,some_metric", "some_metric", "some_source"},
}

func TestParseSource(t *testing.T) {
	for _, s := range parseSourceTests {
		name, source := parseSource(s.in)
		if name != s.name {
			t.Errorf("%s: got '%s', expected '%s'", s.in, name, s.name)
		}
		if source != s.source {
			t.Errorf("%s: got '%s', expected '%s'", s.in, source, s.source)
		}
	}
}
