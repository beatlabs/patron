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
	patronkafka "github.com/beatlabs/patron/client/kafka/v2"
	"github.com/beatlabs/patron/component/http/auth/apikey"
	v2 "github.com/beatlabs/patron/component/http/v2"
	"github.com/beatlabs/patron/component/http/v2/router/httprouter"
	"github.com/beatlabs/patron/component/kafka"
	"github.com/beatlabs/patron/encoding/json"
	"github.com/beatlabs/patron/encoding/protobuf"
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

	asyncComp, err := newAsyncKafkaProducer(kafkaBroker, kafkaTopic, true)
	if err != nil {
		log.Fatalf("failed to create processor %v", err)
	}

	auth, err := apikey.New(&apiKeyValidator{validKey: "123456"})
	if err != nil {
		log.Fatalf("failed to create authenticator %v", err)
	}

	var routes v2.Routes
	routes.Append(v2.NewGetRoute("/", asyncComp.forwardToKafkaHandler, v2.Auth(auth)))
	rr, err := routes.Result()
	if err != nil {
		log.Fatalf("failed to create routes: %v", err)
	}

	router, err := httprouter.New(httprouter.Routes(rr...))
	if err != nil {
		log.Fatalf("failed to create http router: %v", err)
	}

	ctx := context.Background()
	err = service.WithRouter(router).Run(ctx)
	if err != nil {
		log.Fatalf("failed to create and run service %v", err)
	}
}

type kafkaProducer struct {
	prd   *patronkafka.AsyncProducer
	topic string
}

// newAsyncKafkaProducer creates a new asynchronous kafka producer client
func newAsyncKafkaProducer(kafkaBroker, topic string, readCommitted bool) (*kafkaProducer, error) {
	saramaCfg, err := kafka.DefaultConsumerSaramaConfig("http-sec-consumer", readCommitted)
	if err != nil {
		return nil, err
	}

	prd, chErr, err := patronkafka.New([]string{kafkaBroker}, saramaCfg).CreateAsync()
	if err != nil {
		return nil, err
	}
	go func() {
		for {
			err := <-chErr
			log.Errorf("error producing Kafka message: %v", err)
		}
	}()
	return &kafkaProducer{prd: prd, topic: topic}, nil
}

// forwardToKafkaHandler is a http handler that decodes the input request and
// publishes the decoded content as a message into a kafka topic (also does an HTTP GET request to google.com)
func (hc *kafkaProducer) forwardToKafkaHandler(rw http.ResponseWriter, r *http.Request) {
	var u examples.User

	err := protobuf.Decode(r.Body, &u)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		_, _ = rw.Write([]byte(fmt.Sprintf("failed to decode request: %v", err)))
		return
	}

	googleReq, err := http.NewRequest("GET", "https://www.google.com", nil)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		_, _ = rw.Write([]byte(fmt.Sprintf("failed to create request for www.google.com: %v", err)))
		return
	}
	cl, err := clienthttp.New(clienthttp.Timeout(5 * time.Second))
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, err = cl.Do(googleReq)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		_, _ = rw.Write([]byte(fmt.Sprintf("failed to get www.google.com: %v", err)))
		return
	}

	b, err := json.Encode(&u)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		_, _ = rw.Write([]byte(fmt.Sprintf("failed to encode message: %v", err)))
		return
	}

	kafkaMsg := &sarama.ProducerMessage{
		Topic: hc.topic,
		Value: sarama.ByteEncoder(b),
	}

	err = hc.prd.Send(r.Context(), kafkaMsg)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.FromContext(r.Context()).Infof("request processed: %s %s", u.GetFirstname(), u.GetLastname())
	rw.WriteHeader(http.StatusCreated)
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
