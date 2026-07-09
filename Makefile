BINARY := glab-pipelines
MAIN := ./cmd/glab-pipelines

.PHONY: build run test fmt tidy clean

build:
	go build -o bin/$(BINARY) $(MAIN)

run:
	go run $(MAIN)

test:
	go test ./...

fmt:
	gofmt -w cmd

tidy:
	go mod tidy

clean:
	rm -rf bin dist coverage.out
