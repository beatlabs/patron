// +build integration

package sql

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
	"github.com/stretchr/testify/suite"
)

const (
	DB_HOST          = "localhost"
	DB_SCHEMA        = "patrondb"
	DB_PORT          = "3307"
	DB_PASSWORD      = "test123"
	DB_ROOT_PASSWORD = "test123"
	DB_USERNAME      = "patron"
)

type SQLTestSuite struct {
	suite.Suite
	sql  *dockertest.Resource
	pool *dockertest.Pool
	db   *DB
	mtr  *mocktracer.MockTracer
	ctx  context.Context
}

func TestSQLTestSuite(t *testing.T) {
	suite.Run(t, new(SQLTestSuite))
}

func (s *SQLTestSuite) SetupSuite() {
	s.ctx = context.Background()

	pool, err := dockertest.NewPool("")
	s.NoError(err)
	s.pool = pool
	s.pool.MaxWait = time.Minute * 2

	s.sql, err = s.pool.RunWithOptions(&dockertest.RunOptions{Repository: "mysql",
		Tag: "5.7.25",
		PortBindings: map[docker.Port][]docker.PortBinding{
			"3306/tcp":  {{HostIP: "", HostPort: "3307"}},
			"33060/tcp": {{HostIP: "", HostPort: "33061"}},
		},
		ExposedPorts: []string{"3306/tcp", "33060/tcp"},
		Env: []string{
			fmt.Sprintf("MYSQL_ROOT_PASSWORD=%s", DB_ROOT_PASSWORD),
			fmt.Sprintf("MYSQL_USER=%s", DB_USERNAME),
			fmt.Sprintf("MYSQL_PASSWORD=%s", DB_PASSWORD),
			fmt.Sprintf("MYSQL_DATABASE=%s", DB_SCHEMA),
			"TIMEZONE=UTC",
		}})
	s.NoError(err)

	// optional : enable logs for the container
	//s.TailLogs(s.sql.Container.ID, os.Stdout)

	// wait until the container is ready
	err = s.pool.Retry(func() error {
		connString := fmt.Sprintf("%s:%s@(%s:%s)/%s?parseTime=true&multiStatements=true", DB_USERNAME, DB_PASSWORD, DB_HOST, DB_PORT, DB_SCHEMA)
		db, err := Open("mysql", connString)
		if err != nil {
			// container not ready ... return error to try again
			return err
		}
		s.NotNil(db)
		s.db = db
		return db.Ping(s.ctx)
	})
	s.NoError(err)

	// start up any other services

	s.mtr = mocktracer.New()
	opentracing.SetGlobalTracer(s.mtr)

	s.db.SetConnMaxLifetime(time.Minute)
	s.db.SetMaxIdleConns(10)
	s.db.SetMaxOpenConns(10)

}

func (s *SQLTestSuite) TailLogs(containerID string, out io.Writer) {
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
	go func(s *SQLTestSuite) {
		err := s.pool.Client.Logs(opts)
		s.NoError(err)
	}(s)
}

func (s *SQLTestSuite) TearDownSuite() {
	err := s.pool.Purge(s.sql)
	s.NoError(err)
}

func (s *SQLTestSuite) BeforeTest(suiteName, testName string) {
	s.mtr.Reset()
}

