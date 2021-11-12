package v2

import (
	"errors"
	"time"

	patronhttp "github.com/beatlabs/patron/component/http"
)

// OptionFunc definition for configuring the component in a functional way.
type OptionFunc func(*Component) error

// AliveCheck functional option.
func AliveCheck(acf patronhttp.AliveCheckFunc) OptionFunc {
	return func(cmp *Component) error {
		if acf == nil {
			return errors.New("alive check function was nil")
		}

		cmp.aliveCheck = acf
		return nil
	}
}

// ReadyCheck functional option.
func ReadyCheck(rcf patronhttp.ReadyCheckFunc) OptionFunc {
	return func(cmp *Component) error {
		if rcf == nil {
			return errors.New("ready check function was nil")
		}

		cmp.readyCheck = rcf
		return nil
	}
}

// SSL functional option.
func SSL(cert, key string) OptionFunc {
	return func(cmp *Component) error {
		if cert == "" || key == "" {
			return errors.New("cert file or key file was empty")
		}

		cmp.certFile = cert
		cmp.keyFile = key
		return nil
	}
}

// // WithMiddlewares adds middlewares to the HTTP component.
// func (cb *Builder) WithMiddlewares(mm ...MiddlewareFunc) *Builder {
// 	if len(mm) == 0 {
// 		cb.errors = append(cb.errors, errors.New("empty list of middlewares provided"))
// 	} else {
// 		log.Debug("setting middlewares")
// 		cb.middlewares = append(cb.middlewares, mm...)
// 	}

// 	return cb
// }

// ReadTimeout functional option.
func ReadTimeout(rt time.Duration) OptionFunc {
	return func(cmp *Component) error {
		if rt <= 0*time.Second {
			return errors.New("negative or zero read timeout provided")
		}
		cmp.readTimeout = rt
		return nil
	}
}

// WriteTimeout functional option.
func WriteTimeout(wt time.Duration) OptionFunc {
	return func(cmp *Component) error {
		if wt <= 0*time.Second {
			return errors.New("negative or zero write timeout provided")
		}
		cmp.writeTimeout = wt
		return nil
	}
}

// ShutdownGracePeriod functional option.
func ShutdownGracePeriod(gp time.Duration) OptionFunc {
	return func(cmp *Component) error {
		if gp <= 0*time.Second {
			return errors.New("negative or zero shutdown grace period timeout provided")
		}
		cmp.shutdownGracePeriod = gp
		return nil
	}
}

// Port functional option.
func Port(port int) OptionFunc {
	return func(cmp *Component) error {
		if port <= 0 || port > 65535 {
			return errors.New("invalid HTTP Port provided")
		}
		cmp.port = port
		return nil
	}
}

// DeflateLevel functional option.
// Sets the level of compression for Deflate; based on https://golang.org/pkg/compress/flate/
// Levels range from 1 (BestSpeed) to 9 (BestCompression); higher levels typically run slower but compress more.
// Level 0 (NoCompression) does not attempt any compression; it only adds the necessary DEFLATE framing.
// Level -1 (DefaultCompression) uses the default compression level.
// Level -2 (HuffmanOnly) will use Huffman compression only, giving a very fast compression for all types of input, but sacrificing considerable compression efficiency.
func DeflateLevel(level int) OptionFunc {
	return func(cmp *Component) error {
		if level < -2 || level > 9 {
			return errors.New("provided deflate level value not in the [-2, 9] range")
		}
		cmp.deflateLevel = level
		return nil
	}
}

// UncompressedPaths functional option.
// Specifies which routes should be excluded from compression
// Any trailing slashes are trimmed, so we match both /metrics/ and /metrics?seconds=30
func UncompressedPaths(r ...string) OptionFunc {
	return func(cmp *Component) error {
		if len(r) == 0 {
			return errors.New("uncompressed paths are missing")
		}

		res := make([]string, 0, len(r))
		for _, e := range r {
			for len(e) > 1 && e[len(e)-1] == '/' {
				e = e[0 : len(e)-1]
			}
			res = append(res, e)
		}

		cmp.uncompressedPaths = append(cmp.uncompressedPaths, res...)
		return nil
	}
}
