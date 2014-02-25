# StatsD Server for Librato

[![Build Status](https://secure.travis-ci.org/jcoene/statsd-librato.png?branch=master)](http://travis-ci.org/jcoene/statsd-librato)

This is an implementation of Etsy's StatsD written in Go that submits data to Librato Metrics.

# Usage

```
Usage of statsd:
  -address="0.0.0.0:8125": udp listen address
  -debug=false: enable logging of inputs and submissions
  -flush=60: interval at which data is sent to librato (in seconds)
  -percentiles="": comma separated list of percentiles to calculate for timers (eg. "95,99.5")
  -source="": librato api source (LIBRATO_SOURCE)
  -token="": librato api token (LIBRATO_TOKEN)
  -user="": librato api username (LIBRATO_USER)
```

## Installation

**From Source:**

Check out and run "make build"

**From Binary:**

Binary releases are available for linux/amd64 and darwin/amd64. See the [releases page](https://github.com/jcoene/statsd-librato/releases) for the latest downloads.

**With Chef:**

There's a cookbook! See [statsd-librato-cookbook](https://github.com/jcoene/statsd-librato-cookbook).

## Credits

This was forked from [jbuchbinder/statsd-go](https://github.com/jbuchbinder/statsd-go) and altered to provide support for Librato as a submission backend.

## License

MIT License, see LICENSE for details.
