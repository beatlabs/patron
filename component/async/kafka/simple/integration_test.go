package simple

import (
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/log"
	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
	"github.com/stretchr/testify/assert"
)

const (
	kafkaHost     = "localhost"
	kafkaPort     = "9092"
	zookeeperPort = "2181"
)

func TestMain(m *testing.M) {

	d := dockerRuntime{}

	err := d.setup()
	if err != nil {
		log.Errorf("could not start containers %v", err)
		os.Exit(1)
	}

	exitVal := m.Run()

	ee := d.teardown()
	if len(ee) > 0 {
		for _, err = range ee {
			log.Errorf("could not tear down containers %v", err)
		}
		os.Exit(1)
	}

	os.Exit(exitVal)
}

func TestInt(t *testing.T) {
	assert.Equal(t, 1, 1)
}

type dockerRuntime struct {
	topics    []string
	pool      *dockertest.Pool
	kafka     *dockertest.Resource
	zookeeper *dockertest.Resource
}

func (d *dockerRuntime) setup() error {

	d.topics = []string{"simple"}
	expiration := uint(120)
	var err error

	d.pool, err = dockertest.NewPool("")
	if err != nil {
		return err
	}
	d.pool.MaxWait = time.Duration(expiration) * time.Second

	d.zookeeper, err = d.pool.RunWithOptions(&dockertest.RunOptions{Repository: "wurstmeister/zookeeper",
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

	d.zookeeper.Expire(expiration)

	ip := d.zookeeper.Container.NetworkSettings.Networks["bridge"].IPAddress

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
			fmt.Sprintf("KAFKA_CREATE_TOPICS=%s", d.topics),
			fmt.Sprintf("KAFKA_ZOOKEEPER_CONNECT=%s:%s", ip, zookeeperPort),
		}}

	d.kafka, err = d.pool.RunWithOptions(runOptions)
	if err != nil {
		return fmt.Errorf("could not start kafka: %w", err)
	}

	d.kafka.Expire(expiration)

	return d.pool.Retry(func() error {
		consumer, err := NewConsumer()
		if err != nil {
			return err
		}
		topics, err := consumer.Topics()
		if err != nil {
			log.Infof("err or during topic retrieval = %v", err)
			return err
		}
		if reflect.DeepEqual(topics, d.topics) {
			return nil
		}

		return fmt.Errorf("expected topics do not exist: %v != %v", d.topics, topics)
	})
}

func (d *dockerRuntime) teardown() []error {
	ee := make([]error, 0)
	err := d.pool.Purge(d.kafka)
	if err != nil {
		ee = append(ee, err)
	}
	err = d.pool.Purge(d.zookeeper)
	if err != nil {
		ee = append(ee, err)
	}
	return ee
}

// NewConsumer creates a new consumer for the current kafka runtime
func NewConsumer() (sarama.Consumer, error) {
	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true

	brokers := []string{fmt.Sprintf("%s:%s", kafkaHost, kafkaPort)}

	return sarama.NewConsumer(brokers, config)
}
