LINT_IMAGE = golangci/golangci-lint:v2.1.1-alpine

default: help

.PHONY: test
test: fmtcheck ## Run tests
	go test ./... -cover -race -timeout 60s

.PHONY: testint
testint: fmtcheck ## Run integration tests
	go test ./... -race -cover -tags=integration -timeout 60s

.PHONY: testint-nocache
testint-nocache: fmtcheck ## Run integration tests without cache
	go test ./... -race -cover -tags=integration -timeout 60s -count=1

.PHONY: ci
ci: ## Run the CI pipeline
	go test `go list ./... | grep -v -e 'examples' -e 'encoding/protobuf/test'` -race -cover -coverprofile=coverage.txt -covermode=atomic -tags=integration -timeout 120s

.PHONY: fmt
fmt: ## Format the code
	go fmt ./...

.PHONY: fmtcheck
fmtcheck: ## Check if the code is formatted correctly
	@sh -c "'$(CURDIR)/script/gofmtcheck.sh'"

.PHONY: lint
lint: fmtcheck ## Run linter with default rules
	docker run --env=GOFLAGS=-mod=vendor --rm -v $(CURDIR):/app -w /app $(LINT_IMAGE) golangci-lint -v run

.PHONY: deeplint
deeplint: fmtcheck ## Run linter with all rules enabled
	docker run --env=GOFLAGS=-mod=vendor --rm -v $(CURDIR):/app -w /app $(LINT_IMAGE) golangci-lint run --exclude-use-default=false --enable-all -D dupl --build-tags integration

.PHONY: example-service
example-service: ## Run the example service
	OTEL_EXPORTER_OTLP_INSECURE="true" go run examples/service/*.go

.PHONY: example-client
example-client: ## Run the example client
	OTEL_EXPORTER_OTLP_INSECURE="true" go run examples/client/main.go

.PHONY: deps-start
deps-start: ## Start dependencies
	docker compose up -d --wait

.PHONY: deps-stop
deps-stop: ## Stop dependencies
	docker compose down

.PHONY: list
list: ## List all make targets
	@${MAKE} -pRrn : -f $(MAKEFILE_LIST) 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | egrep -v -e '^[^[:alnum:]]' -e '^$@$$' | sort

.PHONY: help
help:
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

# disallow any parallelism (-j) for Make. This is necessary since some
# commands during the build process create temporary files that collide
# under parallel conditions.
.NOTPARALLEL:
