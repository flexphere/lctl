.PHONY: build run test lint vet fmt tidy check clean

BIN := bin/lctl
PKG := ./...

build:
	@mkdir -p bin
	go build -o $(BIN) ./cmd/lctl

run:
	go run ./cmd/lctl

test:
	go test $(PKG)

test-race:
	go test -race $(PKG)

cover:
	go test -coverprofile=coverage.out $(PKG)
	go tool cover -func=coverage.out | tail -1

lint:
	golangci-lint run

vet:
	go vet $(PKG)

fmt:
	gofmt -s -w .

tidy:
	go mod tidy

check: vet lint test

clean:
	rm -rf bin coverage.out
