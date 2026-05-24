APP_BIN := bin/api

.PHONY: run build tidy fmt test run-docker docker-reset

run:
	go run ./cmd/api

run-docker:
	docker compose up --build

docker-reset:
	docker compose down -v
	docker compose up --build

build:
	mkdir -p bin
	go build -o $(APP_BIN) ./cmd/api

fmt:
	gofmt -w ./cmd ./internal

tidy:
	go mod tidy

test:
	go test ./...
