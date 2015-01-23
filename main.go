package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

const VERSION = "1.0.0"

var (
	address       = flag.String("address", "0.0.0.0:8125", "udp listen address")
	libratoUser   = flag.String("user", "", "librato api username (LIBRATO_USER)")
	libratoToken  = flag.String("token", "", "librato api token (LIBRATO_TOKEN)")
	libratoSource = flag.String("source", "", "librato api source (LIBRATO_SOURCE)")
	interval      = flag.Int64("flush", 60, "interval at which data is sent to librato (in seconds)")
	percentiles   = flag.String("percentiles", "", "comma separated list of percentiles to calculate for timers (eg. \"95,99.5\")")
	proxy         = flag.String("proxy", "", "send metrics to a proxy rather than directly to librato")
	debug         = flag.Bool("debug", false, "enable logging of inputs and submissions")
	version       = flag.Bool("version", false, "print version and exit")
)

func monitor() {
	var err error

	t := time.NewTicker(time.Duration(*interval) * time.Second)

	for {
		select {
		case <-t.C:
			if *proxy != "" {
				if err = submitProxy(); err != nil {
					log.Printf("unable to submit to proxy at %s: %s\n", *proxy, err)
				}
			} else {
				if err := submitLibrato(); err != nil {
					log.Printf("unable to submit measurements: %s\n", err)
				}
			}

		case p := <-packets:
			readPacket(p)
		}
	}
}

func main() {
	flag.Parse()

	if *version {
		fmt.Printf("statsd-librato v%s\n", VERSION)
		return
	}

	if *proxy == "" {
		getEnv(proxy, "PROXY")
	}

	if *proxy != "" {
		log.Printf("sending metrics to proxy at %s\n", *proxy)
	} else {
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

		if *percentiles == "" {
			getEnv(percentiles, "PERCENTILES")
		}

		if *percentiles != "" {
			for _, s := range strings.Split(*percentiles, ",") {
				if f := parseFloat(s); f > 0.0 && f < 100.0 {
					tiles = append(tiles, f)
					log.Printf("including percentile %f for timers\n", f)
				}
			}
		}

		log.Printf("sending metrics to librato\n")
	}

	log.Printf("flushing metrics every %d seconds\n", *interval)

	go listenUdp()
	go listenTcp()

	monitor()
}

func getEnv(p *string, key string) bool {
	if s := os.Getenv(key); s != "" {
		*p = s
		return true
	}

	return false
}
