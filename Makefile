default: fmt run

fmt:
	go fmt *.go

test:
	go test -cover
