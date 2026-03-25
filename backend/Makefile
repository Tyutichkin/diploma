APP_BIN := bin/api

.PHONY: run build tidy fmt test

run:
	go run ./cmd/api

build:
	mkdir -p bin
	go build -o $(APP_BIN) ./cmd/api

fmt:
	gofmt -w ./cmd ./internal

tidy:
	go mod tidy

test:
	go test ./...
