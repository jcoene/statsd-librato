VERSION=0.1.0

default: fmt run

fmt:
	go fmt *.go

test:
	go test -cover

release:
	mkdir -p bin dist

	GOOS=darwin GOARCH=amd64 go build -o bin/statsd
	tar zcvf dist/statsd-${VERSION}.darwin-amd64.tar.gz bin/statsd
	rm -f bin/statsd

	GOOS=linux GOARCH=amd64 go build -o bin/statsd
	tar zcvf dist/statsd-${VERSION}.linux-amd64.tar.gz bin/statsd
	rm -f bin/statsd

	rmdir bin
