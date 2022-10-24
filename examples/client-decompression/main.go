package main

import (
	"context"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/beatlabs/patron"
	clienthttp "github.com/beatlabs/patron/client/http"
	"github.com/beatlabs/patron/encoding"
	"github.com/beatlabs/patron/log"
)

func init() {
	err := os.Setenv("PATRON_LOG_LEVEL", "debug")
	if err != nil {
		log.Fatalf("failed to set log level env var: %v", err)
	}
	err = os.Setenv("PATRON_JAEGER_SAMPLER_PARAM", "1.0")
	if err != nil {
		log.Fatalf("failed to set sampler env vars: %v", err)
	}
}

func main() {
	name := "client-decompression"
	version := "1.0.0"
	ctx := context.Background()

	service, err := patron.New(name, version)
	if err != nil {
		log.Fatalf("failed to set up service: %v", err)
	}
	err = service.Run(ctx)
	if err != nil {
		log.Fatalf("failed to create and run service %v", err)
	}

	cl, err := clienthttp.New(clienthttp.WithTimeout(1 * time.Second))

	noCompReq, err := http.NewRequest("GET", "http://localhost:50000/hello", nil)
	rsp1, err := cl.Do(noCompReq)
	handle(err)
	bdy1, err := ioutil.ReadAll(rsp1.Body)
	handle(err)

	gzipReq, err := http.NewRequest("GET", "http://localhost:50000/hello", nil)
	if err != nil {
		log.Fatalf("failed to create gzip request: %v", err)
	}
	gzipReq.Header.Add(encoding.AcceptEncodingHeader, "gzip")
	rsp2, err := cl.Do(gzipReq)
	handle(err)
	bdy2, err := ioutil.ReadAll(rsp2.Body)
	handle(err)

	deflateReq, err := http.NewRequest("GET", "http://localhost:50000/hello", nil)
	if err != nil {
		log.Fatalf("failed to create deflate request: %v", err)
	}
	deflateReq.Header.Add(encoding.AcceptEncodingHeader, "deflate")
	rsp3, err := cl.Do(deflateReq)
	handle(err)
	bdy3, err := ioutil.ReadAll(rsp3.Body)
	handle(err)

	log.Infof("Response without compression : %v\n", string(bdy1))
	log.Infof("Response with GZIP compression : %v\n", string(bdy2))
	log.Infof("Response with Deflate compression : %v\n", string(bdy3))
}

func handle(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
