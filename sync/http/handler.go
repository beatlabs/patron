package http

import (
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	"github.com/thebeatapp/patron/encoding"
	"github.com/thebeatapp/patron/encoding/json"
	"github.com/thebeatapp/patron/encoding/protobuf"
	"github.com/thebeatapp/patron/errors"
	"github.com/thebeatapp/patron/log"
	"github.com/thebeatapp/patron/sync"
)

func handler(hnd sync.ProcessorFunc) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		ct, dec, enc, err := determineEncoding(r)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusUnsupportedMediaType), http.StatusUnsupportedMediaType)
			return
		}
		prepareResponse(w, ct)

		f := extractFields(r)
		for k, v := range extractParams(r) {
			f[k] = v
		}

		h := extractHeaders(r)

		ctx := log.WithContext(r.Context(), log.Sub(map[string]interface{}{"requestID": uuid.New().String()}))
		req := sync.NewRequest(f, r.Body, h, dec)
		rsp, err := hnd(ctx, req)
		if err != nil {
			handleError(w, enc, err)
			return
		}

		err = handleSuccess(w, r, rsp, enc)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}

func determineEncoding(r *http.Request) (string, encoding.DecodeFunc, encoding.EncodeFunc, error) {
	cth, cok := r.Header[encoding.ContentTypeHeader]
	ach, aok := r.Header[encoding.AcceptHeader]

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
			return "", nil, nil, errors.New("content type header not supported")
		}
	}

	if aok {
		switch ach[0] {
		case "*/*", json.Type, json.TypeCharset:
			enc = json.Encode
			if dec == nil {
				dec = json.Decode
			}
			ct = json.TypeCharset
		case protobuf.Type, protobuf.TypeGoogle:
			enc = protobuf.Encode
			if dec == nil {
				dec = protobuf.Decode
			}
			ct = protobuf.Type
		default:
			return "", nil, nil, errors.New("accept header not supported")
		}
	}

	return ct, dec, enc, nil
}

func extractFields(r *http.Request) map[string]string {
	f := make(map[string]string)

	for name, values := range r.URL.Query() {
		f[name] = values[0]
	}
	return f
}

func extractHeaders(r *http.Request) map[string]string {
	h := make(map[string]string)

	for name, values := range r.Header {
		for _, value := range values {
			if len(value) > 0 {
				h[strings.ToUpper(name)] = value
			}
		}
	}
	return h
}

func handleSuccess(w http.ResponseWriter, r *http.Request, rsp *sync.Response, enc encoding.EncodeFunc) error {
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

	_, err = w.Write(p)
	return err
}

func handleError(w http.ResponseWriter, enc encoding.EncodeFunc, err error) {
	// Assert error to type Error in order to leverage the code and payload values that such errors contain.
	if err, ok := err.(*Error); ok {
		p, encErr := enc(err.payload)
		if encErr != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(err.code)
		if _, err := w.Write(p); err != nil {
			log.Errorf("failed to write response: %v", err)
		}
		return
	}
	// Using http.Error helper hijacks the content type header of the response returning plain text payload.
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func prepareResponse(w http.ResponseWriter, ct string) {
	w.Header().Set(encoding.ContentTypeHeader, ct)
}

func extractParams(r *http.Request) map[string]string {
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
