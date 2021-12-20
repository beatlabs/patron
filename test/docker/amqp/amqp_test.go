//go:build integration
// +build integration

package amqp

import (
	"fmt"
	"os"
	"testing"
	"time"

	patronDocker "github.com/beatlabs/patron/test/docker"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/ory/dockertest/v3"
	"github.com/streadway/amqp"
)

const (
	rabbitMQQueue = "rmq-test-queue"
	rabbitMQPort  = "5672/tcp"
)

var (
	runtime *rabbitMQRuntime
	mtr     *mocktracer.MockTracer
)

func TestMain(m *testing.M) {
	var err error
	runtime, err = create(60 * time.Second)
	if err != nil {
		fmt.Printf("could not create AWS runtime: %v\n", err)
		os.Exit(1)
	}

	mtr = mocktracer.New()
	opentracing.SetGlobalTracer(mtr)
	defer mtr.Reset()

	exitCode := m.Run()

	ee := runtime.Teardown()
	if len(ee) > 0 {
		for _, err := range ee {
			fmt.Printf("could not tear down containers: %v\n", err)
		}
	}

	os.Exit(exitCode)
}

type rabbitMQRuntime struct {
	patronDocker.Runtime
}

func create(expiration time.Duration) (*rabbitMQRuntime, error) {
	br, err := patronDocker.NewRuntime(expiration)
	if err != nil {
		return nil, fmt.Errorf("could not create base runtime: %w", err)
	}

	runtime := &rabbitMQRuntime{Runtime: *br}

	runOptions := &dockertest.RunOptions{
		Repository: "bitnami/rabbitmq",
		Tag:        "3.8.12",
	}
	_, err = runtime.RunWithOptions(runOptions)
	if err != nil {
		return nil, fmt.Errorf("could not start mysql: %w", err)
	}

	// wait until the container is ready
	err = runtime.Pool().Retry(func() error {
		conn, err := amqp.Dial(runtime.getEndpoint())
		if err != nil {
			return err
		}

		channel, err := conn.Channel()
		if err != nil {
			return err
		}

		_, err = channel.QueueDeclare(rabbitMQQueue, true, false, false, false, nil)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		for _, err1 := range runtime.Teardown() {
			fmt.Printf("failed to teardown: %v\n", err1)
		}
		return nil, fmt.Errorf("container not ready: %w", err)
	}

	return runtime, nil
}

func (s *rabbitMQRuntime) getEndpoint() string {
	return fmt.Sprintf("amqp://user:bitnami@localhost:%s/", s.Resources()[0].GetPort(rabbitMQPort))
}
