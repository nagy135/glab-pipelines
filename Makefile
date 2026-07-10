BINARY := glab-pipelines
MAIN := ./cmd/glab-pipelines

.PHONY: build run test fmt tidy clean

build:
	mkdir -p bin
	go build -o bin/$(BINARY) $(MAIN)

run:
	go run $(MAIN)

test:
	go test ./...

fmt:
	go fmt ./...

tidy:
	go mod tidy

clean:
	rm -rf bin dist coverage.out
