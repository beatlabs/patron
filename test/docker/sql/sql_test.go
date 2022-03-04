//go:build integration
// +build integration

package sql

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/ory/dockertest/v3/docker"

	patronDocker "github.com/beatlabs/patron/test/docker"
	// Integration test.
	_ "github.com/go-sql-driver/mysql"
	"github.com/ory/dockertest/v3"
)

const (
	dbHost           = "localhost"
	dbSchema         = "patrondb"
	dbPassword       = "test123"
	dbRootPassword   = "test123"
	dbUsername       = "patron"
	connectionFormat = "%s:%s@(%s:%s)/%s?parseTime=true"
)

// SQL defines a docker SQL runtime.
type sqlRuntime struct {
	patronDocker.Runtime
}

// Create initializes a Sql docker runtime.
func create(expiration time.Duration) (*sqlRuntime, error) {
	br, err := patronDocker.NewRuntime(expiration)
	if err != nil {
		return nil, fmt.Errorf("could not create base runtime: %w", err)
	}

	runtime := &sqlRuntime{Runtime: *br}

	runOptions := &dockertest.RunOptions{
		Repository: "mysql",
		Tag:        "5.7.25",
		PortBindings: map[docker.Port][]docker.PortBinding{
			"3306/tcp":  {{HostIP: "", HostPort: ""}},
			"33060/tcp": {{HostIP: "", HostPort: ""}},
		},

		// ExposedPorts: []string{"3306/tcp", "33060/tcp"},
		Env: []string{
			fmt.Sprintf("MYSQL_ROOT_PASSWORD=%s", dbRootPassword),
			fmt.Sprintf("MYSQL_USER=%s", dbUsername),
			fmt.Sprintf("MYSQL_PASSWORD=%s", dbPassword),
			fmt.Sprintf("MYSQL_DATABASE=%s", dbSchema),
			"TIMEZONE=UTC",
		},
	}

	_, err = runtime.RunWithOptions(runOptions)
	if err != nil {
		return nil, fmt.Errorf("could not start mysql: %w", err)
	}

	// wait until the container is ready
	err = runtime.Pool().Retry(func() error {
		db, err := sql.Open("mysql", runtime.DSN())
		if err != nil {
			// container not ready ... return error to try again
			return err
		}
		return db.Ping()
	})
	if err != nil {
		for _, err1 := range runtime.Teardown() {
			fmt.Printf("failed to teardown: %v\n", err1)
		}
		return nil, fmt.Errorf("container not ready: %w", err)
	}

	return runtime, nil
}

// Port returns a port where the container service can be reached.
func (s *sqlRuntime) Port() string {
	return s.Resources()[0].GetPort("3306/tcp")
}

// DSN of the runtime.
func (s *sqlRuntime) DSN() string {
	return fmt.Sprintf(connectionFormat, dbUsername, dbPassword, dbHost, s.Port(), dbSchema)
}
