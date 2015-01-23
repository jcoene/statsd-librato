package main

import (
	"math"
	"regexp"
	"strconv"
	"strings"
)

var re = regexp.MustCompile("([a-zA-Z0-9_\\.,]+):(\\-?[0-9\\.]+)\\|(c|g|ms)(\\|@([0-9\\.]+))?")

func parsePacket(msg string) (packets []packet) {
	packets = make([]packet, 0)
	matches := re.FindAllStringSubmatch(msg, -1)

	for _, match := range matches {
		p := packet{
			name:   match[1],
			bucket: match[3],
			value:  parseFloat(match[2]),
		}

		if len(match) >= 5 {
			sample := parseFloat(match[5])
			if sample > 0.0 && sample < 1.0 && p.bucket == "c" {
				p.value = math.Floor(p.value * (1.0 / sample))
			}
		}

		packets = append(packets, p)
	}

	return
}

// Extracts a key into a name and source, if present.
// "my_key"           => "my_key", ""
// "my_source,my_key" => "my_key", "my_source"
func parseSource(s string) (name string, source string) {
	ss := strings.SplitN(s, ",", 2)
	if len(ss) == 2 {
		return ss[1], ss[0]
	}

	return s, ""
}

// Converts a string to a float, ignoring any errors.
// In case of error, the float will be empty(0.0)
func parseFloat(s string) (n float64) {
	n, _ = strconv.ParseFloat(s, 64)
	return
}
