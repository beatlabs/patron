package sql

import (
	"database/sql"
	"fmt"
	"os"
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
	br, err := patronDocker.NewRuntime(expiration)
	if err != nil {
		fmt.Printf("could not create base runtime: %v\n", err)
		return 1
	}
	d := sqlRuntime{Runtime: *br}

	err = d.setup()
	if err != nil {
		fmt.Printf("could not start containers: %v\n", err)
		return 1
	}

	exitVal := m.Run()

	ee := d.Teardown()
	if len(ee) > 0 {
		for _, err = range ee {
			fmt.Printf("could not tear down containers: %v\n", err)
		}
		os.Exit(1)
	}

	return exitVal
}

type sqlRuntime struct {
	patronDocker.Runtime
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
		return fmt.Errorf("could not start zookeeper: %w", err)
	}

	// wait until the container is ready
	return s.Pool().Retry(func() error {
		db, err := sql.Open("mysql", fmt.Sprintf(connectionFormat, dbUsername, dbPassword, dbHost, dbPort, dbSchema))
		if err != nil {
			// container not ready ... return error to try again
			return err
		}
		return db.Ping()
	})
}

// DSN of the set up database.
func DSN() string {
	return fmt.Sprintf(connectionFormat, dbUsername, dbPassword, dbHost, dbPort, dbSchema)
}
