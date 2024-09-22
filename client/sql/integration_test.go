//go:build integration

package sql

import (
	"context"
	"testing"
	"time"

	"github.com/beatlabs/patron/internal/test"
	"github.com/beatlabs/patron/observability/trace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

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
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := Open(tt.args.driverName, dsn)

			if tt.expectedErr != "" {
				require.EqualError(t, err, tt.expectedErr)
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, got)
			}
		})
	}
}

func TestIntegration(t *testing.T) {
	// Tracing monitor setup.
	exp := tracetest.NewInMemoryExporter()
	tracePublisher := trace.Setup("test", nil, exp)

	ctx := context.Background()

	shutdownProvider, assertCollectMetrics := test.SetupMetrics(ctx, t)
	defer shutdownProvider()

	const query = "SELECT * FROM employee LIMIT 1"
	const insertQuery = "INSERT INTO employee(name) value (?)"

	db, err := Open("mysql", dsn)
	require.NoError(t, err)
	assert.NotNil(t, db)
	db.SetConnMaxLifetime(time.Minute)
	db.SetMaxIdleConns(10)
	db.SetMaxOpenConns(10)

	t.Run("db.Ping", func(t *testing.T) {
		exp.Reset()
		require.NoError(t, db.Ping(ctx))
		require.NoError(t, tracePublisher.ForceFlush(ctx))
		_ = assertCollectMetrics(1)
		assertSpans(t, exp.GetSpans(), "db.Ping", "")
	})

	t.Run("db.Stats", func(t *testing.T) {
		exp.Reset()
		stats := db.Stats(ctx)
		assert.NotNil(t, stats)
		require.NoError(t, tracePublisher.ForceFlush(ctx))
		_ = assertCollectMetrics(1)
		assertSpans(t, exp.GetSpans(), "db.Stats", "")
	})

	t.Run("db.Exec", func(t *testing.T) {
		result, err := db.Exec(ctx, "CREATE TABLE IF NOT EXISTS employee(id int NOT NULL AUTO_INCREMENT PRIMARY KEY,name VARCHAR(255) NOT NULL)")
		require.NoError(t, err)
		count, err := result.RowsAffected()
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(0))
		require.NoError(t, tracePublisher.ForceFlush(ctx))
		exp.Reset()
		result, err = db.Exec(ctx, insertQuery, "patron")
		require.NoError(t, err)
		assert.NotNil(t, result)
		require.NoError(t, tracePublisher.ForceFlush(ctx))
		_ = assertCollectMetrics(1)
		assertSpans(t, exp.GetSpans(), "db.Exec", insertQuery)
	})

	t.Run("db.Query", func(t *testing.T) {
		exp.Reset()
		rows, err := db.Query(ctx, query)
		defer func() {
			require.NoError(t, rows.Close())
		}()
		require.NoError(t, err)
		require.NoError(t, rows.Err())
		assert.NotNil(t, rows)
		require.NoError(t, tracePublisher.ForceFlush(ctx))
		_ = assertCollectMetrics(1)
		assertSpans(t, exp.GetSpans(), "db.Query", query)
	})

	t.Run("db.QueryRow", func(t *testing.T) {
		exp.Reset()
		row := db.QueryRow(ctx, query)
		assert.NotNil(t, row)
		require.NoError(t, tracePublisher.ForceFlush(ctx))
		_ = assertCollectMetrics(1)
		assertSpans(t, exp.GetSpans(), "db.QueryRow", query)
	})

	t.Run("db.Driver", func(t *testing.T) {
		exp.Reset()
		drv := db.Driver(ctx)
		assert.NotNil(t, drv)
		require.NoError(t, tracePublisher.ForceFlush(ctx))
		_ = assertCollectMetrics(1)
		assertSpans(t, exp.GetSpans(), "db.Driver", "")
	})

	t.Run("stmt", func(t *testing.T) {
		exp.Reset()
		stmt, err := db.Prepare(ctx, query)
		require.NoError(t, err)
		require.NoError(t, tracePublisher.ForceFlush(ctx))
		_ = assertCollectMetrics(1)
		assertSpans(t, exp.GetSpans(), "db.Prepare", query)

		t.Run("stmt.Exec", func(t *testing.T) {
			exp.Reset()
			result, err := stmt.Exec(ctx)
			require.NoError(t, err)
			assert.NotNil(t, result)
			require.NoError(t, tracePublisher.ForceFlush(ctx))
			_ = assertCollectMetrics(1)
			assertSpans(t, exp.GetSpans(), "stmt.Exec", query)
		})

		t.Run("stmt.Query", func(t *testing.T) {
			exp.Reset()
			rows, err := stmt.Query(ctx)
			require.NoError(t, err)
			require.NoError(t, rows.Err())
			defer func() {
				require.NoError(t, rows.Close())
			}()
			require.NoError(t, tracePublisher.ForceFlush(ctx))
			_ = assertCollectMetrics(1)
			assertSpans(t, exp.GetSpans(), "stmt.Query", query)
		})

		t.Run("stmt.QueryRow", func(t *testing.T) {
			exp.Reset()
			row := stmt.QueryRow(ctx)
			assert.NotNil(t, row)
			require.NoError(t, tracePublisher.ForceFlush(ctx))
			_ = assertCollectMetrics(1)
			assertSpans(t, exp.GetSpans(), "stmt.QueryRow", query)
		})

		exp.Reset()
		require.NoError(t, stmt.Close(ctx))
		require.NoError(t, tracePublisher.ForceFlush(ctx))
		_ = assertCollectMetrics(1)
		assertSpans(t, exp.GetSpans(), "stmt.Close", "")
	})

	t.Run("conn", func(t *testing.T) {
		exp.Reset()
		conn, err := db.Conn(ctx)
		require.NoError(t, err)
		require.NoError(t, tracePublisher.ForceFlush(ctx))
		assertSpans(t, exp.GetSpans(), "db.Conn", "")

		t.Run("conn.Ping", func(t *testing.T) {
			exp.Reset()
			require.NoError(t, conn.Ping(ctx))
			require.NoError(t, tracePublisher.ForceFlush(ctx))
			_ = assertCollectMetrics(1)
			assertSpans(t, exp.GetSpans(), "conn.Ping", "")
		})

		t.Run("conn.Exec", func(t *testing.T) {
			exp.Reset()
			result, err := conn.Exec(ctx, insertQuery, "patron")
			require.NoError(t, err)
			assert.NotNil(t, result)
			require.NoError(t, tracePublisher.ForceFlush(ctx))
			_ = assertCollectMetrics(1)
			assertSpans(t, exp.GetSpans(), "conn.Exec", insertQuery)
		})

		t.Run("conn.Query", func(t *testing.T) {
			exp.Reset()
			rows, err := conn.Query(ctx, query)
			require.NoError(t, err)
			require.NoError(t, rows.Err())
			defer func() {
				require.NoError(t, rows.Close())
			}()
			require.NoError(t, tracePublisher.ForceFlush(ctx))
			_ = assertCollectMetrics(1)
			assertSpans(t, exp.GetSpans(), "conn.Query", query)
		})

		t.Run("conn.QueryRow", func(t *testing.T) {
			exp.Reset()
			row := conn.QueryRow(ctx, query)
			var id int
			var name string
			require.NoError(t, row.Scan(&id, &name))
			require.NoError(t, tracePublisher.ForceFlush(ctx))
			_ = assertCollectMetrics(1)
			assertSpans(t, exp.GetSpans(), "conn.QueryRow", query)
		})

		t.Run("conn.Prepare", func(t *testing.T) {
			exp.Reset()
			stmt, err := conn.Prepare(ctx, query)
			require.NoError(t, err)
			require.NoError(t, tracePublisher.ForceFlush(ctx))
			_ = assertCollectMetrics(1)
			assertSpans(t, exp.GetSpans(), "conn.Prepare", query)
			exp.Reset()
			require.NoError(t, stmt.Close(ctx))
			require.NoError(t, tracePublisher.ForceFlush(ctx))
			_ = assertCollectMetrics(1)
			assertSpans(t, exp.GetSpans(), "stmt.Close", "")
		})

		t.Run("conn.BeginTx", func(t *testing.T) {
			exp.Reset()
			tx, err := conn.BeginTx(ctx, nil)
			require.NoError(t, err)
			require.NoError(t, tracePublisher.ForceFlush(ctx))
			_ = assertCollectMetrics(1)
			assertSpans(t, exp.GetSpans(), "conn.BeginTx", "")
			exp.Reset()
			require.NoError(t, tx.Commit(ctx))
			require.NoError(t, tracePublisher.ForceFlush(ctx))
			_ = assertCollectMetrics(1)
			assertSpans(t, exp.GetSpans(), "tx.Commit", "")
		})

		exp.Reset()
		require.NoError(t, conn.Close(ctx))
		require.NoError(t, tracePublisher.ForceFlush(ctx))
		_ = assertCollectMetrics(1)
		assertSpans(t, exp.GetSpans(), "conn.Close", "")
	})

	t.Run("tx", func(t *testing.T) {
		exp.Reset()
		tx, err := db.BeginTx(ctx, nil)
		require.NoError(t, err)
		assert.NotNil(t, tx)
		require.NoError(t, tracePublisher.ForceFlush(ctx))
		_ = assertCollectMetrics(1)
		assertSpans(t, exp.GetSpans(), "db.BeginTx", "")

		t.Run("tx.Exec", func(t *testing.T) {
			exp.Reset()
			result, err := tx.Exec(ctx, insertQuery, "patron")
			require.NoError(t, err)
			assert.NotNil(t, result)
			require.NoError(t, tracePublisher.ForceFlush(ctx))
			_ = assertCollectMetrics(1)
			assertSpans(t, exp.GetSpans(), "tx.Exec", insertQuery)
		})

		t.Run("tx.Query", func(t *testing.T) {
			exp.Reset()
			rows, err := tx.Query(ctx, query)
			require.NoError(t, err)
			require.NoError(t, rows.Err())
			defer func() {
				require.NoError(t, rows.Close())
			}()
			require.NoError(t, tracePublisher.ForceFlush(ctx))
			_ = assertCollectMetrics(1)
			assertSpans(t, exp.GetSpans(), "tx.Query", query)
		})

		t.Run("tx.QueryRow", func(t *testing.T) {
			exp.Reset()
			row := tx.QueryRow(ctx, query)
			var id int
			var name string
			require.NoError(t, row.Scan(&id, &name))
			require.NoError(t, tracePublisher.ForceFlush(ctx))
			_ = assertCollectMetrics(1)
			assertSpans(t, exp.GetSpans(), "tx.QueryRow", query)
		})

		t.Run("tx.Prepare", func(t *testing.T) {
			exp.Reset()
			stmt, err := tx.Prepare(ctx, query)
			require.NoError(t, err)
			require.NoError(t, tracePublisher.ForceFlush(ctx))
			_ = assertCollectMetrics(1)
			assertSpans(t, exp.GetSpans(), "tx.Prepare", query)
			exp.Reset()
			require.NoError(t, stmt.Close(ctx))
			require.NoError(t, tracePublisher.ForceFlush(ctx))
			_ = assertCollectMetrics(1)
			assertSpans(t, exp.GetSpans(), "stmt.Close", "")
		})

		t.Run("tx.Stmt", func(t *testing.T) {
			exp.Reset()
			stmt, err := db.Prepare(ctx, query)
			require.NoError(t, err)
			require.NoError(t, tracePublisher.ForceFlush(ctx))
			_ = assertCollectMetrics(1)
			assertSpans(t, exp.GetSpans(), "db.Prepare", query)
			exp.Reset()
			txStmt := tx.Stmt(ctx, stmt)
			require.NoError(t, tracePublisher.ForceFlush(ctx))
			_ = assertCollectMetrics(1)
			assertSpans(t, exp.GetSpans(), "tx.Stmt", query)
			exp.Reset()
			require.NoError(t, txStmt.Close(ctx))
			require.NoError(t, tracePublisher.ForceFlush(ctx))
			_ = assertCollectMetrics(1)
			assertSpans(t, exp.GetSpans(), "stmt.Close", "")
			exp.Reset()
			require.NoError(t, stmt.Close(ctx))
			require.NoError(t, tracePublisher.ForceFlush(ctx))
			_ = assertCollectMetrics(1)
			assertSpans(t, exp.GetSpans(), "stmt.Close", "")
		})

		t.Run("tx.Rollback", func(t *testing.T) {
			exp.Reset()
			tx, err := db.BeginTx(ctx, nil)
			require.NoError(t, err)
			assert.NotNil(t, db)
			require.NoError(t, tracePublisher.ForceFlush(ctx))
			_ = assertCollectMetrics(1)
			assertSpans(t, exp.GetSpans(), "db.BeginTx", "")
			exp.Reset()
			row := tx.QueryRow(ctx, query)
			require.NoError(t, tracePublisher.ForceFlush(ctx))
			_ = assertCollectMetrics(1)
			assertSpans(t, exp.GetSpans(), "tx.QueryRow", query)
			var id int
			var name string
			require.NoError(t, row.Scan(&id, &name))
			exp.Reset()
			require.NoError(t, tx.Rollback(ctx))
			require.NoError(t, tracePublisher.ForceFlush(ctx))
			_ = assertCollectMetrics(1)
			assertSpans(t, exp.GetSpans(), "tx.Rollback", "")
		})

		exp.Reset()
		require.NoError(t, tx.Commit(ctx))
		require.NoError(t, tracePublisher.ForceFlush(ctx))
		_ = assertCollectMetrics(1)
		assertSpans(t, exp.GetSpans(), "tx.Commit", "")
		exp.Reset()
		require.NoError(t, db.Close(ctx))
		require.NoError(t, tracePublisher.ForceFlush(ctx))
		_ = assertCollectMetrics(1)
		assertSpans(t, exp.GetSpans(), "db.Close", "")
	})
}

func assertSpans(t *testing.T, spans tracetest.SpanStubs, opName, statement string) {
	assert.Len(t, spans, 1)
	assert.Equal(t, opName, spans[0].Name)
	for _, v := range spans[0].Attributes {
		switch v.Key {
		case "db.instance":
			assert.Equal(t, "localhost:3306", v.Value.AsString())
		case "db.name":
			assert.Equal(t, "patrondb", v.Value.AsString())
		case "db.user":
			assert.Equal(t, "patron", v.Value.AsString())
		case "db.statement":
			assert.Equal(t, statement, v.Value.AsString())
		}
	}
}
