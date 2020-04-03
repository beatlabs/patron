# Example#1

## Content
- [Logging](#logging)
- [Tracing](#tracing)
- [Service](#service)
    - [Router](#router)
        - [ProcessorFunc](#processorfunc) 
    - [Middlewares](#middlewares)
    - [SIGHUP handling](#sighup-handling)
- [Let's run it!](#lets-run-it)

### Let's go!
In this example, you'll get to know how to set up a simple server with an endpoint and a middleware using `Patron`. It also covers a few other topics such as logging, tracing, and internals features of `Patron`.

#### Logging
The very first step is logging initializing. It's configured via [patron.SetupLogging](https://github.com/beatlabs/patron/blob/master/service.go#L24) that accepts `name`( e.g. `billing-api` ) and  `version` of your service.  

These fields ( and `hostname` as well ) will be attached to all the log entries - so, pay attention to it and don't ignore them.

> In this example `patron.SetupLogging` in being executed inside `Run` method.

Please, bear in mind that by default, the log level is `Info`.  

`Patron` provides quite a typical set of levels `debug`, `info`, `warn`, `error`, `fatal`, `panic`, and a special "nolevel" one.

The only way to set the level is by using ENV variables. For setting the log level, you need to use `PATRON_LOG_LEVEL` - it has done in this example in `init` function.

 
#### Tracing
`Patron` uses [Jaeger](https://www.jaegertracing.io/) for tracing.
Here's a list of ENV variables you can use for setting tracing:
- `PATRON_JAEGER_AGENT_HOST` (default value `0.0.0.0`);
- `PATRON_JAEGER_AGENT_PORT` - (default value `6831`);
- `PATRON_JAEGER_SAMPLER_TYPE` - (default value `probabilistic`); [See other options](https://www.jaegertracing.io/docs/1.17/sampling/)
- `PATRON_JAEGER_SAMPLER_PARAM` - (default value `0.0`) ;

`Patron` initializes tracing internally by using these variables. Since it set by using the `Run` method, Patron uses the same `name` and `version` ( from the [Logging](#logging) step ) variables. This information will propagate to all spans that Patron creates.

> As you can see, in this example, `PATRON_JAEGER_SAMPLER_PARAM` is being set to `1.0` that means 100% sampling.

#### Service
`Patron` provides a convenient API to cover quite all basic "Service" requirements like routes, middlewares, k8s probes, customization, and OS signals handling.

In this example, you could have already noticed this piece of code:
```go
    err = patron.New(name, version).
        WithRoutesBuilder(routesBuilder).
        WithMiddlewares(middlewareCors).
        WithSIGHUP(sig).
        Run(context.Background())
```
So that is how a service initializes. Let's take a look at each part deeper.

##### Router
`Patron` comes with a built-in routes builder. It simplifies the configuring HTTP endpoints. 

In this particular example, there is only one HTTP endpoint is configured.

This is `POST /` with `first` handler. (see [documentation](#processorfunc) )
```go
   routesBuilder := patronhttp.NewRoutesBuilder().
        Append(patronhttp.NewRouteBuilder("/", first).MethodPost())
```

There are two ways of defining endpoints:
- Using `NewRouteBuilder`. It creates a new route and cares about all the boilerplate generic logic ( see [ProcessorFunc](#processorFunc) ).
- Using `NewRawRouteBuilder` that allows you to use [http.HandlerFunc](https://golang.org/pkg/net/http/#HandlerFunc) directly with your custom handler implementation.

> Please note that by using `NewRouteBuilder`, your handler should implement the `ProcessorFunc` type that limits access to the original [http.Request](https://golang.org/pkg/net/http/#Request) from stdlib.

The documentation is [here](https://godoc.org/github.com/beatlabs/patron/component/http#RouteBuilder).

###### ProcessorFunc
As mentioned in [Router](#router) part, there is a simple way of creation endpoints by using `NewRouteBuilder`. It accepts the `path` and implementation of the `ProcessorFunc` type.
```go
type ProcessorFunc func(context.Context, *Request) (*Response, error)
```
`ProcessorFunc` is a wrapper on top of standard Go `http.HandlerFunc` type, which adds a convenient abstraction layer. It helps you to work on business logic instead of worrying about boilerplate stuff ( such as decoding requests, logging, context propagation, preparing responses, etc. ).

Let's take a look at highlights of `first` endpoint in this example.

```go
    var u examples.User
    err := req.Decode(&u)
    if err != nil {
        return nil, fmt.Errorf("failed to decode request: %w", err)
    }
```
At the first step, the incoming request is decoding into a `User` type is defined in the `examples` package. That's the `Protobuf` generated type.

`Patron` provides unified API for decoding requests that understands how to decode an incoming request by checking `Content-Type` and `Accept` HTTP headers ( see [determineEncoding](https://github.com/beatlabs/patron/blob/master/component/http/handler.go#L53) function ).

The next step is encoding received `User` into Protobuf and send it further.
```go
    b, err := protobuf.Encode(&u)
    if err != nil {
        return nil, fmt.Errorf("failed create request: %w", err)
    }
```
The only thing you should pay attention to is that `Patron` provides `encoding` package with built-in helpers for encoding/decoding operations. 

```go
    secondRouteReq, err := http.NewRequest("GET", "http://localhost:50001", bytes.NewReader(b))
    if err != nil {
    	return nil, fmt.Errorf("failed create request: %w", err)
    }
    secondRouteReq.Header.Add("Content-Type", protobuf.Type)
    secondRouteReq.Header.Add("Accept", protobuf.Type)
    secondRouteReq.Header.Add("Authorization", "Apikey 123456")
	
    cl, err := clienthttp.New(clienthttp.Timeout(5 * time.Second))
    if err != nil {
    	return nil, err
    }
    rsp, err := cl.Do(ctx, secondRouteReq)
    if err != nil {
    	return nil, fmt.Errorf("failed to post to second service: %w", err)
    }
```
The sending part doesn't have new things. The creation of `secondRouteReq` is pretty usual.  

For sending `secondRouteReq` `Patron` a built-in HTTP client is used. What's differs this client from stdlib client is integrated tracing and the circuit breaker ability.

The last part of this handler is the response.
```go
    return patronhttp.NewResponse(fmt.Sprintf("got %s from second HTTP route", rsp.Status)), nil
```

It's relatively easy to use. What happens under the hood, `Patron` encodes the response payload and sends it to the caller.   Encoding depends on the headers mentioned above ( see [handleSuccess](https://github.com/beatlabs/patron/blob/master/component/http/handler.go#L125) function )

That's it about `ProcessorFunc` so far.

##### Middlewares
`Patron` wouldn't be fully complete without middleware supporting. It also comes out-of-the-box.

For using middleware `Patron` provides `WithMiddlewares` method that accepts `http.Middleware` implementation.

In this case, it's defined a specialized CORS middleware:
```go
    middlewareCors := func(h http.Handler) http.Handler {
    	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    		w.Header().Add("Access-Control-Allow-Origin", "*")
    		w.Header().Add("Access-Control-Allow-Methods", "GET, POST")
    		w.Header().Add("Access-Control-Allow-Headers", "Origin, Authorization, Content-Type")
    		w.Header().Add("Access-Control-Allow-Credentials", "Allow")
			h.ServeHTTP(w, r)
		})
	}
 ```

This middleware adds a few extra HTTP headers to the incoming request and delegates execution to the next HTTP handler.
> If you're not aware of middleware concept [here's](https://www.alexedwards.net/blog/making-and-using-middleware), a good explanation.

`Patron` provides a few ready middlewares you can use in your service. 
- [NewAuthMiddleware](https://godoc.org/github.com/beatlabs/patron/component/http#NewAuthMiddleware)
- [NewLoggingTracingMiddleware](https://godoc.org/github.com/beatlabs/patron/component/http#NewLoggingTracingMiddleware) - it's used if `WithTrace` is set.
- [NewRecoveryMiddleware](https://godoc.org/github.com/beatlabs/patron/component/http#NewRecoveryMiddleware)

##### SIGHUP handling
There is a way to catch `SIGHUP` signal and handle it in your way. Sometimes it might be helpful (e.g., configuration reloading). In this example, you can see that a signal handler is being set by `WithSIGHUP(sig)` where `sig` is a simple function:
```go
    sig := func() {
	fmt.Println("exit gracefully...")
	os.Exit(0)
    }
```

### Let's run it!
We're going to send a simple query to a service `first` from this example. As you remember, the service sends a request to another service. That service is called service `second`. Let's run it.

#### 1. Spin up the environment.
```sh
$> docker-compose -f ./examples/docker-compose.yml up -d
```
#### 2. Run the examples
```sh
$> go run ./examples/first/main.go
$> go run ./examples/second/main.go
```
#### 3. Send a request
```sh
$> bash ./examples/start_processing.sh
```

#### 4. Observe it
Check the collected traces and metrics during this call.

- Jaeger: `http://localhost:16686/search`
- Prometheus: `http://localhost:9090` (e.g. `second_http_requests`)

### Next step
- [Example#2](../second/)


