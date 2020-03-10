package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/beatlabs/patron"
	"github.com/beatlabs/patron/examples"
	"github.com/beatlabs/patron/log"
	"github.com/beatlabs/patron/sync"
	patronhttp "github.com/beatlabs/patron/sync/http"
	tracehttp "github.com/beatlabs/patron/trace/http"
	"github.com/beatlabs/patron/trace/kafka"
	synckafka "github.com/beatlabs/patron/trace/kafka/sync"
)

const (
	kafkaTopic  = "patron-topic"
	kafkaBroker = "localhost:9092"
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

	err = os.Setenv("PATRON_HTTP_DEFAULT_PORT", "50001")
	if err != nil {
		fmt.Printf("failed to set default patron port env vars: %v", err)
		os.Exit(1)
	}
}

func main() {
	name := "sixth"
	version := "1.0.0"

	err := patron.SetupLogging(name, version)
	if err != nil {
		fmt.Printf("failed to set up logging: %v", err)
		os.Exit(1)
	}

	httpCmp, err := newHTTPComponent(kafkaBroker, kafkaTopic, "http://localhost:50000/sixth")
	if err != nil {
		log.Fatalf("failed to create processor %v", err)
	}

	// Set up routes
	routes := []patronhttp.Route{
		patronhttp.NewAuthGetRoute("/", httpCmp.sixth, true, nil),
	}

	srv := patron.New(name, version).WithRoutes(routes)

	ctx := context.Background()
	err = srv.Run(ctx)
	if err != nil {
		log.Fatalf("failed to run service %v", err)
	}
}

type httpComponent struct {
	prd   synckafka.Producer
	topic string
}

func newHTTPComponent(kafkaBroker, topic, url string) (*httpComponent, error) {
	sb := synckafka.SyncBuilder{kafka.NewBuilder([]string{kafkaBroker})}
	prd, err := sb.Create()
	if err != nil {
		return nil, err
	}

	return &httpComponent{prd: prd, topic: topic}, nil
}

func (hc *httpComponent) sixth(ctx context.Context, req *sync.Request) (*sync.Response, error) {

	var u examples.User
	err := req.Decode(&u)
	if err != nil {
		return nil, fmt.Errorf("failed to decode message: %w", err)
	}

	googleReq, err := http.NewRequest("GET", "https://www.google.com", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for www.google.com: %w", err)
	}
	cl, err := tracehttp.New(tracehttp.Timeout(5 * time.Second))
	if err != nil {
		return nil, err
	}
	_, err = cl.Do(ctx, googleReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get www.google.com: %w", err)
	}

	kafkaMsg := kafka.NewMessage(hc.topic, &u)

	_, _, err = hc.prd.Send(ctx, kafkaMsg)
	if err != nil {
		return nil, err
	}

	log.FromContext(ctx).Infof("request processed: %s %s", u.GetFirstname(), u.GetLastname())
	return nil, nil
}
