package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/jbuchbinder/go-gmetric/gmetric"
	"log"
	"net"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	TCP = "tcp"
	UDP = "udp"
)

type Packet struct {
	Bucket   string
	Value    string
	Modifier string
	Sampling float32
}

var (
	serviceAddress   = flag.String("address", ":8125", "UDP service address")
	graphiteAddress  = flag.String("graphite", "", "Graphite service address (example: 'localhost:2003')")
	gangliaAddress   = flag.String("ganglia", "localhost", "Ganglia gmond servers, comma separated")
	gangliaPort      = flag.Int("ganglia-port", 8649, "Ganglia gmond service port")
	gangliaSpoofHost = flag.String("ganglia-spoof-host", "", "Ganglia gmond spoof host string")
	flushInterval    = flag.Int64("flush-interval", 10, "Flush interval")
	percentThreshold = flag.Int("percent-threshold", 90, "Threshold percent")
	debug            = flag.Bool("debug", false, "Debug mode")
)

var (
	In       = make(chan Packet, 10000)
	counters = make(map[string]int)
	timers   = make(map[string][]float64)
	gauges   = make(map[string]int)
)

func monitor() {
	var err error
	if err != nil {
		log.Println(err)
	}
	t := time.NewTicker(time.Duration(*flushInterval) * time.Second)
	for {
		select {
		case <-t.C:
			submit()
		case s := <-In:
			if s.Modifier == "ms" {
				_, ok := timers[s.Bucket]
				if !ok {
					var t []float64
					timers[s.Bucket] = t
				}
				//intValue, _ := strconv.Atoi(s.Value)
				floatValue, _ := strconv.ParseFloat(s.Value, 64)
				timers[s.Bucket] = append(timers[s.Bucket], floatValue)
			} else if s.Modifier == "g" {
				_, ok := gauges[s.Bucket]
				if !ok {
					gauges[s.Bucket] = 0
				}
				intValue, _ := strconv.Atoi(s.Value)
				gauges[s.Bucket] += intValue
			} else {
				_, ok := counters[s.Bucket]
				if !ok {
					counters[s.Bucket] = 0
				}
				floatValue, _ := strconv.ParseFloat(s.Value, 32)
				counters[s.Bucket] += int(float32(floatValue) * (1 / s.Sampling))
			}
		}
	}
}

