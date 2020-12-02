# Logging

The log package is designed to be a leveled logger with field support.

The log package defines the logger interface and a factory function type that needs to be implemented in order to set up the logging in this framework.

```go
    func Setup(l Logger) error
```

`If the setup is omitted the package will not setup any logging!`

From there logging is as simple as

```go
  log.Info("Hello world!")
```

The implementations should support the following log levels:

- Debug, which should log the message with debug level
- Info, which should log the message with info level
- Warn, which should log the message with warn level
- Error, which should log the message with error level
- Panic, which should log the message with panic level and panics
- Fatal, which should log the message with fatal level and terminates the application

The first four (Debug, Info, Warn and Error) give the opportunity to differentiate the messages based on severity. The last two (Panic and Fatal) do the same and do additional actions (panic and termination).

The package supports fields, which are logged along with the message, to augment the information further to ease querying in the log management system.

## Logger Interface

The existing 

```go
type Logger interface {
    Sub(map[string]interface{}) Logger
    Fatal(...interface{})
    Fatalf(string, ...interface{})
    Panic(...interface{})
    Panicf(string, ...interface{})
    Error(...interface{})
    Errorf(string, ...interface{})
    Warn(...interface{})
    Warnf(string, ...interface{})
    Info(...interface{})
    Infof(string, ...interface{})
    Debug(...interface{})
    Debugf(string, ...interface{})
    Level() Level
}
```

In order to be consistent with the design, the implementation of the `Fatal(f)` has to terminate the application with an error and the `Panic(f)` needs to panic.
The `Sub` function should return a new logger instance which is a clone of the current one, along with the additional fields applied.

## Context Logging

Logs can be associated with some contextual data e.g. a request id. Every line logged should contain this id thus grouping the logs together. This is achieved with the usage of the context package as demonstrated below:

```go
ctx := log.WithContext(r.Context(), log.Sub(map[string]interface{}{"requestID": uuid.New().String()}))
```

The context travels through the code as an argument and can be acquired as follows:

```go
logger:=log.FromContext(ctx)
logger.Infof("request processed")
```

Benchmarks are provided to show the performance of this.

`Every provided component creates a context logger which is then propagated in the context`

## Correlation ID propagation

Patron receives and propagates a correlation ID. Much like the distributed tracing id, the correlation id is receiver on the entry points of the service e.g. HTTP, Kafka, etc. and is propagated via the provided clients. In case no correlation ID has been received, a new one is created.  
The ID is usually received and sent via a header with key `X-Correlation-Id`.

## Provided logger implementations

The following implementations are provided as sub-packages

- zerolog, which supports the excellent [zerolog](https://github.com/rs/zerolog) package, which provides structured logging
- std, which wraps the standard log package by implementing the `Logger` interface and provides textual logging