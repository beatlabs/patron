package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/beatlabs/patron"
	clienthttp "github.com/beatlabs/patron/client/http"
	"github.com/beatlabs/patron/encoding"
)

func init() {
	err := os.Setenv("PATRON_LOG_LEVEL", "debug")
	if err != nil {
		fmt.Printf("failed to set log level env var: %v", err)
		os.Exit(1)
	}
	err = os.Setenv("PATRON_JAEGER_SAMPLER_PARAM", "1.0")
	if err != nil {
		fmt.Printf("failed to set sampler env vars: %v", err)
		os.Exit(1)
	}
}

func main() {
	name := "client-decompression"
	version := "1.0.0"
	ctx := context.Background()

	service, err := patron.New(name, version)
	if err != nil {
		fmt.Printf("failed to set up service: %v", err)
		os.Exit(1)
	}
	err = service.Run(ctx)
	if err != nil {
		log.Fatalf("failed to create and run service %v", err)
	}

	cl, err := clienthttp.New(clienthttp.Timeout(1 * time.Second))

	noCompReq, err := http.NewRequest("GET", "http://localhost:50000/hello", nil)
	rsp1, err := cl.Do(ctx, noCompReq)
	handle(err)
	bdy1, err := ioutil.ReadAll(rsp1.Body)
	handle(err)

	gzipReq, err := http.NewRequest("GET", "http://localhost:50000/hello", nil)
	if err != nil {
		log.Fatalf("failed to create gzip request: %v", err)
	}
	gzipReq.Header.Add(encoding.AcceptEncodingHeader, "gzip")
	rsp2, err := cl.Do(ctx, gzipReq)
	handle(err)
	bdy2, err := ioutil.ReadAll(rsp2.Body)
	handle(err)

	deflateReq, err := http.NewRequest("GET", "http://localhost:50000/hello", nil)
	if err != nil {
		log.Fatalf("failed to create deflate request: %v", err)
	}
	deflateReq.Header.Add(encoding.AcceptEncodingHeader, "deflate")
	rsp3, err := cl.Do(ctx, deflateReq)
	handle(err)
	bdy3, err := ioutil.ReadAll(rsp3.Body)
	handle(err)

	fmt.Printf("Response without compression : %v\n", string(bdy1))
	fmt.Printf("Response with GZIP compression : %v\n", string(bdy2))
	fmt.Printf("Response with Deflate compression : %v\n", string(bdy3))
}

func handle(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
