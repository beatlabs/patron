package redis

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"time"

	patronDocker "github.com/beatlabs/patron/test/docker"
	"github.com/go-redis/redis/v8"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

type redisRuntime struct {
	patronDocker.Runtime
}

// Create initializes a Sql docker runtime.
func create(expiration time.Duration) (*redisRuntime, error) {
	br, err := patronDocker.NewRuntime(expiration)
	if err != nil {
		return nil, fmt.Errorf("could not create base runtime: %w", err)
	}

	runtime := &redisRuntime{Runtime: *br}

	runOptions := &dockertest.RunOptions{
		Repository: "bitnami/redis",
		Tag:        "6.0.9",
		PortBindings: map[docker.Port][]docker.PortBinding{
			"6379/tcp": {{HostIP: "", HostPort: ""}},
		},
		Env: []string{
			"ALLOW_EMPTY_PASSWORD=yes",
			"REDIS_DISABLE_COMMANDS=FLUSHDB,FLUSHALL",
		},
	}
	_, err = runtime.RunWithOptions(runOptions)
	if err != nil {
		return nil, fmt.Errorf("could not start mysql: %w", err)
	}

	// wait until the container is ready
	err = runtime.Pool().Retry(func() error {
		dsn, err := runtime.DSN()
		if err != nil {
			return err
		}

		client := redis.NewClient(&redis.Options{
			Addr:     dsn,
			Password: "", // no password set
			DB:       0,  // use default DB
		})

		ping := client.Ping(context.Background())
		return ping.Err()
	})
	if err != nil {
		for _, err1 := range runtime.Teardown() {
			fmt.Printf("failed to teardown: %v\n", err1)
		}
		return nil, fmt.Errorf("container not ready: %w", err)
	}

	return runtime, nil
}

// DSN of the runtime.
func (s *redisRuntime) DSN() (string, error) {
	// if run with docker-machine the hostname needs to be set
	u, err := url.Parse(s.Pool().Client.Endpoint())
	if err != nil {
		return "", fmt.Errorf("could not parse endpoint: %s", s.Pool().Client.Endpoint())
	}

	return net.JoinHostPort(u.Hostname(), s.Resources()[0].GetPort("6379/tcp")), nil
}
