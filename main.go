package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/jcoene/gologger"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"time"
)

type Packet struct {
	Bucket   string
	Value    string
	Modifier string
	Sampling float32
}

var log *logger.Logger

var (
	serviceAddress = flag.String("address", "0.0.0.0:8125", "UDP service address")
	libratoUser    = flag.String("user", "", "Librato Username")
	libratoToken   = flag.String("token", "", "Librato API Token")
	flushInterval  = flag.Int64("flush", 60, "Flush Interval (seconds)")
	debug          = flag.Bool("debug", false, "Enable Debugging")
)

var (
	In       = make(chan Packet, 10000)
	counters = make(map[string]int64)
	timers   = make(map[string][]float64)
	gauges   = make(map[string]float64)
)

type Measurement struct {
	Counters []Counter     `json:"counters"`
	Gauges   []interface{} `json:"gauges"`
}

func (m *Measurement) Count() int {
	return (len(m.Counters) + len(m.Gauges))
}

type Counter struct {
	Name  string `json:"name"`
	Value int64  `json:"value"`
}

type SimpleGauge struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}

type ComplexGauge struct {
	Name       string  `json:"name"`
	Count      int     `json:"count"`
	Sum        float64 `json:"sum"`
	Min        float64 `json:"min"`
	Max        float64 `json:"max"`
	SumSquares float64 `json:"sum_squares"`
}

func monitor() {
	t := time.NewTicker(time.Duration(*flushInterval) * time.Second)

	for {
		select {
		case <-t.C:
			if err := submit(); err != nil {
				log.Error("submit: %s", err)
			}
		case s := <-In:
			if s.Modifier == "ms" {
				_, ok := timers[s.Bucket]
				if !ok {
					var t []float64
					timers[s.Bucket] = t
				}
				floatValue, _ := strconv.ParseFloat(s.Value, 64)
				timers[s.Bucket] = append(timers[s.Bucket], floatValue)
			} else if s.Modifier == "g" {
				_, ok := gauges[s.Bucket]
				if !ok {
					gauges[s.Bucket] = float64(0)
				}
				floatValue, _ := strconv.ParseFloat(s.Value, 64)
				gauges[s.Bucket] += floatValue
			} else {
				_, ok := counters[s.Bucket]
				if !ok {
					counters[s.Bucket] = 0
				}
				floatValue, _ := strconv.ParseFloat(s.Value, 32)
				counters[s.Bucket] += int64(float32(floatValue) * (1 / s.Sampling))
			}
		}
	}
}

func submit() (err error) {
	m := new(Measurement)
	m.Counters = make([]Counter, 0)
	m.Gauges = make([]interface{}, 0)

	for k, v := range counters {
		c := new(Counter)
		c.Name = k
		c.Value = v
		m.Counters = append(m.Counters, *c)
	}

	for k, v := range gauges {
		g := new(SimpleGauge)
		g.Name = k
		g.Value = v
		m.Gauges = append(m.Gauges, *g)
	}

	for k, t := range timers {
		g := new(ComplexGauge)
		g.Name = k
		g.Count = len(t)

		if g.Count > 0 {
			sort.Float64s(t)
			g.Min = t[0]
			g.Max = t[len(t)-1]
			for _, v := range t {
				g.Sum += v
				g.SumSquares += (v * v)
			}
		}

		m.Gauges = append(m.Gauges, *g)
	}

	if m.Count() == 0 {
		log.Info("no new measurements in the last %d seconds", *flushInterval)
		return
	}

	payload, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return
	}

	log.Debug("sending payload:\n%s", payload)

	req, err := http.NewRequest("POST", "https://metrics-api.librato.com/v1/metrics", bytes.NewBuffer(payload))
	if err != nil {
		return
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Set("User-Agent", "statsd/1.0")
	req.SetBasicAuth(*libratoUser, *libratoToken)
	resp, err := http.DefaultClient.Do(req)
	if err == nil && resp.StatusCode != 200 {
		if err == nil {
			raw, _ := ioutil.ReadAll(resp.Body)
			err = errors.New(fmt.Sprintf("%s: %s", resp.Status, string(raw)))
		}
		log.Warn("error sending %d measurements: %s", m.Count(), err)
		return
	}

	log.Info("%d measurements sent", m.Count())

	for k, _ := range gauges {
		gauges[k] = 0.0
	}

	for k, _ := range timers {
		var z []float64
		timers[k] = z
	}

	return
}

func handle(conn *net.UDPConn, remaddr net.Addr, buf *bytes.Buffer) {
	var packet Packet
	var value string
	var sanitizeRegexp = regexp.MustCompile("[^a-zA-Z0-9\\-_\\.:\\|@]")
	var packetRegexp = regexp.MustCompile("([a-zA-Z0-9_\\.]+):(\\-?[0-9\\.]+)\\|(c|ms|g)(\\|@([0-9\\.]+))?")
	s := sanitizeRegexp.ReplaceAllString(buf.String(), "")

	for _, item := range packetRegexp.FindAllStringSubmatch(s, -1) {
		value = item[2]
		if item[3] == "ms" {
			_, err := strconv.ParseFloat(item[2], 32)
			if err != nil {
				value = "0"
			}
		}

		sampleRate, err := strconv.ParseFloat(item[5], 32)
		if err != nil {
			sampleRate = 1
		}

		packet.Bucket = item[1]
		packet.Value = value
		packet.Modifier = item[3]
		packet.Sampling = float32(sampleRate)

		In <- packet
	}
}

func listen() {
	address, _ := net.ResolveUDPAddr("udp", *serviceAddress)
	listener, err := net.ListenUDP("udp", address)
	defer listener.Close()
	if err != nil {
		log.Fatal("unable to listen: %s", err)
		os.Exit(1)
	}

	log.Info("listening for events...")

	for {
		message := make([]byte, 512)
		n, remaddr, error := listener.ReadFrom(message)
		if error != nil {
			continue
		}
		buf := bytes.NewBuffer(message[0:n])
		go handle(listener, remaddr, buf)
	}
}

func main() {
	flag.Parse()

	if *debug {
		log = logger.NewLogger(logger.LOG_LEVEL_DEBUG, "statsd")
	} else {
		log = logger.NewLogger(logger.LOG_LEVEL_INFO, "statsd")
	}

	go listen()
	monitor()
}
