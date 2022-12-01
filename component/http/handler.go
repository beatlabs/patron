package http

import (
	"errors"
	"net/http"
	"strings"

	"github.com/beatlabs/patron/correlation"
	"github.com/beatlabs/patron/encoding"
	"github.com/beatlabs/patron/encoding/json"
	"github.com/beatlabs/patron/encoding/protobuf"
	"github.com/beatlabs/patron/log"
	"github.com/julienschmidt/httprouter"
)

func handler(hnd ProcessorFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ct, dec, enc, err := determineEncoding(r.Header)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusUnsupportedMediaType), http.StatusUnsupportedMediaType)
			return
		}
		prepareResponse(w, ct)

		f := extractFields(r)
		for k, v := range ExtractParams(r) {
			f[k] = v
		}

		// TODO : for cached responses this becomes inconsistent, to be fixed in #160
		// the corID will be passed to all consecutive responses
		// if it was missing from the initial request
		corID := getOrSetCorrelationID(r.Header)
		ctx := correlation.ContextWithID(r.Context(), corID)
		logger := log.Sub(map[string]interface{}{correlation.ID: corID})
		ctx = log.WithContext(ctx, logger)

		h := extractHeaders(r.Header)

		req := NewRequest(f, r.Body, h, dec)

		rsp, err := hnd(ctx, req)
		if err != nil {
			handleError(logger, w, enc, err)
			return
		}

		err = handleSuccess(w, r, rsp, enc)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}

func getOrSetCorrelationID(h http.Header) string {
	cor, ok := h[correlation.HeaderID]
	if !ok {
		corID := correlation.New()
		h.Set(correlation.HeaderID, corID)
		return corID
	}
	if len(cor) == 0 {
		corID := correlation.New()
		h.Set(correlation.HeaderID, corID)
		return corID
	}
	if cor[0] == "" {
		corID := correlation.New()
		h.Set(correlation.HeaderID, corID)
		return corID
	}
	return cor[0]
}

func determineEncoding(h http.Header) (string, encoding.DecodeFunc, encoding.EncodeFunc, error) {
	cth, cok := h[encoding.ContentTypeHeader]
	ach, aok := h[encoding.AcceptHeader]

	// No headers default to JSON
	if !cok && !aok {
		return json.TypeCharset, json.Decode, json.Encode, nil
	}

	var enc encoding.EncodeFunc
	var dec encoding.DecodeFunc
	var ct string

	if cok {
		switch cth[0] {
		case "*/*", json.Type, json.TypeCharset:
			enc = json.Encode
			dec = json.Decode
			ct = json.TypeCharset
		case protobuf.Type, protobuf.TypeGoogle:
			enc = protobuf.Encode
			dec = protobuf.Decode
			ct = protobuf.Type
		default:
			return "", nil, nil, errors.New("content type Header not supported")
		}
	}

	if aok {
		var err error
		encFound := false

		ah := getMultiValueHeaders(ach[0])
		for _, v := range ah {
			ct, dec, enc, err = getSingleHeaderEncoding(v)
			if err == nil {
				encFound = true
				break
			}
		}
		// No valid headers found after going through all headers
		if !encFound {
			return "", nil, nil, errors.New("accept header not supported")
		}
	}

	return ct, dec, enc, nil
}

func getSingleHeaderEncoding(header string) (string, encoding.DecodeFunc, encoding.EncodeFunc, error) {
	var enc encoding.EncodeFunc
	var dec encoding.DecodeFunc
	var ct string

	parts := strings.SplitN(header, ";", 2)
	switch parts[0] {
	case "*/*", "*", "identity", json.Type, json.TypeCharset:
		enc = json.Encode
		dec = json.Decode
		ct = json.TypeCharset
	case protobuf.Type, protobuf.TypeGoogle:
		enc = protobuf.Encode
		dec = protobuf.Decode
		ct = protobuf.Type
	default:
		return "", nil, nil, errors.New("accept header not supported")
	}

	return ct, dec, enc, nil
}

func getMultiValueHeaders(header string) []string {
	if !strings.Contains(header, ",") {
		return []string{header}
	}

	splitHeaders := strings.Split(header, ",")

	trimmedHeaders := make([]string, 0, len(splitHeaders))
	for _, v := range splitHeaders {
		trimmedHeaders = append(trimmedHeaders, strings.TrimSpace(v))
	}

	return trimmedHeaders
}

func extractFields(r *http.Request) map[string]string {
	f := make(map[string]string)

	for name, values := range r.URL.Query() {
		f[name] = values[0]
	}
	return f
}

func extractHeaders(header http.Header) Header {
	h := make(map[string]string)

	for name, values := range header {
		for _, value := range values {
			if len(value) > 0 {
				h[strings.ToUpper(name)] = value
			}
		}
	}
	return h
}

func handleSuccess(w http.ResponseWriter, r *http.Request, rsp *Response, enc encoding.EncodeFunc) error {
	if rsp == nil {
		w.WriteHeader(http.StatusNoContent)
		return nil
	}

	p, err := enc(rsp.Payload)
	if err != nil {
		return err
	}

	if r.Method == http.MethodPost {
		w.WriteHeader(http.StatusCreated)
	}

	propagateHeaders(rsp.Header, w.Header())

	_, err = w.Write(p)
	return err
}

func handleError(logger log.Logger, w http.ResponseWriter, enc encoding.EncodeFunc, err error) {
	var errAs *Error
	if errors.As(err, &errAs) {
		p, encErr := enc(errAs.payload)
		if encErr != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		for k, v := range errAs.headers {
			w.Header().Set(k, v)
		}
		w.WriteHeader(errAs.code)
		if _, err := w.Write(p); err != nil {
			logger.Errorf("failed to write Response: %v", err)
		}
		return
	}

	// Using http.Error helper hijacks the content type Header of the Response returning plain text Payload.
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func prepareResponse(w http.ResponseWriter, ct string) {
	w.Header().Set(encoding.ContentTypeHeader, ct)
}

// ExtractParams extracts dynamic URL parameters using httprouter's functionality.
func ExtractParams(r *http.Request) map[string]string {
	par := httprouter.ParamsFromContext(r.Context())
	if len(par) == 0 {
		return make(map[string]string)
	}
	p := make(map[string]string)
	for _, v := range par {
		p[v.Key] = v.Value
	}
	return p
}
