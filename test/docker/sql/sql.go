package sql

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	patronDocker "github.com/beatlabs/patron/test/docker"
	// Integration test.
	_ "github.com/go-sql-driver/mysql"
	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
)

const (
	dbHost           = "localhost"
	dbSchema         = "patrondb"
	dbPort           = "3309"
	dbRouterPort     = "33069"
	dbPassword       = "test123"
	dbRootPassword   = "test123"
	dbUsername       = "patron"
	connectionFormat = "%s:%s@(%s:%s)/%s?parseTime=true"
)

// RunWithSQL sets up and tears down Mysql and runs the tests.
func RunWithSQL(m *testing.M, expiration time.Duration) int {
	d, err := create(expiration)
	if err != nil {
		fmt.Printf("could not create mysql runtime: %v\n", err)
		return 1
	}

	err = d.setup()
	if err != nil {
		fmt.Printf("could not setup mysql runtime: %v\n", err)
		return 1
	}
	defer func() {
		ee := d.Teardown()
		if len(ee) > 0 {
			for _, err = range ee {
				fmt.Printf("could not tear down containers: %v\n", err)
			}
		}
	}()

	return m.Run()
}

type sqlRuntime struct {
	patronDocker.Runtime
}

func create(expiration time.Duration) (*sqlRuntime, error) {
	br, err := patronDocker.NewRuntime(expiration)
	if err != nil {
		return nil, fmt.Errorf("could not create base runtime: %w", err)
	}
	return &sqlRuntime{Runtime: *br}, nil
}

func (s *sqlRuntime) setup() error {

	runOptions := &dockertest.RunOptions{Repository: "mysql",
		Tag: "5.7.25",
		PortBindings: map[docker.Port][]docker.PortBinding{
			"3306/tcp":  {{HostIP: "", HostPort: dbPort}},
			"33060/tcp": {{HostIP: "", HostPort: dbRouterPort}},
		},
		ExposedPorts: []string{"3306/tcp", "33060/tcp"},
		Env: []string{
			fmt.Sprintf("MYSQL_ROOT_PASSWORD=%s", dbRootPassword),
			fmt.Sprintf("MYSQL_USER=%s", dbUsername),
			fmt.Sprintf("MYSQL_PASSWORD=%s", dbPassword),
			fmt.Sprintf("MYSQL_DATABASE=%s", dbSchema),
			"TIMEZONE=UTC",
		}}

	_, err := s.RunWithOptions(runOptions)
	if err != nil {
		return fmt.Errorf("could not start mysql: %w", err)
	}

	// wait until the container is ready
	err = s.Pool().Retry(func() error {
		db, err := sql.Open("mysql", DSN())
		if err != nil {
			// container not ready ... return error to try again
			return err
		}
		return db.Ping()
	})
	if err != nil {
		return fmt.Errorf("container not ready: %w", err)
	}

	return nil
}

// DSN of the set up database.
func DSN() string {
	return fmt.Sprintf(connectionFormat, dbUsername, dbPassword, dbHost, dbPort, dbSchema)
}
