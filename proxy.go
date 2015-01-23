package main

import (
	"fmt"
	"log"
	"net"
)

func submitProxy() (err error) {
	msg, num := buildPayload()

	if num == 0 {
		return
	}

	conn, err := net.Dial("tcp", *proxy)
	if err != nil {
		return
	}

	defer conn.Close()

	n, err := conn.Write(msg)
	if err != nil {
		return
	}

	if n != len(msg) {
		return fmt.Errorf("wrote %d of %d bytes", n, len(msg))
	}

	log.Printf("%d measurements sent to proxy\n", num)

	resetAll()

	return
}

func buildPayload() ([]byte, int) {
	result := ""

	for k, v := range counters {
		result += buildMetric(k, "c", v)
	}

	for k, v := range gauges {
		result += buildMetric(k, "g", v)
	}

	n := len(counters) + len(gauges)
	for k, vs := range timers {
		n += len(vs)
		for _, v := range vs {
			result += buildMetric(k, "ms", v)
		}
	}

	return []byte(result), n
}

func buildMetric(name string, bucket string, value float64) string {
	return fmt.Sprintf("%s:%f|%s\n", name, value, bucket)
}
