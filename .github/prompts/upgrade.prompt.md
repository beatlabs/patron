---
mode: agent
---

- Create a branch to work on.
- Upgrade all Go modules with the following commands. Make sure to match the `go.opentelemetry.io/otel/semconv` package version in the code if the otel packages change.
```bash
go get -u ./... && go mod tidy && go mod vendor
```
- Upgrade all Docker images.
- Run unit and integration tests to ensure everything works as expected.
```bash
make test
make deps-start
make testint-nocache
make deps-stop
```
- Commit changes and push the branch to origin.
- Create a pull request to merge the changes into the master branch.
- After CI checks are successful, merge (squash and merge) the pull request.