func submit() {
	var clientGraphite net.Conn
	if clientGraphite != nil {
		log.Println(clientGraphite)
	}
	var err error
	if err != nil {
		log.Println(err)
	}
	if *graphiteAddress != "" {
		clientGraphite, err := net.Dial(TCP, *graphiteAddress)
		if clientGraphite != nil {
			// Run this when we're all done, only if clientGraphite was opened.
			defer clientGraphite.Close()
		}
		if err != nil {
			log.Printf(err.Error())
		}
	}
	var useGanglia bool
	var gm gmetric.Gmetric
	gmSubmit := func(name string, value uint32) {
		if useGanglia {
			if *debug {
				fmt.Printf("Ganglia send metric %s value %d\n", name, value)
			}
			m_value := fmt.Sprint(value)
			m_units := "count"
			m_type := uint32(gmetric.VALUE_UNSIGNED_INT)
			m_slope := uint32(gmetric.SLOPE_BOTH)
			m_grp := "statsd"
			m_ival := uint32(*flushInterval * int64(2))

			go gm.SendMetric(name, m_value, m_type, m_units, m_slope, m_ival, m_ival, m_grp)
		}
	}
	gmSubmitFloat := func(name string, value float64) {
		if useGanglia {
			if *debug {
				fmt.Printf("Ganglia send metric %s value %f\n", name, value)
			}
			m_value := fmt.Sprint(value)
			m_units := "count"
			m_type := uint32(gmetric.VALUE_DOUBLE)
			m_slope := uint32(gmetric.SLOPE_BOTH)
			m_grp := "statsd"
			m_ival := uint32(*flushInterval * int64(2))

			go gm.SendMetric(name, m_value, m_type, m_units, m_slope, m_ival, m_ival, m_grp)
		}
	}
	if *gangliaAddress != "" {
		gm = gmetric.Gmetric{
			Host:  *gangliaSpoofHost,
			Spoof: *gangliaSpoofHost,
		}
		gm.SetVerbose(false)

		if strings.Contains(*gangliaAddress, ",") {
			segs := strings.Split(*gangliaAddress, ",")
			for i := 0; i < len(segs); i++ {
				gIP, err := net.ResolveIPAddr("ip4", segs[i])
				if err != nil {
					panic(err.Error())
				}
				gm.AddServer(gmetric.GmetricServer{gIP.IP, *gangliaPort})
			}
		} else {
			gIP, err := net.ResolveIPAddr("ip4", *gangliaAddress)
			if err != nil {
				panic(err.Error())
			}
			gm.AddServer(gmetric.GmetricServer{gIP.IP, *gangliaPort})
		}
		useGanglia = true
	} else {
		useGanglia = false
	}
	numStats := 0
	now := time.Now()
	buffer := bytes.NewBufferString("")
	for s, c := range counters {
		value := float64(c) / float64((float64(*flushInterval)*float64(time.Second))/float64(1e3))
		fmt.Fprintf(buffer, "stats.%s %d %d\n", s, value, now)
		gmSubmitFloat(fmt.Sprintf("stats_%s", s), value)
		fmt.Fprintf(buffer, "stats_counts.%s %d %d\n", s, c, now)
		gmSubmit(fmt.Sprintf("stats_counts_%s", s), uint32(c))
		counters[s] = 0
		numStats++
	}
	for i, g := range gauges {
		value := int64(g)
		fmt.Fprintf(buffer, "stats.%s %d %d\n", i, value, now)
		gmSubmit(fmt.Sprintf("stats_%s", i), uint32(value))
		numStats++
	}
	for u, t := range timers {
		if len(t) > 0 {
			sort.Float64s(t)
			min := float64(t[0])
			max := float64(t[len(t)-1])
			mean := float64(min)
			maxAtThreshold := float64(max)
			count := len(t)
			if len(t) > 1 {
				var thresholdIndex int
				thresholdIndex = ((100 - *percentThreshold) / 100) * count
				numInThreshold := count - thresholdIndex
				values := t[0:numInThreshold]

				sum := float64(0)
				for i := 0; i < numInThreshold; i++ {
					sum += values[i]
				}
				mean = float64(sum) / float64(numInThreshold)
			}
			var z []float64
			timers[u] = z

			fmt.Fprintf(buffer, "stats.timers.%s.mean %f %d\n", u, mean, now)
			gmSubmitFloat(fmt.Sprintf("stats_timers_%s_mean", u), mean)
			fmt.Fprintf(buffer, "stats.timers.%s.upper %f %d\n", u, max, now)
			gmSubmitFloat(fmt.Sprintf("stats_timers_%s_upper", u), max)
			fmt.Fprintf(buffer, "stats.timers.%s.upper_%d %f %d\n", u,
				*percentThreshold, maxAtThreshold, now)
			gmSubmitFloat(fmt.Sprintf("stats_timers_%s_upper_%d", u, *percentThreshold), maxAtThreshold)
			fmt.Fprintf(buffer, "stats.timers.%s.lower %f %d\n", u, min, now)
			gmSubmitFloat(fmt.Sprintf("stats_timers_%s_lower", u), min)
			fmt.Fprintf(buffer, "stats.timers.%s.count %d %d\n", u, count, now)
			gmSubmit(fmt.Sprintf("stats_timers_%s_count", u), uint32(count))
		}
		numStats++
	}
	fmt.Fprintf(buffer, "statsd.numStats %d %d\n", numStats, now)
	gmSubmit("statsd_numStats", uint32(numStats))
	if clientGraphite != nil {
		if *debug {
			fmt.Printf("Send to graphite: [[[%s]]]\n", string(buffer.Bytes()))
		}
		clientGraphite.Write(buffer.Bytes())
	}
}

func handleMessage(conn *net.UDPConn, remaddr net.Addr, buf *bytes.Buffer) {
	var packet Packet
	var value string
	var sanitizeRegexp = regexp.MustCompile("[^a-zA-Z0-9\\-_\\.:\\|@]")
	var packetRegexp = regexp.MustCompile("([a-zA-Z0-9_]+):(\\-?[0-9\\.]+)\\|(c|ms)(\\|@([0-9\\.]+))?")
	s := sanitizeRegexp.ReplaceAllString(buf.String(), "")
	for _, item := range packetRegexp.FindAllStringSubmatch(s, -1) {
		value = item[2]
		if item[3] == "ms" {
			_, err := strconv.ParseFloat(item[2], 32)
			if err != nil {
				value = "0"
			} else {
				value = "1"
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

		if *debug {
			fmt.Printf("Packet: bucket = %s, value = %s, modifier = %s, sampling = %f\n", packet.Bucket, packet.Value, packet.Modifier, packet.Sampling)
		}

		In <- packet
	}
}

func udpListener() {
	address, _ := net.ResolveUDPAddr(UDP, *serviceAddress)
	listener, err := net.ListenUDP(UDP, address)
	defer listener.Close()
	if err != nil {
		log.Fatalf("ListenAndServe: %s", err.Error())
	}
	for {
		message := make([]byte, 512)
		n, remaddr, error := listener.ReadFrom(message)
		if error != nil {
			continue
		}
		buf := bytes.NewBuffer(message[0:n])
		if *debug {
			fmt.Printf("Packet received: " + string(message[0:n]) + "\n")
		}
		go handleMessage(listener, remaddr, buf)
	}
}

func main() {
	flag.Parse()
	go udpListener()
	monitor()
}
