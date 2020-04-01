// +build integration

package sql

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/beatlabs/patron/log"

	_ "github.com/go-sql-driver/mysql"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
	"github.com/stretchr/testify/assert"
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

func TestMain(m *testing.M) {

	d := dockerRuntime{}

	err := d.startUpContainerSync()
	if err != nil {
		log.Errorf("could not start containers %v", err)
		os.Exit(1)
	}

	exitVal := m.Run()

	err = d.tearDownContainerSync()
	if err != nil {
		log.Errorf("could not tear down containers %v", err)
		os.Exit(1)
	}

	os.Exit(exitVal)
}

func TestOpen(t *testing.T) {
	type args struct {
		driverName string
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success":            {args: args{driverName: "mysql"}},
		"failure with wrong": {args: args{driverName: "XXX"}, expectedErr: "sql: unknown driver \"XXX\" (forgotten import?)"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := Open(tt.args.driverName, fmt.Sprintf(connectionFormat, dbUsername, dbPassword, dbHost, dbPort, dbSchema))

			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
			}
		})
	}
}

func TestIntegration(t *testing.T) {
	mtr := mocktracer.New()
	opentracing.SetGlobalTracer(mtr)
	ctx := context.Background()

	const query = "SELECT * FROM employee LIMIT 1"
	const insertQuery = "INSERT INTO employee(name) value (?)"

	db, err := Open("mysql", fmt.Sprintf(connectionFormat, dbUsername, dbPassword, dbHost, dbPort, dbSchema))
	assert.NoError(t, err)
	assert.NotNil(t, db)
	db.SetConnMaxLifetime(time.Minute)
	db.SetMaxIdleConns(10)
	db.SetMaxOpenConns(10)

	t.Run("db.Ping", func(t *testing.T) {
		mtr.Reset()
		assert.NoError(t, db.Ping(ctx))
		assertSpan(t, mtr.FinishedSpans()[0], "db.Ping", "")
	})

	t.Run("db.Stats", func(t *testing.T) {
		mtr.Reset()
		stats := db.Stats(ctx)
		assert.NotNil(t, stats)
		assertSpan(t, mtr.FinishedSpans()[0], "db.Stats", "")
	})

	t.Run("db.Exec", func(t *testing.T) {
		result, err := db.Exec(ctx, "CREATE TABLE IF NOT EXISTS employee(id int NOT NULL AUTO_INCREMENT PRIMARY KEY,name VARCHAR(255) NOT NULL)")
		assert.NoError(t, err)
		count, err := result.RowsAffected()
		assert.NoError(t, err)
		assert.True(t, count >= 0)
		mtr.Reset()
		result, err = db.Exec(ctx, insertQuery, "patron")
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assertSpan(t, mtr.FinishedSpans()[0], "db.Exec", insertQuery)
	})

	t.Run("db.Query", func(t *testing.T) {
		mtr.Reset()
		rows, err := db.Query(ctx, query)
		defer func() {
			assert.NoError(t, rows.Close())
		}()
		assert.NoError(t, err)
		assert.NotNil(t, rows)
		assertSpan(t, mtr.FinishedSpans()[0], "db.Query", query)
	})

	t.Run("db.QueryRow", func(t *testing.T) {
		mtr.Reset()
		row := db.QueryRow(ctx, query)
		assert.NotNil(t, row)
		assertSpan(t, mtr.FinishedSpans()[0], "db.QueryRow", query)
	})

	t.Run("db.Driver", func(t *testing.T) {
		mtr.Reset()
		drv := db.Driver(ctx)
		assert.NotNil(t, drv)
		assertSpan(t, mtr.FinishedSpans()[0], "db.Driver", "")
	})

	t.Run("stmt", func(t *testing.T) {
		mtr.Reset()
		stmt, err := db.Prepare(ctx, query)
		assert.NoError(t, err)
		assertSpan(t, mtr.FinishedSpans()[0], "db.Prepare", query)

		t.Run("stmt.Exec", func(t *testing.T) {
			mtr.Reset()
			result, err := stmt.Exec(ctx)
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assertSpan(t, mtr.FinishedSpans()[0], "stmt.Exec", query)
		})

		t.Run("stmt.Query", func(t *testing.T) {
			mtr.Reset()
			rows, err := stmt.Query(ctx)
			assert.NoError(t, err)
			defer func() {
				assert.NoError(t, rows.Close())
			}()
			assertSpan(t, mtr.FinishedSpans()[0], "stmt.Query", query)
		})

		t.Run("stmt.QueryRow", func(t *testing.T) {
			mtr.Reset()
			row := stmt.QueryRow(ctx)
			assert.NotNil(t, row)
			assertSpan(t, mtr.FinishedSpans()[0], "stmt.QueryRow", query)
		})

		mtr.Reset()
		assert.NoError(t, stmt.Close(ctx))
		assertSpan(t, mtr.FinishedSpans()[0], "stmt.Close", "")
	})

	t.Run("conn", func(t *testing.T) {
		mtr.Reset()
		conn, err := db.Conn(ctx)
		assert.NoError(t, err)
		assertSpan(t, mtr.FinishedSpans()[0], "db.Conn", "")

		t.Run("conn.Ping", func(t *testing.T) {
			mtr.Reset()
			assert.NoError(t, conn.Ping(ctx))
			assertSpan(t, mtr.FinishedSpans()[0], "conn.Ping", "")
		})

		t.Run("conn.Exec", func(t *testing.T) {
			mtr.Reset()
			result, err := conn.Exec(ctx, insertQuery, "patron")
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assertSpan(t, mtr.FinishedSpans()[0], "conn.Exec", insertQuery)
		})

		t.Run("conn.Query", func(t *testing.T) {
			mtr.Reset()
			rows, err := conn.Query(ctx, query)
			assert.NoError(t, err)
			defer func() {
				assert.NoError(t, rows.Close())
			}()
			assertSpan(t, mtr.FinishedSpans()[0], "conn.Query", query)
		})

		t.Run("conn.QueryRow", func(t *testing.T) {
			mtr.Reset()
			row := conn.QueryRow(ctx, query)
			var id int
			var name string
			assert.NoError(t, row.Scan(&id, &name))
			assertSpan(t, mtr.FinishedSpans()[0], "conn.QueryRow", query)
		})

		t.Run("conn.Prepare", func(t *testing.T) {
			mtr.Reset()
			stmt, err := conn.Prepare(ctx, query)
			assert.NoError(t, err)
			assert.NoError(t, stmt.Close(ctx))
			assertSpan(t, mtr.FinishedSpans()[0], "conn.Prepare", query)
		})

		t.Run("conn.BeginTx", func(t *testing.T) {
			mtr.Reset()
			tx, err := conn.BeginTx(ctx, nil)
			assert.NoError(t, err)
			assert.NoError(t, tx.Commit(ctx))
			assertSpan(t, mtr.FinishedSpans()[0], "conn.BeginTx", "")
		})

		mtr.Reset()
		assert.NoError(t, conn.Close(ctx))
		assertSpan(t, mtr.FinishedSpans()[0], "conn.Close", "")
	})

	t.Run("tx", func(t *testing.T) {
		mtr.Reset()
		tx, err := db.BeginTx(ctx, nil)
		assert.NoError(t, err)
		assert.NotNil(t, tx)
		assertSpan(t, mtr.FinishedSpans()[0], "db.BeginTx", "")

		t.Run("tx.Exec", func(t *testing.T) {
			mtr.Reset()
			result, err := tx.Exec(ctx, insertQuery, "patron")
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assertSpan(t, mtr.FinishedSpans()[0], "tx.Exec", insertQuery)
		})

		t.Run("tx.Query", func(t *testing.T) {
			mtr.Reset()
			rows, err := tx.Query(ctx, query)
			assert.NoError(t, err)
			defer func() {
				assert.NoError(t, rows.Close())
			}()
			assertSpan(t, mtr.FinishedSpans()[0], "tx.Query", query)
		})

		t.Run("tx.QueryRow", func(t *testing.T) {
			mtr.Reset()
			row := tx.QueryRow(ctx, query)
			var id int
			var name string
			assert.NoError(t, row.Scan(&id, &name))
			assertSpan(t, mtr.FinishedSpans()[0], "tx.QueryRow", query)
		})

		t.Run("tx.Prepare", func(t *testing.T) {
			mtr.Reset()
			stmt, err := tx.Prepare(ctx, query)
			assert.NoError(t, err)
			assert.NoError(t, stmt.Close(ctx))
			assertSpan(t, mtr.FinishedSpans()[0], "tx.Prepare", query)
		})

		t.Run("tx.Stmt", func(t *testing.T) {
			stmt, err := db.Prepare(ctx, query)
			assert.NoError(t, err)
			mtr.Reset()
			txStmt := tx.Stmt(ctx, stmt)
			assert.NoError(t, txStmt.Close(ctx))
			assert.NoError(t, stmt.Close(ctx))
			assertSpan(t, mtr.FinishedSpans()[0], "tx.Stmt", query)
		})

		assert.NoError(t, tx.Commit(ctx))

		t.Run("tx.Rollback", func(t *testing.T) {
			tx, err := db.BeginTx(ctx, nil)
			assert.NoError(t, err)
			assert.NotNil(t, db)

			row := tx.QueryRow(ctx, query)
			var id int
			var name string
			assert.NoError(t, row.Scan(&id, &name))

			mtr.Reset()
			assert.NoError(t, tx.Rollback(ctx))
			assertSpan(t, mtr.FinishedSpans()[0], "tx.Rollback", "")
		})
	})

	mtr.Reset()
	assert.NoError(t, db.Close(ctx))
	assertSpan(t, mtr.FinishedSpans()[0], "db.Close", "")
}

