package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/Shopify/sarama"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	patronamqp "github.com/beatlabs/patron/client/amqp/v2"
	patrongrpc "github.com/beatlabs/patron/client/grpc"
	patronhttp "github.com/beatlabs/patron/client/http"
	patronkafka "github.com/beatlabs/patron/client/kafka/v2"
	patronsqs "github.com/beatlabs/patron/client/sqs"
	"github.com/beatlabs/patron/component/kafka"
	"github.com/beatlabs/patron/encoding/protobuf"
	"github.com/beatlabs/patron/examples"
	"github.com/streadway/amqp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	ctx, cnl := context.WithCancel(context.Background())
	defer cnl()

	err := sendHTTPRequest(ctx)
	handleError(err)

	err = sendGRPCRequest(ctx)
	handleError(err)

	err = sendKafkaMessage(ctx)
	handleError(err)

	err = sendAMQPMessage(ctx)
	handleError(err)

	err = sendSQSMessage(ctx)
	handleError(err)
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

	fmt.Printf("HTTP response received: %d\n", rsp.StatusCode)
	return nil
}

func sendGRPCRequest(ctx context.Context) error {
	cc, err := patrongrpc.DialContext(ctx, examples.GRPCTarget, grpc.WithTransportCredentials(insecure.NewCredentials()))
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
	defer producer.Close()

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

func sendSQSMessage(ctx context.Context) error {
	api, err := examples.CreateSQSAPI()
	if err != nil {
		return err
	}

	out, err := api.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{QueueName: aws.String(examples.AWSSQSQueue)})
	if err != nil {
		return err
	}

	publisher, err := patronsqs.New(api)
	if err != nil {
		return err
	}

	_, err = publisher.Publish(ctx, &sqs.SendMessageInput{
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
