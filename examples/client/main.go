package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	smithyendpoints "github.com/aws/smithy-go/endpoints"
	patronamqp "github.com/beatlabs/patron/client/amqp"
	patrongrpc "github.com/beatlabs/patron/client/grpc"
	patronhttp "github.com/beatlabs/patron/client/http"
	patronkafka "github.com/beatlabs/patron/client/kafka"
	patronsqs "github.com/beatlabs/patron/client/sqs"
	"github.com/beatlabs/patron/component/kafka"
	"github.com/beatlabs/patron/encoding/protobuf"
	"github.com/beatlabs/patron/examples"
	"github.com/beatlabs/patron/observability/trace"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/sdk/resource"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type process func(context.Context) error

type mode string

const (
	modeAll   mode = "all"
	modeHTTP  mode = "http"
	modeGRPC  mode = "grpc"
	modeKafka mode = "kafka"
	modeAMQP  mode = "amqp"
	modeSQS   mode = "sqs"
)

func main() {
	var modes string
	flag.StringVar(&modes, "modes", string(modeAll), `modes determines what clients to run. 
	Multiple modes are allowed in a comma separated fashion. 
	Valid values are: all, http, grpc, kafka, amqp, sqs. Default value is all.`)

	flag.Parse()

	prs, err := processModes(modes)
	if err != nil {
		fmt.Printf("failed to parse flags: %v\n", err)
		flag.Usage()
		os.Exit(1)
	}

	tp, err := trace.SetupGRPC(context.Background(), "example-client", resource.Default())
	handleError(err)

	defer func() {
		handleError(tp.ForceFlush(context.Background()))
		handleError(tp.Shutdown(context.Background()))
	}()

	ctx, cnl := context.WithTimeout(context.Background(), 50000*time.Second)
	defer cnl()

	ctx, sp := trace.StartSpan(ctx, "example-client")
	defer sp.End()

	for _, process := range prs {
		err = process(ctx)
		handleError(err)
	}
}

func processModes(modes string) ([]process, error) {
	if modes == "" {
		return nil, errors.New("modes was empty")
	}

	mds := strings.Split(modes, ",")
	if len(mds) == 0 {
		return nil, errors.New("modes was empty")
	}

	var prs []process

	for _, mode := range mds {
		switch mode {
		case string(modeAll):
			return []process{sendHTTPRequest, sendGRPCRequest, sendKafkaMessage, sendAMQPMessage, sendSQSMessage}, nil
		case string(modeHTTP):
			prs = append(prs, sendHTTPRequest)
		case string(modeGRPC):
			prs = append(prs, sendGRPCRequest)
		case string(modeKafka):
			prs = append(prs, sendKafkaMessage)
		case string(modeAMQP):
			prs = append(prs, sendAMQPMessage)
		case string(modeSQS):
			prs = append(prs, sendSQSMessage)
		default:
			return nil, fmt.Errorf("unsupported mode %s", mode)
		}
	}

	return prs, nil
}

func sendHTTPRequest(ctx context.Context) error {
	httpClient, err := patronhttp.New()
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, examples.HTTPURL, nil)
	if err != nil {
		return err
	}

	rsp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer rsp.Body.Close()

	fmt.Printf("HTTP response received: %d\n", rsp.StatusCode)
	return nil
}

func sendGRPCRequest(ctx context.Context) error {
	cc, err := patrongrpc.NewClient(examples.GRPCTarget, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}

	client := examples.NewGreeterClient(cc)

	_, err = client.SayHello(ctx, &examples.HelloRequest{Firstname: "John", Lastname: "Doe"})
	if err != nil {
		return err
	}

	fmt.Println("gRPC reply received")
	return nil
}

func sendKafkaMessage(ctx context.Context) error {
	cfg, err := kafka.DefaultConsumerSaramaConfig("patron-producer", true)
	if err != nil {
		return err
	}

	producer, err := patronkafka.New([]string{examples.KafkaBroker}, cfg).Create()
	if err != nil {
		return err
	}
	defer func() {
		err := producer.Close()
		if err != nil {
			fmt.Println(err)
		}
	}()

	msg := &sarama.ProducerMessage{
		Topic: examples.KafkaTopic,
		Value: sarama.StringEncoder("example message"),
	}

	_, _, err = producer.Send(ctx, msg)
	if err != nil {
		return err
	}

	fmt.Println("kafka message sent")
	return nil
}

func sendAMQPMessage(ctx context.Context) error {
	publisher, err := patronamqp.New(examples.AMQPURL)
	if err != nil {
		return err
	}

	amqpMsg := amqp.Publishing{
		ContentType: protobuf.Type,
		Body:        []byte("example message"),
	}

	err = publisher.Publish(ctx, examples.AMQPExchangeName, "", false, false, amqpMsg)
	if err != nil {
		return err
	}

	fmt.Println("AMQP message sent")
	return nil
}

type customResolver struct{}

func (cr *customResolver) ResolveEndpoint(_ context.Context, _ sqs.EndpointParameters) (smithyendpoints.Endpoint, error) {
	uri, err := url.Parse(examples.AWSSQSEndpoint)
	if err != nil {
		return smithyendpoints.Endpoint{}, err
	}
	return smithyendpoints.Endpoint{
		URI: *uri,
	}, nil
}

func sendSQSMessage(ctx context.Context) error {
	cfg, err := examples.CreateSQSConfig(ctx)
	if err != nil {
		return err
	}

	client := patronsqs.NewFromConfig(cfg, sqs.WithEndpointResolverV2(&customResolver{}))

	out, err := client.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{QueueName: aws.String(examples.AWSSQSQueue)})
	if err != nil {
		return err
	}

	_, err = client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    out.QueueUrl,
		MessageBody: aws.String("example message"),
	})
	if err != nil {
		return err
	}

	fmt.Println("AWS SQS message sent")
	return nil
}

func handleError(err error) {
	if err == nil {
		return
	}
	fmt.Println(err)
	os.Exit(1)
}
