LINT_IMAGE = golangci/golangci-lint:v1.61.0

.PHONY: default
default: test

.PHONY: test
test: fmtcheck
	go test ./... -cover -race -timeout 60s

.PHONY: testint
testint: fmtcheck
	go test ./... -race -cover -tags=integration -timeout 300s

.PHONY: testint-nocache
testint-nocache: fmtcheck
	go test ./... -race -cover -tags=integration -timeout 300s -count=1

.PHONY: ci
ci: 
	go test `go list ./... | grep -v -e 'examples' -e 'encoding/protobuf/test'` -race -cover -coverprofile=coverage.txt -covermode=atomic -tags=integration 

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: fmtcheck
fmtcheck:
	@sh -c "'$(CURDIR)/script/gofmtcheck.sh'"

.PHONY: lint
lint: fmtcheck
	docker run --env=GOFLAGS=-mod=vendor --rm -v $(CURDIR):/app -w /app $(LINT_IMAGE) golangci-lint -v run

.PHONY: deeplint
deeplint: fmtcheck
	docker run --env=GOFLAGS=-mod=vendor --rm -v $(CURDIR):/app -w /app $(LINT_IMAGE) golangci-lint run --exclude-use-default=false --enable-all -D dupl --build-tags integration

.PHONY: example-service
example-service:
	OTEL_EXPORTER_OTLP_INSECURE="true" go run examples/service/*.go

.PHONY: example-client
example-client:
	OTEL_EXPORTER_OTLP_INSECURE="true" go run examples/client/main.go

.PHONY: deps-start
deps-start:
	docker compose up -d

.PHONY: deps-stop
deps-stop:
	docker compose down

# disallow any parallelism (-j) for Make. This is necessary since some
# commands during the build process create temporary files that collide
# under parallel conditions.
.NOTPARALLEL:
