BINARY_NAME := driftcheck

.PHONY: build test test-race vet lint clean

build:
	go build -o bin/$(BINARY_NAME) .

test:
	go test ./...

test-race:
	go test -race ./...

vet:
	go vet ./...

lint:
	golangci-lint run

clean:
	rm -rf bin
