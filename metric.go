package main

var (
	counters = make(map[string]float64)
	gauges   = make(map[string]float64)
	timers   = make(map[string][]float64)
	tiles    = make([]float64, 0)
)

func init() {
	tiles = append(tiles, 100.0)
}

func readPacket(p packet) {
	switch p.bucket {
	case "c":
		if _, f := counters[p.name]; !f {
			counters[p.name] = 0.0
		}
		counters[p.name] += p.value

	case "g":
		gauges[p.name] = p.value

	case "ms":
		if _, f := timers[p.name]; !f {
			timers[p.name] = make([]float64, 0)
		}
		timers[p.name] = append(timers[p.name], p.value)
	}
}

func resetTimers() {
	timers = make(map[string][]float64)
}

func resetAll() {
	counters = make(map[string]float64)
	gauges = make(map[string]float64)
	timers = make(map[string][]float64)
}
