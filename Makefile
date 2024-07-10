LINT_IMAGE = golangci/golangci-lint:v1.59.0

default: test

test: fmtcheck
	go test ./... -cover -race -timeout 60s

testint: fmtcheck
	go test ./... -race -cover -tags=integration -timeout 300s

testint-nocache: fmtcheck
	go test ./... -race -cover -tags=integration -timeout 300s -count=1

ci: fmtcheck
	go test `go list ./... | grep -v -e 'examples' -e 'encoding/protobuf/test'` -race -cover -coverprofile=coverage.txt -covermode=atomic -tags=integration 

fmt:
	go fmt ./...

fmtcheck:
	@sh -c "'$(CURDIR)/script/gofmtcheck.sh'"

lint: fmtcheck
	docker run --env=GOFLAGS=-mod=vendor --rm -v $(CURDIR):/app -w /app $(LINT_IMAGE) golangci-lint -v run

deeplint: fmtcheck
	docker run --env=GOFLAGS=-mod=vendor --rm -v $(CURDIR):/app -w /app $(LINT_IMAGE) golangci-lint run --exclude-use-default=false --enable-all -D dupl --build-tags integration

example-service:
	OTEL_EXPORTER_OTLP_INSECURE="true" go run examples/service/*.go

example-client:
	OTEL_EXPORTER_OTLP_INSECURE="true" go run examples/client/main.go

deps-start:
	docker-compose up -d

deps-stop:
	docker-compose down

# disallow any parallelism (-j) for Make. This is necessary since some
# commands during the build process create temporary files that collide
# under parallel conditions.
.NOTPARALLEL:

.PHONY: default test testint testint-nocache cover coverci fmt fmtcheck lint deeplint ci modsync deps-start deps-stop example-service example-client
