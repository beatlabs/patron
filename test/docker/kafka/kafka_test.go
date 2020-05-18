package kafka

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/beatlabs/patron/component/async"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/log"
	patrondocker "github.com/beatlabs/patron/test/docker"
	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
)

const (
	kafkaHost     = "localhost"
	kafkaPort     = "9092"
	zookeeperPort = "2181"
)

func TestMain(m *testing.M) {
	topics := []string{
		getTopic(simpleTopic1),
		getTopic(simpleTopic2),
		getTopic(groupTopic1),
		getTopic(groupTopic2),
	}
	k, err := create(60*time.Second, topics...)
	if err != nil {
		fmt.Printf("could not create kafka runtime: %v\n", err)
		os.Exit(1)
	}
	err = k.setup()
	if err != nil {
		fmt.Printf("could not start containers: %v\n", err)
		os.Exit(1)
	}

	exitCode := m.Run()

	ee := k.Teardown()
	if len(ee) > 0 {
		for _, err := range ee {
			fmt.Printf("could not tear down containers: %v\n", err)
		}
	}

	os.Exit(exitCode)
}

type kafkaRuntime struct {
	patrondocker.Runtime
	topics []string
}

func create(expiration time.Duration, topics ...string) (*kafkaRuntime, error) {
	br, err := patrondocker.NewRuntime(expiration)
	if err != nil {
		return nil, fmt.Errorf("could not create base runtime: %w", err)
	}
	return &kafkaRuntime{topics: topics, Runtime: *br}, nil
}

func (k *kafkaRuntime) setup() error {

	var err error

	runOptions := &dockertest.RunOptions{Repository: "wurstmeister/zookeeper",
		PortBindings: map[docker.Port][]docker.PortBinding{
			docker.Port(fmt.Sprintf("%s/tcp", zookeeperPort)): {{HostIP: "", HostPort: zookeeperPort}},
			// port 22 is too generic to be used for the test
			"29/tcp":   {{HostIP: "", HostPort: "22"}},
			"2888/tcp": {{HostIP: "", HostPort: "2888"}},
			"3888/tcp": {{HostIP: "", HostPort: "3888"}},
		},
	}
	zookeeper, err := k.RunWithOptions(runOptions)
	if err != nil {
		return fmt.Errorf("could not start zookeeper: %w", err)
	}

	ip := zookeeper.Container.NetworkSettings.Networks["bridge"].IPAddress

	kafkaTCPPort := fmt.Sprintf("%s/tcp", kafkaPort)

	runOptions = &dockertest.RunOptions{
		Repository: "wurstmeister/kafka",
		Tag:        "2.12-2.5.0",
		PortBindings: map[docker.Port][]docker.PortBinding{
			docker.Port(kafkaTCPPort): {{HostIP: "", HostPort: kafkaPort}},
		},
		ExposedPorts: []string{kafkaTCPPort},
		Mounts:       []string{"/tmp/local-kafka:/etc/kafka"},
		Env: []string{
			"KAFKA_ADVERTISED_HOST_NAME=127.0.0.1",
			fmt.Sprintf("KAFKA_CREATE_TOPICS=%s", strings.Join(k.topics, ",")),
			fmt.Sprintf("KAFKA_ZOOKEEPER_CONNECT=%s:%s", ip, zookeeperPort),
		}}

	_, err = k.RunWithOptions(runOptions)
	if err != nil {
		return fmt.Errorf("could not start kafka: %w", err)
	}

	return k.Pool().Retry(func() error {
		consumer, err := NewConsumer()
		if err != nil {
			return err
		}
		topics, err := consumer.Topics()
		if err != nil {
			log.Infof("err or during topic retrieval = %v", err)
			return err
		}

		return validateTopics(topics, k.topics)
	})
}

func validateTopics(clusterTopics, wantTopics []string) error {
	var found int
	for _, wantTopic := range wantTopics {
		topic := strings.Split(wantTopic, ":")
		for _, clusterTopic := range clusterTopics {
			if topic[0] == clusterTopic {
				found++
			}
		}
	}

	if found != len(wantTopics) {
		return fmt.Errorf("failed to find topics %v in cluster topics %v", wantTopics, clusterTopics)
	}

	return nil
}

// NewProducer creates a new sync producer.
func NewProducer() (sarama.SyncProducer, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.Return.Errors = true

	brokers := []string{fmt.Sprintf("%s:%s", kafkaHost, kafkaPort)}

	return sarama.NewSyncProducer(brokers, config)
}

// NewConsumer creates a new consumer.
func NewConsumer() (sarama.Consumer, error) {
	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true

	return sarama.NewConsumer(Brokers(), config)
}

// Brokers returns a list of brokers.
func Brokers() []string {
	return []string{fmt.Sprintf("%s:%s", kafkaHost, kafkaPort)}
}

func getTopic(name string) string {
	return fmt.Sprintf("%s:1:1", name)
}

func consumeMessages(consumer async.Consumer, expectedMessageCount int) ([]string, error) {

	ctx, cnl := context.WithCancel(context.Background())
	defer cnl()

	ch, chErr, err := consumer.Consume(ctx)
	if err != nil {
		return nil, err
	}

	received := make([]string, 0, expectedMessageCount)

	for {
		select {
		case msg := <-ch:
			received = append(received, string(msg.Payload()))
			expectedMessageCount--
			if expectedMessageCount == 0 {
				return received, nil
			}
		case err := <-chErr:
			return nil, err
		}
	}
}

func sendMessages(messages ...*sarama.ProducerMessage) error {
	prod, err := NewProducer()
	if err != nil {
		return err
	}
	for _, message := range messages {
		_, _, err = prod.SendMessage(message)
		if err != nil {
			return err
		}
	}

	return nil
}

func getProducerMessage(topic, message string) *sarama.ProducerMessage {
	return &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.StringEncoder(message),
	}
}
