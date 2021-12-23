 DOCKER = docker

default: test

test: fmtcheck
	go test ./... -cover -race -timeout 60s

testint: fmtcheck
	go test ./... -race -cover -tags=integration -timeout 120s -count=1

cover: fmtcheck
	go test ./... -coverpkg=./... -coverprofile=coverage.txt -tags=integration -covermode=atomic && \
	go tool cover -func=coverage.txt && \
	rm coverage.txt

coverci: fmtcheck
	go test ./... -race -cover -mod=vendor -coverprofile=coverage.txt -covermode=atomic -tags=integration && \
	mv coverage.txt coverage.txt.tmp && \
	cat coverage.txt.tmp | grep -v "/cmd/patron/" > coverage.txt

fmt:
	go fmt ./...

fmtcheck:
	@sh -c "'$(CURDIR)/script/gofmtcheck.sh'"

lint: fmtcheck
	$(DOCKER) run --env=GOFLAGS=-mod=vendor --rm -v $(CURDIR):/app -w /app golangci/golangci-lint:v1.43.0 golangci-lint -v run -E tparallel,whitespace

deeplint: fmtcheck
	$(DOCKER) run --env=GOFLAGS=-mod=vendor --rm -v $(CURDIR):/app -w /app golangci/golangci-lint:v1.43.0 golangci-lint run --exclude-use-default=false --enable-all -D dupl --build-tags integration

ci: fmtcheck lint coverci

modsync: fmtcheck
	go mod tidy && \
	go mod vendor

examples:
	$(MAKE) -C examples

# disallow any parallelism (-j) for Make. This is necessary since some
# commands during the build process create temporary files that collide
# under parallel conditions.
.NOTPARALLEL:

.PHONY: default test testint cover coverci fmt fmtcheck lint deeplint ci modsync
