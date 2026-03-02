.PHONY: build install test lint clean

build:
	go build -o bin/muxify ./cmd/muxify

install:
	go install ./cmd/muxify

test:
	go test -race ./...

lint:
	golangci-lint run

clean:
	rm -rf bin/
