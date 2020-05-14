package test

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

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

// RunWithKafka sets up and tears down Kafka and runs the tests.
func RunWithKafka(m *testing.M, expiration time.Duration, topics ...string) int {
	br, err := patrondocker.NewRuntime(expiration)
	if err != nil {
		log.Errorf("could not create base runtime: %v", err)
		os.Exit(1)
	}
	d := kafkaRuntime{topics: topics, Runtime: *br}

	err = d.setup()
	if err != nil {
		log.Errorf("could not start containers: %v", err)
		os.Exit(1)
	}

	exitVal := m.Run()

	ee := d.Teardown()
	if len(ee) > 0 {
		for _, err = range ee {
			log.Errorf("could not tear down containers %v", err)
		}
		os.Exit(1)
	}

	return exitVal
}

type kafkaRuntime struct {
	patrondocker.Runtime
	topics []string
}

func (k *kafkaRuntime) setup() error {

	var err error

	zookeeper, err := k.Runtime.Pool().RunWithOptions(&dockertest.RunOptions{Repository: "wurstmeister/zookeeper",
		PortBindings: map[docker.Port][]docker.PortBinding{
			docker.Port(fmt.Sprintf("%s/tcp", zookeeperPort)): {{HostIP: "", HostPort: zookeeperPort}},
			// port 22 is too generic to be used for the test
			"29/tcp":   {{HostIP: "", HostPort: "22"}},
			"2888/tcp": {{HostIP: "", HostPort: "2888"}},
			"3888/tcp": {{HostIP: "", HostPort: "3888"}},
		},
	})
	if err != nil {
		return fmt.Errorf("could not start zookeeper: %w", err)
	}

	err = zookeeper.Expire(uint(k.Expiration().Seconds()))
	if err != nil {
		return errors.New("could not set expiration on zookeeper")
	}

	k.AppendResources(zookeeper)

	ip := zookeeper.Container.NetworkSettings.Networks["bridge"].IPAddress

	kafkaTCPPort := fmt.Sprintf("%s/tcp", kafkaPort)

	runOptions := &dockertest.RunOptions{
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

	kafka, err := k.Pool().RunWithOptions(runOptions)
	if err != nil {
		return fmt.Errorf("could not start kafka: %w", err)
	}

	err = kafka.Expire(uint(k.Expiration().Seconds()))
	if err != nil {
		return errors.New("could not set expiration on kafka")
	}

	k.AppendResources(kafka)

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

	brokers := []string{fmt.Sprintf("%s:%s", kafkaHost, kafkaPort)}

	return sarama.NewSyncProducer(brokers, config)
}

// NewConsumer creates a new consumer.
func NewConsumer() (sarama.Consumer, error) {
	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true

	brokers := []string{fmt.Sprintf("%s:%s", kafkaHost, kafkaPort)}

	return sarama.NewConsumer(brokers, config)
}
