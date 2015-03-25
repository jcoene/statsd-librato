FROM debian:wheezy

RUN \
  apt-get update && \
  apt-get install -y --no-install-recommends ca-certificates && \
  rm -rf /var/lib/apt/lists/*

ADD statsd_linux_amd64 /usr/bin/statsd

EXPOSE 8125
EXPOSE 8125/udp

ENTRYPOINT ["statsd"]
