//go:build integration
// +build integration

package sql

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/beatlabs/patron/client/sql"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
)

var runtime *sqlRuntime

func TestMain(m *testing.M) {
	var err error
	runtime, err = create(60 * time.Second)
	if err != nil {
		fmt.Printf("could not create mysql runtime: %v\n", err)
		os.Exit(1)
	}
	defer func() {
	}()
	exitCode := m.Run()

	ee := runtime.Teardown()
	if len(ee) > 0 {
		for _, err = range ee {
			fmt.Printf("could not tear down containers: %v\n", err)
		}
	}
	os.Exit(exitCode)
}

func TestOpen(t *testing.T) {
	t.Parallel()
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
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := sql.Open(tt.args.driverName, runtime.DSN())

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

	db, err := sql.Open("mysql", runtime.DSN())
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
