package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron"
	clienthttp "github.com/beatlabs/patron/client/http"
	v2 "github.com/beatlabs/patron/client/kafka/v2"
	patronhttp "github.com/beatlabs/patron/component/http"
	"github.com/beatlabs/patron/component/http/auth/apikey"
	"github.com/beatlabs/patron/encoding/json"
	"github.com/beatlabs/patron/examples"
	"github.com/beatlabs/patron/log"
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
	name := "http-sec"
	version := "1.0.0"

	service, err := patron.New(name, version, patron.LogFields(map[string]interface{}{"env": "staging"}))
	if err != nil {
		fmt.Printf("failed to set up service: %v", err)
		os.Exit(1)
	}

	httpCmp, err := newHTTPComponent(kafkaBroker, kafkaTopic, "http://localhost:50000/kafka")
	if err != nil {
		log.Fatalf("failed to create processor %v", err)
	}

	auth, err := apikey.New(&apiKeyValidator{validKey: "123456"})
	if err != nil {
		log.Fatalf("failed to create authenticator %v", err)
	}

	routesBuilder := patronhttp.NewRoutesBuilder().
		Append(patronhttp.NewGetRouteBuilder("/", httpCmp.kafkaHandler).WithTrace().WithAuth(auth))

	ctx := context.Background()
	err = service.WithRoutesBuilder(routesBuilder).Run(ctx)
	if err != nil {
		log.Fatalf("failed to create and run service %v", err)
	}
}

type httpComponent struct {
	prd   *v2.AsyncProducer
	topic string
}

func newHTTPComponent(kafkaBroker, topic, _ string) (*httpComponent, error) {
	prd, chErr, err := v2.New([]string{kafkaBroker}).CreateAsync()
	if err != nil {
		return nil, err
	}
	go func() {
		for {
			err := <-chErr
			log.Errorf("error producing Kafka message: %v", err)
		}
	}()
	return &httpComponent{prd: prd, topic: topic}, nil
}

func (hc *httpComponent) kafkaHandler(ctx context.Context, req *patronhttp.Request) (*patronhttp.Response, error) {
	var u examples.User
	err := req.Decode(&u)
	if err != nil {
		return nil, fmt.Errorf("failed to decode message: %w", err)
	}

	googleReq, err := http.NewRequest("GET", "https://www.google.com", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for www.google.com: %w", err)
	}
	cl, err := clienthttp.New(clienthttp.Timeout(5 * time.Second))
	if err != nil {
		return nil, err
	}
	_, err = cl.Do(ctx, googleReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get www.google.com: %w", err)
	}

	b, err := json.Encode(u)
	if err != nil {
		return nil, fmt.Errorf("failed to encode message: %w", err)
	}

	kafkaMsg := &sarama.ProducerMessage{
		Topic: hc.topic,
		Value: sarama.ByteEncoder(b),
	}

	err = hc.prd.Send(ctx, kafkaMsg)
	if err != nil {
		return nil, err
	}

	log.FromContext(ctx).Infof("request processed: %s %s", u.GetFirstname(), u.GetLastname())
	return nil, nil
}

type apiKeyValidator struct {
	validKey string
}

func (av apiKeyValidator) Validate(key string) (bool, error) {
	if key == av.validKey {
		return true, nil
	}
	return false, nil
}
