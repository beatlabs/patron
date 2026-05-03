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

package elastictransport

import (
	"compress/gzip"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Option configures a Client. Use the With* functions to obtain Option values.
//
// Each Option carries a human-readable description accessible via [Option.String]
// (secrets redacted) or [Option.Describe](true) (secrets visible). This makes
// it safe to log options by default while still allowing full output when needed.
type Option struct {
	name     string
	masked   string
	unmasked string
	apply    func(*Config) error
}

// Name returns the constructor function name (e.g. "WithMaxRetries").
func (o Option) Name() string { return o.name }

// String returns a human-readable description with sensitive values redacted.
// It implements [fmt.Stringer].
func (o Option) String() string { return o.masked }

// Describe returns a human-readable description of the option. When
// showSensitive is true, secret values (API keys, passwords, tokens,
// certificates) are included; otherwise they are replaced with "****".
func (o Option) Describe(showSensitive bool) string {
	if showSensitive {
		return o.unmasked
	}
	return o.masked
}

func newOption(name, desc string, apply func(*Config) error) Option {
	return Option{name: name, masked: desc, unmasked: desc, apply: apply}
}

func newSensitiveOption(name, masked, unmasked string, apply func(*Config) error) Option {
	return Option{name: name, masked: masked, unmasked: unmasked, apply: apply}
}

// WithURLs sets the Elasticsearch node URLs the transport will connect to.
// Multiple URLs enable round-robin load balancing and failover.
func WithURLs(urls ...*url.URL) Option {
	parts := make([]string, len(urls))
	for i, u := range urls {
		parts[i] = u.String()
	}
	desc := fmt.Sprintf("WithURLs(%s)", strings.Join(parts, ", "))
	return newOption("WithURLs", desc, func(c *Config) error {
		c.URLs = urls
		return nil
	})
}

// WithBasicAuth configures HTTP Basic Authentication with the given username
// and password.
func WithBasicAuth(username, password string) Option {
	masked := fmt.Sprintf("WithBasicAuth(%q, ****)", username)
	unmasked := fmt.Sprintf("WithBasicAuth(%q, %q)", username, password)
	return newSensitiveOption("WithBasicAuth", masked, unmasked, func(c *Config) error {
		c.Username = username
		c.Password = password
		return nil
	})
}

// WithAPIKey configures API Key authentication. The key should be the
// base64-encoded value returned by the Elasticsearch create API key endpoint.
func WithAPIKey(apiKey string) Option {
	return newSensitiveOption("WithAPIKey", "WithAPIKey(****)",
		fmt.Sprintf("WithAPIKey(%q)", apiKey),
		func(c *Config) error {
			c.APIKey = apiKey
			return nil
		})
}

// WithServiceToken configures service token authentication using the given
// bearer token.
func WithServiceToken(token string) Option {
	return newSensitiveOption("WithServiceToken", "WithServiceToken(****)",
		fmt.Sprintf("WithServiceToken(%q)", token),
		func(c *Config) error {
			c.ServiceToken = token
			return nil
		})
}

// WithHeader sets global HTTP headers that are added to every request.
// Per-request headers set on the *http.Request take precedence.
func WithHeader(header http.Header) Option {
	desc := fmt.Sprintf("WithHeader(%d entries)", len(header))
	return newOption("WithHeader", desc, func(c *Config) error {
		c.Header = header
		return nil
	})
}

// WithCACert sets the PEM-encoded CA certificate used to verify the server's
// TLS certificate. This requires the underlying transport to be an
// *http.Transport.
func WithCACert(cert []byte) Option {
	desc := fmt.Sprintf("WithCACert(len=%d)", len(cert))
	return newSensitiveOption("WithCACert", "WithCACert(****)", desc,
		func(c *Config) error {
			c.CACert = cert
			return nil
		})
}

// WithCertificateFingerprint configures SHA-256 certificate fingerprint
// verification. When set, the transport verifies that at least one certificate
// in the chain matches the hex-encoded fingerprint, independent of CA trust.
func WithCertificateFingerprint(fingerprint string) Option {
	return newSensitiveOption("WithCertificateFingerprint",
		"WithCertificateFingerprint(****)",
		fmt.Sprintf("WithCertificateFingerprint(%q)", fingerprint),
		func(c *Config) error {
			c.CertificateFingerprint = fingerprint
			return nil
		})
}

// WithUserAgent sets the User-Agent header value sent with every request.
func WithUserAgent(ua string) Option {
	return newOption("WithUserAgent", fmt.Sprintf("WithUserAgent(%q)", ua),
		func(c *Config) error {
			c.UserAgent = ua
			return nil
		})
}

// WithTransport sets the http.RoundTripper used for HTTP requests. If not set,
// a clone of http.DefaultTransport is used.
func WithTransport(rt http.RoundTripper) Option {
	return newOption("WithTransport", fmt.Sprintf("WithTransport(%T)", rt),
		func(c *Config) error {
			c.Transport = rt
			return nil
		})
}

// WithLogger sets the Logger used to log request and response information.
// WithLogger sets a round-trip [Logger] for request/response logging.
//
// Deprecated: Use [WithLeveledLogger] instead. When both are set,
// WithLogger takes precedence for round-trip logging.
func WithLogger(l Logger) Option {
	return newOption("WithLogger", fmt.Sprintf("WithLogger(%T)", l),
		func(c *Config) error {
			c.Logger = l
			return nil
		})
}

// WithLeveledLogger sets a structured, leveled logger for transport-internal
// events such as node discovery, connection resurrection, and connection
// removal.
//
// The logger is also injected into the request context during [Client.Perform],
// making it available to [InterceptorFunc] implementations via
// [LoggerFromContext]. Use [LoggingInterceptor] to add round-trip request/response
// logging through the same logger.
//
// The provided logger must be safe for concurrent use. See [SlogLogger] for a
// ready-made adapter that wraps [*slog.Logger].
func WithLeveledLogger(l LeveledLogger) Option {
	return newOption("WithLeveledLogger", fmt.Sprintf("WithLeveledLogger(%T)", l),
		func(c *Config) error {
			c.LeveledLogger = l
			return nil
		})
}

// WithSelector sets the Selector used to pick connections from the pool.
func WithSelector(s Selector) Option {
	return newOption("WithSelector", fmt.Sprintf("WithSelector(%T)", s),
		func(c *Config) error {
			c.Selector = s
			return nil
		})
}

// WithConnectionPoolFunc sets a factory function for creating a custom
// connection pool. Pools are synchronised by default; implement
// ConcurrentSafeConnectionPool to opt out when your pool is already safe for
// concurrent use.
//
// Custom pools do not receive the transport's LeveledLogger. Request/response
// logging via LoggingInterceptor works regardless of pool type, but
// pool-internal logging (e.g. node health) must be handled by the pool itself.
//
// During discovery, if the current pool implements UpdatableConnectionPool,
// discovery prefers in-place Update() and this function is only called when
// Update() is not available.
func WithConnectionPoolFunc(f func([]*Connection, Selector) ConnectionPool) Option {
	return newOption("WithConnectionPoolFunc", "WithConnectionPoolFunc(func)",
		func(c *Config) error {
			c.ConnectionPoolFunc = f
			return nil
		})
}

// WithDisableRetry disables automatic request retries. When set,
// RetryOnStatus, RetryOnError, MaxRetries, and RetryBackoff are ignored.
func WithDisableRetry() Option {
	return newOption("WithDisableRetry", "WithDisableRetry()",
		func(c *Config) error {
			c.DisableRetry = true
			return nil
		})
}

// WithRetry configures retry behaviour with the given maximum number of
// retries and optional HTTP status codes that trigger a retry. If no status
// codes are provided, the defaults (502, 503, 504) are used.
//
// maxRetries must be non-negative.
func WithRetry(maxRetries int, onStatus ...int) Option {
	desc := fmt.Sprintf("WithRetry(%d", maxRetries)
	if len(onStatus) > 0 {
		desc += fmt.Sprintf(", %v", onStatus)
	}
	desc += ")"
	return newOption("WithRetry", desc, func(c *Config) error {
		if maxRetries < 0 {
			return fmt.Errorf("WithRetry: maxRetries must be >= 0, got %d", maxRetries)
		}
		c.MaxRetries = maxRetries
		if len(onStatus) > 0 {
			c.RetryOnStatus = onStatus
		}
		return nil
	})
}

// WithRetryOnStatus sets the HTTP status codes that trigger a retry.
// The defaults are 502, 503, and 504.
func WithRetryOnStatus(statuses ...int) Option {
	return newOption("WithRetryOnStatus", fmt.Sprintf("WithRetryOnStatus(%v)", statuses),
		func(c *Config) error {
			c.RetryOnStatus = statuses
			return nil
		})
}

// WithRetryOnError sets a function that decides whether a transport-level error
// should trigger a retry. The function receives the original request and the
// error. Returning true allows the retry.
func WithRetryOnError(fn func(*http.Request, error) bool) Option {
	return newOption("WithRetryOnError", "WithRetryOnError(func)",
		func(c *Config) error {
			c.RetryOnError = fn
			return nil
		})
}

// WithMaxRetries sets the maximum number of retries for a failed request.
// The default is 3. n must be non-negative.
func WithMaxRetries(n int) Option {
	return newOption("WithMaxRetries", fmt.Sprintf("WithMaxRetries(%d)", n),
		func(c *Config) error {
			if n < 0 {
				return fmt.Errorf("WithMaxRetries: value must be >= 0, got %d", n)
			}
			c.MaxRetries = n
			return nil
		})
}

// WithRetryBackoff sets a backoff function called between retries. The function
// receives the retry attempt number (starting at 1) and returns the duration to
// wait before the next attempt.
func WithRetryBackoff(fn func(attempt int) time.Duration) Option {
	return newOption("WithRetryBackoff", "WithRetryBackoff(func)",
		func(c *Config) error {
			c.RetryBackoff = fn
			return nil
		})
}

// WithCompression enables gzip compression for request bodies using a pooled
// gzip writer. An optional compression level may be provided (see the
// compress/gzip constants, e.g. gzip.BestSpeed). When omitted, gzip.DefaultCompression
// is used.
func WithCompression(level ...int) Option {
	var desc string
	if len(level) > 0 {
		desc = fmt.Sprintf("WithCompression(%d)", level[0])
	} else {
		desc = "WithCompression()"
	}
	return newOption("WithCompression", desc, func(c *Config) error {
		c.CompressRequestBody = true
		c.PoolCompressor = true
		if len(level) > 0 {
			l := level[0]
			if l < gzip.HuffmanOnly || l > gzip.BestCompression {
				return fmt.Errorf("WithCompression: invalid gzip level %d (valid range: %d to %d)",
					l, gzip.HuffmanOnly, gzip.BestCompression)
			}
			c.CompressRequestBodyLevel = l
		}
		return nil
	})
}

// WithMetrics enables internal transport metrics collection.
func WithMetrics() Option {
	return newOption("WithMetrics", "WithMetrics()", func(c *Config) error {
		c.EnableMetrics = true
		return nil
	})
}

// WithDebugLogger enables a debug logger that writes connection-management
// information via [slog.Default].
//
// Deprecated: Use [WithLeveledLogger] instead for full control over log
// destination, levels, and structured output.
func WithDebugLogger() Option {
	return newOption("WithDebugLogger", "WithDebugLogger()", func(c *Config) error {
		c.EnableDebugLogger = true
		return nil
	})
}

// WithInstrumentation sets the Instrumentation used for tracing and metrics
// propagation (e.g. OpenTelemetry).
func WithInstrumentation(i Instrumentation) Option {
	return newOption("WithInstrumentation", fmt.Sprintf("WithInstrumentation(%T)", i),
		func(c *Config) error {
			c.Instrumentation = i
			return nil
		})
}

// WithDiscoverNodesInterval sets the interval at which the transport
// automatically discovers cluster nodes. A zero value disables periodic
// discovery.
func WithDiscoverNodesInterval(d time.Duration) Option {
	return newOption("WithDiscoverNodesInterval", fmt.Sprintf("WithDiscoverNodesInterval(%s)", d),
		func(c *Config) error {
			c.DiscoverNodesInterval = d
			return nil
		})
}

// WithDiscoverNodeTimeout sets the per-request timeout for node discovery
// calls. If not set, discovery uses the default transport timeout.
func WithDiscoverNodeTimeout(d time.Duration) Option {
	return newOption("WithDiscoverNodeTimeout", fmt.Sprintf("WithDiscoverNodeTimeout(%s)", d),
		func(c *Config) error {
			c.DiscoverNodeTimeout = &d
			return nil
		})
}

// WithInterceptors sets the request/response interceptors applied on every
// round trip. Interceptors are composed in order and cannot be changed after
// transport creation.
func WithInterceptors(interceptors ...InterceptorFunc) Option {
	desc := fmt.Sprintf("WithInterceptors(%d funcs)", len(interceptors))
	return newOption("WithInterceptors", desc, func(c *Config) error {
		c.Interceptors = interceptors
		return nil
	})
}
