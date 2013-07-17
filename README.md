# StatsD Server for Librato

This is an implementation of Etsy's StatsD written in Go that submits data to Librato Metrics.

This was forked from [https://github.com/jbuchbinder/statsd-go](jbuchbinder/statsd-go) and altered to provide support for Librato as a submission backend.

# Usage

```
Usage of statsd:
  -address="0.0.0.0:8125": UDP service address
  -debug=false: Enable Debugging
  -flush=30: Flush Interval (seconds)
  -token="": Librato API Token
  -user="": Librato Username
```

## License

MIT License, see LICENSE for details.
