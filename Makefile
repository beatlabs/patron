.DEFAULT_GOAL := test

OTEL_INSECURE ?= true
TEST_TIMEOUT ?= 60s
INTEGRATION_TIMEOUT ?= 30s
CI_TIMEOUT ?= 2m

.PHONY: help test testint testint-nocache ci fmt fmtcheck lint example-service example-client deps-start deps-stop

help:
	@printf '%s\n' \
		'Available targets:' \
		'  test              Run unit tests with race detection and coverage' \
		'  testint           Run integration tests with race detection and coverage' \
		'  testint-nocache   Run integration tests without cache' \
		'  ci                Run CI tests with coverage report' \
		'  fmt               Format code with go fmt' \
		'  fmtcheck          Check code formatting compliance' \
		'  lint              Run linter with golangci-lint CLI' \
		'  example-service   Run example service with OTEL configuration' \
		'  example-client    Run example client with OTEL configuration' \
		'  deps-start        Start Docker Compose dependencies' \
		'  deps-stop         Stop Docker Compose dependencies'

test: fmtcheck
	go test ./... -cover -race -timeout $(TEST_TIMEOUT)

testint: fmtcheck
	go test ./... -race -cover -tags=integration -timeout $(INTEGRATION_TIMEOUT)

testint-nocache: fmtcheck
	go test ./... -race -cover -tags=integration -timeout $(INTEGRATION_TIMEOUT) -count=1

ci:
	go test $$(go list ./... | grep -v -e 'examples' -e 'encoding/protobuf/test') -race -cover -coverprofile=coverage.txt -covermode=atomic -tags=integration -timeout $(CI_TIMEOUT)

fmt:
	go fmt ./...

fmtcheck:
	sh -c "$$(pwd)/script/gofmtcheck.sh"

lint: fmtcheck
	GOFLAGS=-mod=vendor golangci-lint run -v

example-service:
	OTEL_EXPORTER_OTLP_INSECURE=$(OTEL_INSECURE) go run examples/service/*.go

example-client:
	OTEL_EXPORTER_OTLP_INSECURE=$(OTEL_INSECURE) go run examples/client/main.go

deps-start:
	docker compose up -d --wait

deps-stop:
	docker compose down
