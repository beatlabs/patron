//go:build integration
// +build integration

package sql

import (
	"context"
	"testing"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	// Integration test.
	_ "github.com/go-sql-driver/mysql"
)

const (
	dsn = "patron:test123@(localhost:3306)/patrondb?parseTime=true"
)

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
			got, err := Open(tt.args.driverName, dsn)

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

	db, err := Open("mysql", dsn)
	assert.NoError(t, err)
	assert.NotNil(t, db)
	db.SetConnMaxLifetime(time.Minute)
	db.SetMaxIdleConns(10)
	db.SetMaxOpenConns(10)

	t.Run("db.Ping", func(t *testing.T) {
		mtr.Reset()
		assert.NoError(t, db.Ping(ctx))
		assertSpanAndMetric(t, mtr.FinishedSpans()[0], "db.Ping", "", 1)
	})

	t.Run("db.Stats", func(t *testing.T) {
		mtr.Reset()
		stats := db.Stats(ctx)
		assert.NotNil(t, stats)
		assertSpanAndMetric(t, mtr.FinishedSpans()[0], "db.Stats", "", 1)
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
		assertSpanAndMetric(t, mtr.FinishedSpans()[0], "db.Exec", insertQuery, 1)
	})

	t.Run("db.Query", func(t *testing.T) {
		mtr.Reset()
		rows, err := db.Query(ctx, query)
		defer func() {
			assert.NoError(t, rows.Close())
		}()
		assert.NoError(t, err)
		assert.NotNil(t, rows)
		assertSpanAndMetric(t, mtr.FinishedSpans()[0], "db.Query", query, 1)
	})

	t.Run("db.QueryRow", func(t *testing.T) {
		mtr.Reset()
		row := db.QueryRow(ctx, query)
		assert.NotNil(t, row)
		assertSpanAndMetric(t, mtr.FinishedSpans()[0], "db.QueryRow", query, 1)
	})

	t.Run("db.Driver", func(t *testing.T) {
		mtr.Reset()
		drv := db.Driver(ctx)
		assert.NotNil(t, drv)
		assertSpanAndMetric(t, mtr.FinishedSpans()[0], "db.Driver", "", 1)
	})

	t.Run("stmt", func(t *testing.T) {
		mtr.Reset()
		stmt, err := db.Prepare(ctx, query)
		assert.NoError(t, err)
		assertSpanAndMetric(t, mtr.FinishedSpans()[0], "db.Prepare", query, 1)

		t.Run("stmt.Exec", func(t *testing.T) {
			mtr.Reset()
			result, err := stmt.Exec(ctx)
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assertSpanAndMetric(t, mtr.FinishedSpans()[0], "stmt.Exec", query, 1)
		})

		t.Run("stmt.Query", func(t *testing.T) {
			mtr.Reset()
			rows, err := stmt.Query(ctx)
			assert.NoError(t, err)
			defer func() {
				assert.NoError(t, rows.Close())
			}()
			assertSpanAndMetric(t, mtr.FinishedSpans()[0], "stmt.Query", query, 1)
		})

		t.Run("stmt.QueryRow", func(t *testing.T) {
			mtr.Reset()
			row := stmt.QueryRow(ctx)
			assert.NotNil(t, row)
			assertSpanAndMetric(t, mtr.FinishedSpans()[0], "stmt.QueryRow", query, 1)
		})

		mtr.Reset()
		assert.NoError(t, stmt.Close(ctx))
		assertSpanAndMetric(t, mtr.FinishedSpans()[0], "stmt.Close", "", 1)
	})

	t.Run("conn", func(t *testing.T) {
		mtr.Reset()
		conn, err := db.Conn(ctx)
		assert.NoError(t, err)
		assertSpanAndMetric(t, mtr.FinishedSpans()[0], "db.Conn", "", 1)

		t.Run("conn.Ping", func(t *testing.T) {
			mtr.Reset()
			assert.NoError(t, conn.Ping(ctx))
			assertSpanAndMetric(t, mtr.FinishedSpans()[0], "conn.Ping", "", 1)
		})

		t.Run("conn.Exec", func(t *testing.T) {
			mtr.Reset()
			result, err := conn.Exec(ctx, insertQuery, "patron")
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assertSpanAndMetric(t, mtr.FinishedSpans()[0], "conn.Exec", insertQuery, 1)
		})

		t.Run("conn.Query", func(t *testing.T) {
			mtr.Reset()
			rows, err := conn.Query(ctx, query)
			assert.NoError(t, err)
			defer func() {
				assert.NoError(t, rows.Close())
			}()
			assertSpanAndMetric(t, mtr.FinishedSpans()[0], "conn.Query", query, 1)
		})

		t.Run("conn.QueryRow", func(t *testing.T) {
			mtr.Reset()
			row := conn.QueryRow(ctx, query)
			var id int
			var name string
			assert.NoError(t, row.Scan(&id, &name))
			assertSpanAndMetric(t, mtr.FinishedSpans()[0], "conn.QueryRow", query, 1)
		})

		t.Run("conn.Prepare", func(t *testing.T) {
			mtr.Reset()
			stmt, err := conn.Prepare(ctx, query)
			assert.NoError(t, err)
			assert.NoError(t, stmt.Close(ctx))
			assertSpanAndMetric(t, mtr.FinishedSpans()[0], "conn.Prepare", query, 2)
		})

		t.Run("conn.BeginTx", func(t *testing.T) {
			mtr.Reset()
			tx, err := conn.BeginTx(ctx, nil)
			assert.NoError(t, err)
			assert.NoError(t, tx.Commit(ctx))
			assertSpanAndMetric(t, mtr.FinishedSpans()[0], "conn.BeginTx", "", 2)
		})

		mtr.Reset()
		assert.NoError(t, conn.Close(ctx))
		assertSpanAndMetric(t, mtr.FinishedSpans()[0], "conn.Close", "", 1)
	})

	t.Run("tx", func(t *testing.T) {
		mtr.Reset()
		tx, err := db.BeginTx(ctx, nil)
		assert.NoError(t, err)
		assert.NotNil(t, tx)
		assertSpanAndMetric(t, mtr.FinishedSpans()[0], "db.BeginTx", "", 1)

		t.Run("tx.Exec", func(t *testing.T) {
			mtr.Reset()
			result, err := tx.Exec(ctx, insertQuery, "patron")
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assertSpanAndMetric(t, mtr.FinishedSpans()[0], "tx.Exec", insertQuery, 1)
		})

		t.Run("tx.Query", func(t *testing.T) {
			mtr.Reset()
			rows, err := tx.Query(ctx, query)
			assert.NoError(t, err)
			defer func() {
				assert.NoError(t, rows.Close())
			}()
			assertSpanAndMetric(t, mtr.FinishedSpans()[0], "tx.Query", query, 1)
		})

		t.Run("tx.QueryRow", func(t *testing.T) {
			mtr.Reset()
			row := tx.QueryRow(ctx, query)
			var id int
			var name string
			assert.NoError(t, row.Scan(&id, &name))
			assertSpanAndMetric(t, mtr.FinishedSpans()[0], "tx.QueryRow", query, 1)
		})

		t.Run("tx.Prepare", func(t *testing.T) {
			mtr.Reset()
			stmt, err := tx.Prepare(ctx, query)
			assert.NoError(t, err)
			assert.NoError(t, stmt.Close(ctx))
			assertSpanAndMetric(t, mtr.FinishedSpans()[0], "tx.Prepare", query, 2)
		})

		t.Run("tx.Stmt", func(t *testing.T) {
			stmt, err := db.Prepare(ctx, query)
			assert.NoError(t, err)
			mtr.Reset()
			txStmt := tx.Stmt(ctx, stmt)
			assert.NoError(t, txStmt.Close(ctx))
			assert.NoError(t, stmt.Close(ctx))
			assertSpanAndMetric(t, mtr.FinishedSpans()[0], "tx.Stmt", query, 3)
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
			assertSpanAndMetric(t, mtr.FinishedSpans()[0], "tx.Rollback", "", 4)
		})
	})

	mtr.Reset()
	assert.NoError(t, db.Close(ctx))
	assertSpanAndMetric(t, mtr.FinishedSpans()[0], "db.Close", "", 1)
}

func assertSpanAndMetric(t *testing.T, sp *mocktracer.MockSpan, opName, statement string, metricCount int) {
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

	assert.Equal(t, metricCount, testutil.CollectAndCount(opDurationMetrics, "client_sql_cmd_duration_seconds"))
	opDurationMetrics.Reset()
}
