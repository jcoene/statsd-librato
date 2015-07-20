FROM gliderlabs/alpine:3.2

ENTRYPOINT ["statsd"]

EXPOSE 8125
EXPOSE 8125/udp

COPY . /go/src/github.com/jcoene/statsd-librato

RUN apk-install -t build-deps go git mercurial \
  && cd /go/src/github.com/jcoene/statsd-librato \
  && export GOPATH=/go \
  && export PATH=$GOPATH/bin:$PATH \
  && go build -o /bin/statsd \
  && rm -rf /go \
  && apk del --purge build-deps go git mercurial
