DOCKER = docker

default: test

test: fmtcheck
	go test ./... -cover -race -timeout 60s

testint: fmtcheck
	go test ./... -race -cover -tags=integration -timeout 60s -count=1

cover: fmtcheck
	go test ./... -coverpkg=./... -coverprofile=coverage.txt -tags=integration -covermode=atomic && \
	go tool cover -func=coverage.txt && \
	rm coverage.txt

coverci: fmtcheck
	go test ./... -race -cover -mod=vendor -coverprofile=coverage.txt -covermode=atomic -tags=integration 

fmt:
	go fmt ./...

fmtcheck:
	@sh -c "'$(CURDIR)/script/gofmtcheck.sh'"

lint: fmtcheck
	$(DOCKER) run --env=GOFLAGS=-mod=vendor --rm -v $(CURDIR):/app -w /app golangci/golangci-lint:v1.28.1 golangci-lint run --enable golint,gofmt,gosec,unparam,goconst,prealloc,stylecheck,unconvert --exclude-use-default=false --deadline=5m  --build-tags integration

deeplint: fmtcheck
	golangci-lint run --enable-all --exclude-use-default=false -D dupl --build-tags integration

ci: fmtcheck lint coverci

modsync: fmtcheck
	go mod tidy && \
	go mod vendor

# disallow any parallelism (-j) for Make. This is necessary since some
# commands during the build process create temporary files that collide
# under parallel conditions.
.NOTPARALLEL:

.PHONY: default test testint cover coverci fmt fmtcheck lint deeplint ci modsync