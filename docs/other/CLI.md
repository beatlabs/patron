# CLI

The framework supplies a CLI in order to simplify repository generation with the following features:

- git repository creation
- cmd folder and main.go creation with build version support (`go build -ldflags '-X main.version=1.0.0' main.go`)
- go module support and vendoring
- Dockerfile with version support (`docker build --build-arg version=1.0.0`)

The latest version can be installed with

```go
go get github.com/beatlabs/patron/cmd/patron
```

Below is an example of a service created with the cli that has a module name `github.com/beatlabs/test` and will be created in the test folder in the current directory.

```go
patron -m "github.com/beatlabs/test" -p "test"
```