# HTTP v2

HTTP v2 tries to create a very thin layer around any http router provided by the end user.  
Patron provides an implementation for the simple and excellent [httprouter](https://github.com/julienschmidt/httprouter) which is 
available in the `httprouter` available in Patron.  
Because the HTTP component relies on the standard Go `http.Handler` any implementation can be provided as long as the interface is implemented.  
The component is responsible for running the HTTP server using the handler and terminating on request.

The component provides out of the box:

- HTTP lifecycle endpoints (liveness and readiness)
- metrics and distributed traces
- profiling using the standard `net/http/pprof` package

The [example](../../examples) folder contains various use cases.

The component create is then passed to the functional option of the Patron services, which replaces the default legacy HTTP handler
with the new HTTP handler.

## httprouter

The implementation provides the following:

- file server route for helping us serving files e.g. SPA
- functional options to set up the handler e.g. live and readiness checks, middlewares, routes, compression, etc.

In the handler creation process are adding automatically to every route our standard middlewares that handle:

- recovery from panics
- configurable logging
- tracing
- metrics
- compression

The router also provides functional options to configure certain aspects of its behavior like:

- readiness and liveness checks
- deflate level
- app name and version in response headers
- etc.
