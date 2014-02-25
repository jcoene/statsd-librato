VERSION=0.1.2

default: fmt run

fmt:
	go fmt *.go

test:
	go test -cover

release:
	mkdir -p dist

	mkdir -p statsd-${VERSION}.darwin-amd64/bin
	GOOS=darwin GOARCH=amd64 go build -o statsd-${VERSION}.darwin-amd64/bin/statsd
	tar zcvf dist/statsd-${VERSION}.darwin-amd64.tar.gz statsd-${VERSION}.darwin-amd64
	rm -rf statsd-${VERSION}.darwin-amd64

	mkdir -p statsd-${VERSION}.linux-amd64/bin
	GOOS=linux GOARCH=amd64 go build -o statsd-${VERSION}.linux-amd64/bin/statsd
	tar zcvf dist/statsd-${VERSION}.linux-amd64.tar.gz statsd-${VERSION}.linux-amd64
	rm -rf statsd-${VERSION}.linux-amd64