func assertSpan(t *testing.T, sp *mocktracer.MockSpan, opName, statement string) {
	assert.Equal(t, opName, sp.OperationName)
	assert.Equal(t, map[string]interface{}{
		"component":    "sql",
		"db.instance":  "patrondb",
		"db.statement": statement,
		"db.type":      "RDBMS",
		"db.user":      "patron",
		"version":      "dev",
		"error":        false,
	}, sp.Tags())
}

type dockerRuntime struct {
	sql  *dockertest.Resource
	pool *dockertest.Pool
}

func (d *dockerRuntime) startUpContainerSync() error {

	pool, err := dockertest.NewPool("")
	if err != nil {
		return err
	}
	d.pool = pool
	d.pool.MaxWait = time.Minute * 2

	d.sql, err = d.pool.RunWithOptions(&dockertest.RunOptions{Repository: "mysql",
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
		}})
	if err != nil {
		return err
	}

	// optionally print the container logs in stdout
	d.tailLogs(d.sql.Container.ID, os.Stdout)

	// wait until the container is ready
	return d.pool.Retry(func() error {
		db, err := sql.Open("mysql", fmt.Sprintf(connectionFormat, dbUsername, dbPassword, dbHost, dbPort, dbSchema))
		if err != nil {
			// container not ready ... return error to try again
			return err
		}
		return db.Ping()
	})
}

func (d *dockerRuntime) tailLogs(containerID string, out io.Writer) {
	opts := docker.LogsOptions{
		Context: context.Background(),

		Stderr:      true,
		Stdout:      true,
		Follow:      true,
		Timestamps:  true,
		RawTerminal: true,

		Container: containerID,

		OutputStream: out,
	}

	// show the logs on a different thread
	go func(d *dockerRuntime) {
		err := d.pool.Client.Logs(opts)
		if err != nil {
			log.Errorf("could not forward container logs to write %v", err)
		}
	}(d)
}

func (d *dockerRuntime) tearDownContainerSync() error {
	return d.pool.Purge(d.sql)
}
