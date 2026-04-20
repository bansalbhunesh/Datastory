.PHONY: run test tidy build lint

run:
	go run ./cmd/server

build:
	go build -o bin/datastory ./cmd/server

test:
	go test ./... -race -count=1

tidy:
	go mod tidy

lint:
	go vet ./...
