.PHONY: test build run

test:
	gofmt -w cmd internal
	go vet ./...
	go test -race -cover ./...

build:
	go build -trimpath -o bin/valueops ./cmd/valueops

run:
	VALUEOPS_API_KEY=local-development-only go run ./cmd/valueops
