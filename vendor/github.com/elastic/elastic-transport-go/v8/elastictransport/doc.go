// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
//

/*
Package elastictransport provides the transport layer for the Elastic clients.

# Creating a Client

Use [NewClient] with functional [Option] values to create a transport client:

	u, _ := url.Parse("https://localhost:9200")
	tp, err := elastictransport.NewClient(
	    elastictransport.WithURLs(u),
	    elastictransport.WithBasicAuth("elastic", "changeme"),
	    elastictransport.WithMaxRetries(5),
	)

Options are applied in order; when the same setting is specified more than once
the last value wins.

The older [New] + [Config] API is still available for backwards compatibility
but is deprecated. New code should prefer [NewClient].

# Validation and Debugging

Use [ValidateOptions] or [Options.Validate] to check option values before
creating a client:

	opts := elastictransport.Options{
	    elastictransport.WithURLs(u),
	    elastictransport.WithMaxRetries(5),
	}
	if err := opts.Validate(); err != nil {
	    log.Fatal(err)
	}

Each [Option] implements [fmt.Stringer] with sensitive values redacted by
default. Call [Option.Describe](true) to include secrets:

	fmt.Println(opt)               // WithAPIKey(****)
	fmt.Println(opt.Describe(true)) // WithAPIKey("Zm9vYmFy")

Use [Options.Visit] to iterate over all options programmatically.

# HTTP Transport

The default HTTP transport of the client is http.Transport; use [WithTransport]
to customize it.

# Retries

The package will automatically retry requests on network-related errors, and on
specific response status codes (by default 502, 503, 504). Use [WithRetry] to
set the maximum retries and retryable status codes in one call, or use
[WithMaxRetries] and [WithRetryOnStatus] individually. Use [WithDisableRetry] to
disable the retry behaviour altogether.

By default, the retry will be performed without any delay; to configure a
backoff interval, use [WithRetryBackoff].

# Connection Management

When multiple addresses are passed via [WithURLs], the package will use them in
a round-robin fashion, and will keep track of live and dead nodes. The status
of dead nodes is checked periodically.

To customize the node selection behaviour, provide a [Selector] implementation
via [WithSelector]. To replace the connection pool entirely, provide a custom
[ConnectionPool] implementation via [WithConnectionPoolFunc]. Discovery prefers
in-place Update() when the pool implements [UpdatableConnectionPool]; pool
replacement via [WithConnectionPoolFunc] happens only when Update() is not
available. Custom pools are synchronized by default; implement
[ConcurrentSafeConnectionPool] to opt out when your custom pool is already safe
for concurrent use.

# Logging

Use [WithLeveledLogger] to supply a structured, leveled logger for
transport-internal events (connection management, node discovery). The
[LeveledLogger] interface uses the same (msg, keysAndValues...) convention
as [log/slog]:

	tp, err := elastictransport.NewClient(
	    elastictransport.WithURLs(u),
	    elastictransport.WithLeveledLogger(&elastictransport.SlogLogger{
	        Logger: slog.New(slog.NewJSONHandler(os.Stderr, nil)),
	    }),
	)

Add [LoggingInterceptor] to also log request/response round-trips through
the same logger. Successful requests are logged at Info level; errors at
Error level:

	tp, err := elastictransport.NewClient(
	    elastictransport.WithURLs(u),
	    elastictransport.WithLeveledLogger(&elastictransport.SlogLogger{
	        Logger: slog.Default(),
	    }),
	    elastictransport.WithInterceptors(
	        elastictransport.LoggingInterceptor(false, false),
	    ),
	)

The logger is injected into the request context during [Client.Perform],
making it available to custom [InterceptorFunc] implementations via
[LoggerFromContext]. Callers can override the logger per-request using
[ContextWithLogger].

The [sloghandler] sub-package provides drop-in [log/slog.Handler]
replacements for each deprecated logger: [sloghandler.NewTextHandler],
[sloghandler.NewColorHandler], [sloghandler.NewCurlHandler], and
[sloghandler.NewJSONECSHandler].

[OTelContextAttrs] can be used as [SlogLogger.ContextAttrs] to
automatically include OpenTelemetry trace_id and span_id in every log entry.

# Custom Pools and Logging

The [LeveledLogger] configured via [WithLeveledLogger] is used in two places:

  - Request/response logging via [LoggingInterceptor], which works with any
    pool type (built-in or custom).
  - Pool-internal events (node resurrection, health checks) in the built-in
    [statusConnectionPool].

Custom pools provided via [WithConnectionPoolFunc] do not receive the
transport's [LeveledLogger]. Pool-internal logging is a feature of the
built-in pools. If your custom pool needs logging, accept a logger in its
constructor.

The older [Logger] interface, [WithLogger], [WithDebugLogger], and the bundled
loggers ([TextLogger], [ColorLogger], [CurlLogger], [JSONLogger]) are
deprecated but remain fully functional.

# Metrics

Use [WithMetrics] to enable metric collection and export.
*/
package elastictransport
