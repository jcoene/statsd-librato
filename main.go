package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/jcoene/gologger"
	"io/ioutil"
	"math"
	"net"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const VERSION = "0.1.4"

type Packet struct {
	Bucket   string
	Value    string
	Modifier string
	Sampling float32
}

var log *logger.Logger
var sanitizeRegexp = regexp.MustCompile("[^a-zA-Z0-9\\-_\\.,:\\|@]")
var packetRegexp = regexp.MustCompile("([a-zA-Z0-9_\\.,]+):(\\-?[0-9\\.]+)\\|(c|ms|g)(\\|@([0-9\\.]+))?")

var (
	serviceAddress = flag.String("address", "0.0.0.0:8125", "udp listen address")
	libratoUser    = flag.String("user", "", "librato api username (LIBRATO_USER)")
	libratoToken   = flag.String("token", "", "librato api token (LIBRATO_TOKEN)")
	libratoSource  = flag.String("source", "", "librato api source (LIBRATO_SOURCE)")
	flushInterval  = flag.Int64("flush", 60, "interval at which data is sent to librato (in seconds)")
	percentiles    = flag.String("percentiles", "", "comma separated list of percentiles to calculate for timers (eg. \"95,99.5\")")
	debug          = flag.Bool("debug", false, "enable logging of inputs and submissions")
	version        = flag.Bool("version", false, "print version and exit")
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
	Source   *string       `json:"source,omitempty"`
}

func (m *Measurement) Count() int {
	return (len(m.Counters) + len(m.Gauges))
}

type Counter struct {
	Name   string  `json:"name"`
	Source *string `json:"source,omitempty"`
	Value  int64   `json:"value"`
}

type SimpleGauge struct {
	Name   string  `json:"name"`
	Source *string `json:"source,omitempty"`
	Value  float64 `json:"value"`
}

type ComplexGauge struct {
	Name       string  `json:"name"`
	Source     *string `json:"source,omitempty"`
	Count      int     `json:"count"`
	Sum        float64 `json:"sum"`
	Min        float64 `json:"min"`
	Max        float64 `json:"max"`
	SumSquares float64 `json:"sum_squares"`
	Median     float64 `json:"median"`
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
				gauges[s.Bucket] = floatValue
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

func parseBucket(bucket string) (string, *string) {
	if strings.Contains(bucket, ",") {
		ss := strings.SplitN(bucket, ",", 2)
		return ss[1], &ss[0]
	}

	return bucket, nil
}

func submit() (err error) {
	m := new(Measurement)
	if *libratoSource != "" {
		m.Source = libratoSource
	}
	m.Counters = make([]Counter, 0)
	m.Gauges = make([]interface{}, 0)

	for k, v := range counters {
		c := new(Counter)
		c.Name, c.Source = parseBucket(k)
		c.Value = v
		m.Counters = append(m.Counters, *c)
	}

	for k, v := range gauges {
		g := new(SimpleGauge)
		g.Name, g.Source = parseBucket(k)
		g.Value = v
		m.Gauges = append(m.Gauges, *g)
	}

	for k, t := range timers {
		g := gaugePercentile(k, t, 100.0, "")
		m.Gauges = append(m.Gauges, *g)

		if *percentiles != "" {
			pcts := strings.Split(*percentiles, ",")
			for _, pct := range pcts {
				pctf, err := strconv.ParseFloat(pct, 64)
				if err != nil {
					log.Warn("error parsing '%s' as float: %s", pct, err)
					continue
				}

				if g = gaugePercentile(k, t, pctf, pct); g != nil {
					m.Gauges = append(m.Gauges, *g)
				}
			}
		}
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

	req.Close = true

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

	for k, _ := range timers {
		delete(timers, k)
	}

	return
}

func gaugePercentile(k string, t []float64, pct float64, suffix string) *ComplexGauge {
	thresholdIdx := ((100.0 - pct) / 100.0) * float64(len(t))
	thresholdIdx = math.Floor(thresholdIdx + 0.5)

	numInPct := len(t) - int(thresholdIdx)
	if numInPct <= 0 {
		return nil
	}

	g := new(ComplexGauge)
	g.Name, g.Source = parseBucket(k)
	if suffix != "" {
		g.Name += "." + suffix
	}
	g.Count = numInPct

	if g.Count > 0 {
		sort.Float64s(t)
		g.Min = t[0]
		g.Max = t[numInPct-1]
		for i := 0; i < numInPct; i++ {
			v := t[i]
			g.Sum += v
			g.SumSquares += (v * v)
		}

		mid := g.Count / 2
		g.Median = t[mid]

		if g.Count > 2 && g.Count%2 == 0 {
			g.Median += t[mid-1]
		}
	}

	return g
}

func handle(conn *net.UDPConn, remaddr net.Addr, buf *bytes.Buffer) {
	var packet Packet
	var value string
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

		log.Debug("received metric: %s", message)

		buf := bytes.NewBuffer(message[0:n])
		go handle(listener, remaddr, buf)
	}
}

func main() {
	flag.Parse()

	if *version {
		fmt.Printf("statsd-librato v%s\n", VERSION)
		return
	}

	if *debug {
		log = logger.NewLogger(logger.LOG_LEVEL_DEBUG, "statsd")
	} else {
		log = logger.NewLogger(logger.LOG_LEVEL_INFO, "statsd")
	}

	if *libratoUser == "" {
		if !getEnv(libratoUser, "LIBRATO_USER") {
			log.Fatal("specify a librato user with -user or the LIBRATO_USER environment variable")
		}
	}

	if *libratoToken == "" {
		if !getEnv(libratoToken, "LIBRATO_TOKEN") {
			log.Fatal("specify a librato token with -token or the LIBRATO_TOKEN environment variable")
		}
	}

	if *libratoSource == "" {
		getEnv(libratoSource, "LIBRATO_SOURCE")
	}

	go listen()
	monitor()
}

func getEnv(p *string, key string) bool {
	if s := os.Getenv(key); s != "" {
		*p = s
		return true
	}

	return false
}
