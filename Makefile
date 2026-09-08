.PHONY: test build collector run

test:
	gofmt -w cmd component internal
	go vet ./...
	go test -race -cover ./...

build:
	go build -trimpath -o bin/valueops ./cmd/valueops

collector:
	builder --config distribution/builder-config.yaml

run:
	VALUEOPS_API_KEY=local-development-only go run ./cmd/valueops
