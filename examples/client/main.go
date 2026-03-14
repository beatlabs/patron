package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	patronamqp "github.com/beatlabs/patron/client/amqp"
	patrongrpc "github.com/beatlabs/patron/client/grpc"
	patronhttp "github.com/beatlabs/patron/client/http"
	patronkafka "github.com/beatlabs/patron/client/kafka"
	patronsqs "github.com/beatlabs/patron/client/sqs"
	"github.com/beatlabs/patron/encoding/protobuf"
	"github.com/beatlabs/patron/examples"
	"github.com/beatlabs/patron/observability/trace"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
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

	handleError(waitForService(ctx))

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

func waitForService(ctx context.Context) error {
	aliveURL := examples.HTTPURL + "/alive"
	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, aliveURL, nil)
		if err != nil {
			return err
		}

		rsp, err := http.DefaultClient.Do(req)
		if err == nil {
			rsp.Body.Close()
			if rsp.StatusCode == http.StatusOK {
				return nil
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
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

	_, err = client.SayHello(ctx, &examples.HelloRequest{FirstName: "John", LastName: "Doe"})
	if err != nil {
		return err
	}

	fmt.Println("gRPC reply received")
	return nil
}

func sendKafkaMessage(ctx context.Context) error {
	if err := ensureTopicExists(ctx, examples.KafkaBroker, examples.KafkaTopic); err != nil {
		return fmt.Errorf("failed to ensure topic exists: %w", err)
	}

	producer, err := patronkafka.New([]string{examples.KafkaBroker}, kgo.RequiredAcks(kgo.AllISRAcks()))
	if err != nil {
		return err
	}
	defer producer.Close()

	msg := &kgo.Record{
		Topic: examples.KafkaTopic,
		Value: []byte("example message"),
	}

	_, err = producer.Send(ctx, msg)
	if err != nil {
		return err
	}

	fmt.Println("kafka message sent")
	return nil
}

func ensureTopicExists(ctx context.Context, broker, topic string) error {
	cl, err := kgo.NewClient(kgo.SeedBrokers(broker))
	if err != nil {
		return err
	}
	defer cl.Close()

	adm := kadm.NewClient(cl)

	resp, err := adm.CreateTopics(ctx, 1, 1, nil, topic)
	if err != nil {
		return err
	}

	for _, r := range resp {
		if r.Err != nil && !errors.Is(r.Err, kerr.TopicAlreadyExists) {
			return fmt.Errorf("failed to create topic %s: %w", r.Topic, r.Err)
		}
	}

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

func sendSQSMessage(ctx context.Context) error {
	cfg, err := examples.CreateSQSConfig(ctx)
	if err != nil {
		return err
	}

	client := patronsqs.NewFromConfig(cfg, sqs.WithEndpointResolverV2(&examples.SQSCustomResolver{}))

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