func (s *SQLTestSuite) TestIntegration() {

	const query = "SELECT * FROM employee LIMIT 1"
	const insertQuery = "INSERT INTO employee(name) value (?)"

	s.Run("db.Ping", func() {
		s.NoError(s.db.Ping(s.ctx))
		assertSpan(s, s.mtr.FinishedSpans()[0], "db.Ping", "")
	})

	s.Run("db.Stats", func() {
		s.mtr.Reset()
		stats := s.db.Stats(s.ctx)
		s.NotNil(stats)
		assertSpan(s, s.mtr.FinishedSpans()[0], "db.Stats", "")
	})

	s.Run("db.Exec", func() {
		result, err := s.db.Exec(s.ctx, "CREATE TABLE IF NOT EXISTS employee(id int NOT NULL AUTO_INCREMENT PRIMARY KEY,name VARCHAR(255) NOT NULL)")
		s.NoError(err)
		count, err := result.RowsAffected()
		s.NoError(err)
		s.True(count >= 0)
		s.mtr.Reset()
		result, err = s.db.Exec(s.ctx, insertQuery, "patron")
		s.NoError(err)
		s.NotNil(result)
		assertSpan(s, s.mtr.FinishedSpans()[0], "db.Exec", insertQuery)
	})

	s.Run("db.Query", func() {
		s.mtr.Reset()
		rows, err := s.db.Query(s.ctx, query)
		defer func() {
			s.NoError(rows.Close())
		}()
		s.NoError(err)
		s.NotNil(rows)
		assertSpan(s, s.mtr.FinishedSpans()[0], "db.Query", query)
	})

	s.Run("db.QueryRow", func() {
		s.mtr.Reset()
		row := s.db.QueryRow(s.ctx, query)
		s.NotNil(row)
		assertSpan(s, s.mtr.FinishedSpans()[0], "db.QueryRow", query)
	})

	s.Run("db.Driver", func() {
		s.mtr.Reset()
		drv := s.db.Driver(s.ctx)
		s.NotNil(drv)
		assertSpan(s, s.mtr.FinishedSpans()[0], "db.Driver", "")
	})

	s.Run("stmt", func() {
		s.mtr.Reset()
		stmt, err := s.db.Prepare(s.ctx, query)
		s.NoError(err)
		assertSpan(s, s.mtr.FinishedSpans()[0], "db.Prepare", query)

		s.Run("stmt.Exec", func() {
			s.mtr.Reset()
			result, err := stmt.Exec(s.ctx)
			s.NoError(err)
			s.NotNil(result)
			assertSpan(s, s.mtr.FinishedSpans()[0], "stmt.Exec", query)
		})

		s.Run("stmt.Query", func() {
			s.mtr.Reset()
			rows, err := stmt.Query(s.ctx)
			s.NoError(err)
			defer func() {
				s.NoError(rows.Close())
			}()
			assertSpan(s, s.mtr.FinishedSpans()[0], "stmt.Query", query)
		})

		s.Run("stmt.QueryRow", func() {
			s.mtr.Reset()
			row := stmt.QueryRow(s.ctx)
			s.NotNil(row)
			assertSpan(s, s.mtr.FinishedSpans()[0], "stmt.QueryRow", query)
		})

		s.mtr.Reset()
		s.NoError(stmt.Close(s.ctx))
		assertSpan(s, s.mtr.FinishedSpans()[0], "stmt.Close", "")
	})

	s.Run("conn", func() {
		s.mtr.Reset()
		conn, err := s.db.Conn(s.ctx)
		s.NoError(err)
		assertSpan(s, s.mtr.FinishedSpans()[0], "db.Conn", "")

		s.Run("conn.Ping", func() {
			s.mtr.Reset()
			s.NoError(conn.Ping(s.ctx))
			assertSpan(s, s.mtr.FinishedSpans()[0], "conn.Ping", "")
		})

		s.Run("conn.Exec", func() {
			s.mtr.Reset()
			result, err := conn.Exec(s.ctx, insertQuery, "patron")
			s.NoError(err)
			s.NotNil(result)
			assertSpan(s, s.mtr.FinishedSpans()[0], "conn.Exec", insertQuery)
		})

		s.Run("conn.Query", func() {
			s.mtr.Reset()
			rows, err := conn.Query(s.ctx, query)
			s.NoError(err)
			defer func() {
				s.NoError(rows.Close())
			}()
			assertSpan(s, s.mtr.FinishedSpans()[0], "conn.Query", query)
		})

		s.Run("conn.QueryRow", func() {
			s.mtr.Reset()
			row := conn.QueryRow(s.ctx, query)
			var id int
			var name string
			s.NoError(row.Scan(&id, &name))
			assertSpan(s, s.mtr.FinishedSpans()[0], "conn.QueryRow", query)
		})

		s.Run("conn.Prepare", func() {
			s.mtr.Reset()
			stmt, err := conn.Prepare(s.ctx, query)
			s.NoError(err)
			s.NoError(stmt.Close(s.ctx))
			assertSpan(s, s.mtr.FinishedSpans()[0], "conn.Prepare", query)
		})

		s.Run("conn.BeginTx", func() {
			s.mtr.Reset()
			tx, err := conn.BeginTx(s.ctx, nil)
			s.NoError(err)
			s.NoError(tx.Commit(s.ctx))
			assertSpan(s, s.mtr.FinishedSpans()[0], "conn.BeginTx", "")
		})

		s.mtr.Reset()
		s.NoError(conn.Close(s.ctx))
		assertSpan(s, s.mtr.FinishedSpans()[0], "conn.Close", "")
	})

	s.Run("tx", func() {
		s.mtr.Reset()
		tx, err := s.db.BeginTx(s.ctx, nil)
		s.NoError(err)
		s.NotNil(tx)
		assertSpan(s, s.mtr.FinishedSpans()[0], "db.BeginTx", "")

		s.Run("tx.Exec", func() {
			s.mtr.Reset()
			result, err := tx.Exec(s.ctx, insertQuery, "patron")
			s.NoError(err)
			s.NotNil(result)
			assertSpan(s, s.mtr.FinishedSpans()[0], "tx.Exec", insertQuery)
		})

		s.Run("tx.Query", func() {
			s.mtr.Reset()
			rows, err := tx.Query(s.ctx, query)
			s.NoError(err)
			defer func() {
				s.NoError(rows.Close())
			}()
			assertSpan(s, s.mtr.FinishedSpans()[0], "tx.Query", query)
		})

		s.Run("tx.QueryRow", func() {
			s.mtr.Reset()
			row := tx.QueryRow(s.ctx, query)
			var id int
			var name string
			s.NoError(row.Scan(&id, &name))
			assertSpan(s, s.mtr.FinishedSpans()[0], "tx.QueryRow", query)
		})

		s.Run("tx.Prepare", func() {
			s.mtr.Reset()
			stmt, err := tx.Prepare(s.ctx, query)
			s.NoError(err)
			s.NoError(stmt.Close(s.ctx))
			assertSpan(s, s.mtr.FinishedSpans()[0], "tx.Prepare", query)
		})

		s.Run("tx.Stmt", func() {
			stmt, err := s.db.Prepare(s.ctx, query)
			s.NoError(err)
			s.mtr.Reset()
			txStmt := tx.Stmt(s.ctx, stmt)
			s.NoError(txStmt.Close(s.ctx))
			s.NoError(stmt.Close(s.ctx))
			assertSpan(s, s.mtr.FinishedSpans()[0], "tx.Stmt", query)
		})

		s.NoError(tx.Commit(s.ctx))

		s.Run("tx.Rollback", func() {
			tx, err := s.db.BeginTx(s.ctx, nil)
			s.NoError(err)
			s.NotNil(s.db)

			row := tx.QueryRow(s.ctx, query)
			var id int
			var name string
			s.NoError(row.Scan(&id, &name))

			s.mtr.Reset()
			s.NoError(tx.Rollback(s.ctx))
			assertSpan(s, s.mtr.FinishedSpans()[0], "tx.Rollback", "")
		})
	})

	s.mtr.Reset()
	s.NoError(s.db.Close(s.ctx))
	assertSpan(s, s.mtr.FinishedSpans()[0], "db.Close", "")
}

func assertSpan(s *SQLTestSuite, sp *mocktracer.MockSpan, opName, statement string) {
	s.Equal(opName, sp.OperationName)
	s.Equal(map[string]interface{}{
		"component":    "sql",
		"db.instance":  "patrondb",
		"db.statement": statement,
		"db.type":      "RDBMS",
		"db.user":      "patron",
		"version":      "dev",
		"error":        false,
	}, sp.Tags())
}
