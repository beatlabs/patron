# Encoding

The encoding package contains the following functions definitions that are used throughout the framework.

```go
// DecodeFunc function definition of a JSON decoding function.
type DecodeFunc func(data io.Reader, v interface{}) error

// DecodeRawFunc function definition of a JSON decoding function from a byte slice.
type DecodeRawFunc func(data []byte, v interface{}) error

// EncodeFunc function definition of a JSON encoding function.
type EncodeFunc func(v interface{}) ([]byte, error) 
```

The following sub-packages are provided:

- `json` which contains implementations of the encoding functions
- `protobuf` which contains implementations of the encoding functions