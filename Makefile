name := statsd
version := $(shell cat main.go | grep VERSION | sed -e 's/\"//g' | head -n1 |cut -d' ' -f4)

default: fmt build

fmt:
	go fmt *.go

build:
	go build -o $(name)

docker-build:
	docker build -t jcoene/statsd-librato:latest .

docker-release: docker-build
	docker push jcoene/statsd-librato:latest

run: build
	./statsd -debug -flush=5 -percentiles=90,95,99

test:
	go test -cover

release:
	mkdir -p dist

	mkdir -p statsd-$(version).darwin-amd64/bin
	GOOS=darwin GOARCH=amd64 go build -o statsd-$(version).darwin-amd64/bin/statsd
	tar zcvf dist/statsd-$(version).darwin-amd64.tar.gz statsd-$(version).darwin-amd64
	rm -rf statsd-$(version).darwin-amd64

	mkdir -p statsd-$(version).linux-amd64/bin
	GOOS=linux GOARCH=amd64 go build -o statsd-$(version).linux-amd64/bin/statsd
	tar zcvf dist/statsd-$(version).linux-amd64.tar.gz statsd-$(version).linux-amd64
	rm -rf statsd-$(version).linux-amd64
