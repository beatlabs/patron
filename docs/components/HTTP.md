* [HTTP](#http)
  * [HTTP lifecycle endpoints](#http-lifecycle-endpoints)
  * [HTTP Middlewares](#http-middlewares)
    * [Middleware Chain](#middleware-chain)
    * [Helper Middlewares](#helper-middlewares)
  * [HTTP Routes](#http-routes)
    * [HTTP Method](#http-method)
    * [Processor](#processor)
    * [File Server](#file-server)
    * [Raw RouteBuilder Constructor](#raw-routebuilder-constructor)
    * [Middlewares per Route](#middlewares-per-route)
    * [Security](#security)
    * [Tracing](#tracing)
    * [HTTP Caching](#http-caching)

# HTTP

The HTTP component provides the functionality for creating an HTTP server exposing the relevant routes. 
It wraps the logic and handles the boilerplate for the `net.http` go package.

The way to initialise an HTTP component is through the `patron http.Builder` struct.
```go
// NewBuilder initiates the HTTP component builder chain.
// The builder instantiates the component using default values for
// HTTP Port, Alive/Ready check functions and Read/Write timeouts.
func NewBuilder() *Builder {
	// ...
}

// WithSSL sets the filenames for the Certificate and Keyfile, in order to enable SSL.
func (cb *Builder) WithSSL(c, k string) *Builder {
	// ..
}

// WithRoutesBuilder adds routes builder to the HTTP component.
func (cb *Builder) WithRoutesBuilder(rb *RoutesBuilder) *Builder {
	// ...
}

// WithMiddlewares adds middlewares to the HTTP component.
func (cb *Builder) WithMiddlewares(mm ...MiddlewareFunc) *Builder {
	// ...
}

// WithReadTimeout sets the Read Timeout for the HTTP component.
func (cb *Builder) WithReadTimeout(rt time.Duration) *Builder {
	// ...
}

// WithWriteTimeout sets the Write Timeout for the HTTP component.
func (cb *Builder) WithWriteTimeout(wt time.Duration) *Builder {
	// ...
}

// WithShutdownGracePeriod sets the Shutdown Grace Period for the HTTP component.
func (cb *Builder) WithShutdownGracePeriod(gp time.Duration) *Builder {
	// ...
}

// WithPort sets the port used by the HTTP component.
func (cb *Builder) WithPort(p int) *Builder {
	// ...
}

// WithAliveCheckFunc sets the AliveCheckFunc used by the HTTP component.
func (cb *Builder) WithAliveCheckFunc(acf AliveCheckFunc) *Builder {
	// ...
}

// WithReadyCheckFunc sets the ReadyCheckFunc used by the HTTP component.
func (cb *Builder) WithReadyCheckFunc(rcf ReadyCheckFunc) *Builder {
	// ...
}

// Create constructs the HTTP component by applying the gathered properties.
func (cb *Builder) Create() (*Component, error) {
	// ...
}
```

## HTTP lifecycle endpoints

When creating a new HTTP component, Patron will automatically create a liveness and readiness route, which can be used to probe the lifecycle of the application:

```
# liveness
GET /alive

# readiness
GET /ready
```

Both can return either a `200 OK` or a `503 Service Unavailable` status code (default: `200 OK`).

It is possible to customize their behaviour by injecting an `http.AliveCheck` and/or an `http.ReadyCheck` `OptionFunc` to the HTTP component constructor.

## HTTP Middlewares

A `MiddlewareFunc` preserves the default net/http middleware pattern.
You can create new middleware functions and pass them to Service to be chained on all routes in the default HTTP Component.

```go
type MiddlewareFunc func(next http.Handler) http.Handler

// Setup a simple middleware for CORS
newMiddleware := func(h http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Add("Access-Control-Allow-Origin", "*")
        // Next
        h.ServeHTTP(w, r)
    })
}
```

### Middleware Chain

Middlewares are invoked sequentially. The object handling this is the MiddlewareChain

```go
// MiddlewareChain chains middlewares to a handler func.
func MiddlewareChain(f http.Handler, mm ...MiddlewareFunc) http.Handler {
	for i := len(mm) - 1; i >= 0; i-- {
		f = mm[i](f)
	}
	return f
}
```

### Helper Middlewares

Patron comes with some predefined middlewares, as helper tools to inject functionality into the HTTP endpoint or individual routes.

```go

// NewRecoveryMiddleware creates a MiddlewareFunc that ensures recovery and no panic.
func NewRecoveryMiddleware() MiddlewareFunc {
    // ...
}

// NewAuthMiddleware creates a MiddlewareFunc that implements authentication using an Authenticator.
func NewAuthMiddleware(auth auth.Authenticator) MiddlewareFunc {
    // ...
}

// NewLoggingTracingMiddleware creates a MiddlewareFunc that continues a tracing span and finishes it.
// It also logs the HTTP request on debug logging level.
func NewLoggingTracingMiddleware(path string) MiddlewareFunc {
    // ...
}

// NewCachingMiddleware creates a cache layer as a middleware.
// when used as part of a middleware chain any middleware later in the chain,
// will not be executed, but the headers it appends will be part of the cache.
func NewCachingMiddleware(rc *cache.RouteCache) MiddlewareFunc {
    // ...
}

// NewCompressionMiddleware initializes a compression middleware.
// As per Section 3.5 of the HTTP/1.1 RFC, we support GZIP and Deflate as compression methods.
// https://tools.ietf.org/html/rfc2616#section-3.5
func NewCompressionMiddleware(deflateLevel int, ignoreRoutes ...string) MiddlewareFunc {


// NewRateLimitingMiddleware creates a MiddlewareFunc that adds a rate limit to a route.
// It uses golang in-built rate library to implement simple rate limiting 
//"https://pkg.go.dev/golang.org/x/time/rate"
func NewRateLimitingMiddleware(limiter *rate.Limiter) MiddlewareFunc {
	// ..
}
```

### Error Logging

It is possible to configure specific status codes that, if returned by an HTTP handler, the response's error will be logged.

This configuration must be done using the `PATRON_HTTP_STATUS_ERROR_LOGGING` environment variable. The syntax of this variable is based on PostgreSQL syntax and allows providing ranges.

For example, setting this environment variable to `409;[500,600)` that an error will be logged if an HTTP handler returns either:
* A status code 409
* A status code greater or equal than 500 (the bracket represents the inclusion) and strictly smaller than 600 (the parenthesis represents the exclusion)

Be it a specific status code or a range; each element must be delimited with `;`.

To enable error logging, we enable route tracing (`WithTrace` option).

## HTTP Routes

Each HTTP component can contain several routes. These are injected through the `RoutesBuilder`

```go
// RouteBuilder for building a route.
type RouteBuilder struct {
	// ...
}

// NewRouteBuilder constructor.
func NewRouteBuilder(path string, processor ProcessorFunc) *RouteBuilder {
    // ...
}


// WithTrace enables route tracing.
func (rb *RouteBuilder) WithTrace() *RouteBuilder {
	// ...
}

// WithMiddlewares adds middlewares.
func (rb *RouteBuilder) WithMiddlewares(mm ...MiddlewareFunc) *RouteBuilder {
	// ...
}

// WithAuth adds authenticator.
func (rb *RouteBuilder) WithAuth(auth auth.Authenticator) *RouteBuilder {
	// ...
}

// WithRouteCache adds a cache to the corresponding route
func (rb *RouteBuilder) WithRouteCache(cache cache.TTLCache, ageBounds httpcache.Age) *RouteBuilder {
	// ...
}

// Build a route.
func (rb *RouteBuilder) Build() (Route, error) {
	// ...
}
```

The main components that hold the logic for a route are the **processor** and the **middlewares**

### HTTP Method 

The method for each route cn be defined through the builder as well

```go

// MethodGet HTTP method.
func (rb *RouteBuilder) MethodGet() *RouteBuilder {
	// ...
}

// MethodHead HTTP method.
func (rb *RouteBuilder) MethodHead() *RouteBuilder {
	// ...
}

// MethodPost HTTP method.
func (rb *RouteBuilder) MethodPost() *RouteBuilder {
	// ...
}

// MethodPut HTTP method.
func (rb *RouteBuilder) MethodPut() *RouteBuilder {
	// ...
}
...
```

and for reducing boilerplate code one can also combine this in the constructor call for the Builder

```go

// NewGetRouteBuilder constructor
func NewGetRouteBuilder(path string, processor ProcessorFunc) *RouteBuilder {
	// ...
}

// NewHeadRouteBuilder constructor.
func NewHeadRouteBuilder(path string, processor ProcessorFunc) *RouteBuilder {
	// ...
}

// NewPostRouteBuilder constructor.
func NewPostRouteBuilder(path string, processor ProcessorFunc) *RouteBuilder {
	// ...
}

... 
```

### Processor

The processor is responsible for creating a `Request` by providing everything that is needed (Headers, Fields, decoder, raw io.Reader), passing it to the implementation by invoking the `Process` method and handling the `Response` or the `error` returned by the processor.

The sync package contains only a function definition along with the models needed:

```go
type ProcessorFunc func(context.Context, *Request) (*Response, error)
```

The `Request` model contains the following properties (which are provided when calling the "constructor" `NewRequest`)

- Fields, which may contain any fields associated with the request
- Raw, the raw request data (if any) in the form of a `io.Reader`
- Headers, the request headers in the form of `map[string]string`
- decode, which is a function of type `encoding.Decode` that decodes the raw reader

An exported function exists for decoding the raw io.Reader in the form of

```go
Decode(v interface{}) error
```

The `Response` model contains the following properties (which are provided when calling the "constructor" `NewResponse`)

- Payload, which may hold a struct of type `interface{}`

### File Server

```go
// NewFileServer constructor.
func NewFileServer(path string, assetsDir string, fallbackPath string) *RouteBuilder {
	// ...
}
```

The File Server exposes files from the filesystem to be accessed from the service. <br />
It has baked in support for Single Page Applications or 404 pages by providing a fallback path

Routes using the file server has to follow a pattern, by convention this path has to end in `*path`.

```go
http.NewFileServer("/some-path/*path", "...", "...")
```

The path is used to resolve where in the filesystem we should serve the file from. If no file is found we will serve the fallback path.


### Raw RouteBuilder Constructor

```go
// NewRawRouteBuilder constructor.
func NewRawRouteBuilder(path string, handler http.HandlerFunc) *RouteBuilder {
	// ...
}
```

The Raw Route Builder allows for lower level processing of the request and response objects. 
It's main difference with the Route Builder is the processing function. Which in this case is the `native` go http handler func.

```go
// The HandlerFunc type is an adapter to allow the use of
// ordinary functions as HTTP handlers. If f is a function
// with the appropriate signature, HandlerFunc(f) is a
// Handler that calls f.
type HandlerFunc func(ResponseWriter, *Request)
```

```
The Raw Route Builder constructor should be used,
if the default behaviour and assumptions of the wrapped Route Builder 
do not fit into the routes requirements or use-case.
```

### Middlewares per Route

Middlewares can also run per routes using the processor as Handler.
So using the `Route` builder:

```go
// WithMiddlewares adds middlewares.
func (rb *RouteBuilder) WithMiddlewares(mm ...MiddlewareFunc) *RouteBuilder {
	if len(mm) == 0 {
		rb.errors = append(rb.errors, errors.New("middlewares are empty"))
	}
	rb.middlewares = mm
	return rb
}
```

### Security

Users can implement the `Authenticator` interface to provide authentication capabilities for HTTP components and Routes
```go
type Authenticator interface {
  Authenticate(req *http.Request) (bool, error)
}
```

Patron also includes a ready-to-use implementation of an *API key authenticator*. 

### Tracing

One of the main features of patron is the tracing functionality for Routes. 
Tracing can either be enabled by default from the Buidler.

```go
// WithTrace enables route tracing.
func (rb *RouteBuilder) WithTrace() *RouteBuilder {
	rb.trace = true
	return rb
}
```

### HTTP Caching

The caching layer for HTTP routes is specified per Route.

```go
// RouteCache is the builder needed to build a cache for the corresponding route
type RouteCache struct {
	// cache is the ttl cache implementation to be used
	cache cache.TTLCache
	// age specifies the minimum and maximum amount for max-age and min-fresh header values respectively
	// regarding the client cache-control requests in seconds
	age age
}

func NewRouteCache(ttlCache cache.TTLCache, age Age) *RouteCache
```

**server cache**
- The **cache key** is based on the route path and the url request parameters.
- The server caches only **GET requests**.
- The server implementation must specify an **Age** parameters upon construction.
- Age with **Min=0** and **Max=0** effectively disables caching
- The route should return always the most fresh object instance.
- An **ETag header** must be always in responses that are part of the cache, representing the hash of the response.
- Requests within the time-to-live threshold, will be served from the cache. 
Otherwise the request will be handled as usual by the route processor function. 
The resulting response will be cached for future requests.
- Requests where the client control header requirements cannot be met i.e. **very low max-age** or **very high min-fresh** parameters,
will be returned to the client with a `Warning` header present in the response. 

```
Note : When a cache is used, the handler execution might be skipped.
That implies that all generic handler functionalities MUST be delegated to a custom middleware.
i.e. counting number of server client requests etc ... 
```

**Usage**

- provide the cache in the route builder
```go
NewRouteBuilder("/", handler).
	WithRouteCache(cache, http.Age{
		Min: 30 * time.Minute,
		Max: 1 * time.Hour,
	}).
    MethodGet()
```

- use the cache as a middleware
```go
NewRouteBuilder("/", handler).
    WithMiddlewares(NewCachingMiddleware(NewRouteCache(cc, Age{Max: 10 * time.Second}))).
    MethodGet()
```

**client cache-control**
The client can control the cache with the appropriate Headers
- `max-age=?` 

returns the cached instance only if the age of the instance is lower than the max-age parameter.
This parameter is bounded from below by the server option `minAge`.
This is to avoid chatty clients with no cache control policy (or very aggressive max-age policy) to effectively disable the cache
- `min-fresh=?` 
 
returns the cached instance if the time left for expiration is lower than the provided parameter.
This parameter is bounded from above by the server option `maxFresh`.
This is to avoid chatty clients with no cache control policy (or very aggressive min-fresh policy) to effectively disable the cache

- `no-cache` / `no-store`

returns a new response to the client by executing the route processing function.
NOTE : Except for cases where a `minAge` or `maxFresh` parameter has been specified in the server.
This is again a safety mechanism to avoid 'aggressive' clients put unexpected load on the server.
The server is responsible to cap the refresh time, BUT must respond with a `Warning` header in such a case.
- `only-if-cached`

expects any response that is found in the cache, otherwise returns an empty response

**metrics**

The http cache exposes several metrics, used to 
- assess the state of the cache
- help trim the optimal time-to-live policy
- identify client control interference

By default we are using prometheus as the the pre-defined metrics framework.

- `additions = misses + evictions`

Always , the cache addition operations (objects added to the cache), 
must be equal to the misses (requests that were not cached) plus the evictions (expired objects).
Otherwise we would expect to notice also an increased amount of errors or having the cache misbehaving in a different manner.

- `additions ~ misses`

If the additions and misses are comparable e.g. misses are almost as many as the additions, 
it would point to some cleanup of the cache itself. In that case the cache seems to not be able to support
the request patterns and control headers.

- `hits ~ additions`

The cache hit count represents how well the cache performs for the access patterns of client requests. 
If this number is rather low e.g. comparable to the additions, 
this would signify that probably a cache is not a good option for the access patterns at hand.

- `eviction age`

The age at which the objects are evicted from the cache is a very useful indicator. 
If the vast amount of evictions are close to the time to live setting, it would indicate a nicely working cache.
If we find that many evictions happen before the time to live threshold, clients would be making use cache-control headers.
 

**cache design reference**
- https://www.w3.org/Protocols/rfc2616/rfc2616-sec13.html
- https://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html#sec14.9

**improvement considerations**
- we can reconsider the storing of the cached objects and their age counter. That way we would avoid loading the whole object in memory,
if the object is already expired. This approach might provide considerable performance (in terms of memory utilisation) 
improvement for big response objects. 
- we could extend the metrics to use the key of the object as a label as well for more fine-grained tuning.
But this has been left out for now, due to the potentially huge number of metric objects.
We can review according to usage or make this optional in the future.
- improve the serialization performance for the cache response objects

## Rate Limiting
- Uses golang in-built rate library to implement simple rate limiting 
- We could pass the limit and burst values as parameters. 
- Limit and burst values are integers. 
  Note: A zero Burst allows no events, unless limit == Inf. More details here - https://pkg.go.dev/golang.org/x/time/rate

**Usage**

- provide the rate limiting in the route builder
```go
NewGetRouteBuilder("/", getHandler).WithRateLimiting(limit, burst)
```

- use the rate limiting as a middleware
```go
NewRouteBuilder("/", handler).
    WithMiddlewares(NewRateLimitingMiddleware(rate.NewLimiter(limit, burst))).
    MethodGet()
```