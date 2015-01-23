FROM debian:wheezy

ADD statsd_linux_amd64 /usr/bin/statsd

EXPOSE 8125

ENTRYPOINT ["statsd"]
