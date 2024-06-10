//go:build integration

package sql

import (
	"context"
	"testing"
	"time"

	"github.com/beatlabs/patron/observability/trace"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/metric"
	metricsdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
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
	// Tracing monitor setup.
	exp := tracetest.NewInMemoryExporter()
	tracePublisher := trace.Setup("test", nil, exp)

	// Metrics monitor setup.
	read := metricsdk.NewManualReader()
	provider := metricsdk.NewMeterProvider(metricsdk.WithReader(read))
	defer func() {
		assert.NoError(t, provider.Shutdown(context.Background()))
	}()

	otel.SetMeterProvider(provider)

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
		exp.Reset()
		assert.NoError(t, db.Ping(ctx))
		assert.NoError(t, tracePublisher.ForceFlush(ctx))
		assertSpanAndMetric(t, exp.GetSpans(), read, "db.Ping", "", 1)
	})

	t.Run("db.Stats", func(t *testing.T) {
		exp.Reset()
		stats := db.Stats(ctx)
		assert.NotNil(t, stats)
		assert.NoError(t, tracePublisher.ForceFlush(ctx))
		assertSpanAndMetric(t, exp.GetSpans(), read, "db.Stats", "", 1)
	})

	t.Run("db.Exec", func(t *testing.T) {
		result, err := db.Exec(ctx, "CREATE TABLE IF NOT EXISTS employee(id int NOT NULL AUTO_INCREMENT PRIMARY KEY,name VARCHAR(255) NOT NULL)")
		assert.NoError(t, err)
		count, err := result.RowsAffected()
		assert.NoError(t, err)
		assert.True(t, count >= 0)
		assert.NoError(t, tracePublisher.ForceFlush(ctx))
		exp.Reset()
		result, err = db.Exec(ctx, insertQuery, "patron")
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NoError(t, tracePublisher.ForceFlush(ctx))
		assertSpanAndMetric(t, exp.GetSpans(), read, "db.Exec", insertQuery, 1)
	})

	t.Run("db.Query", func(t *testing.T) {
		exp.Reset()
		rows, err := db.Query(ctx, query)
		defer func() {
			assert.NoError(t, rows.Close())
		}()
		assert.NoError(t, err)
		assert.NotNil(t, rows)
		assert.NoError(t, tracePublisher.ForceFlush(ctx))
		assertSpanAndMetric(t, exp.GetSpans(), read, "db.Query", query, 1)
	})

	t.Run("db.QueryRow", func(t *testing.T) {
		exp.Reset()
		row := db.QueryRow(ctx, query)
		assert.NotNil(t, row)
		assert.NoError(t, tracePublisher.ForceFlush(ctx))
		assertSpanAndMetric(t, exp.GetSpans(), read, "db.QueryRow", query, 1)
	})

	t.Run("db.Driver", func(t *testing.T) {
		exp.Reset()
		drv := db.Driver(ctx)
		assert.NotNil(t, drv)
		assert.NoError(t, tracePublisher.ForceFlush(ctx))
		assertSpanAndMetric(t, exp.GetSpans(), read, "db.Driver", "", 1)
	})

	t.Run("stmt", func(t *testing.T) {
		exp.Reset()
		stmt, err := db.Prepare(ctx, query)
		assert.NoError(t, err)
		assert.NoError(t, tracePublisher.ForceFlush(ctx))
		assertSpanAndMetric(t, exp.GetSpans(), read, "db.Prepare", query, 1)

		t.Run("stmt.Exec", func(t *testing.T) {
			exp.Reset()
			result, err := stmt.Exec(ctx)
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.NoError(t, tracePublisher.ForceFlush(ctx))
			assertSpanAndMetric(t, exp.GetSpans(), read, "stmt.Exec", query, 1)
		})

		t.Run("stmt.Query", func(t *testing.T) {
			exp.Reset()
			rows, err := stmt.Query(ctx)
			assert.NoError(t, err)
			defer func() {
				assert.NoError(t, rows.Close())
			}()
			assert.NoError(t, tracePublisher.ForceFlush(ctx))
			assertSpanAndMetric(t, exp.GetSpans(), read, "stmt.Query", query, 1)
		})

		t.Run("stmt.QueryRow", func(t *testing.T) {
			exp.Reset()
			row := stmt.QueryRow(ctx)
			assert.NotNil(t, row)
			assert.NoError(t, tracePublisher.ForceFlush(ctx))
			assertSpanAndMetric(t, exp.GetSpans(), read, "stmt.QueryRow", query, 1)
		})

		exp.Reset()
		assert.NoError(t, stmt.Close(ctx))
		assert.NoError(t, tracePublisher.ForceFlush(ctx))
		assertSpanAndMetric(t, exp.GetSpans(), read, "stmt.Close", "", 1)
	})

	t.Run("conn", func(t *testing.T) {
		exp.Reset()
		conn, err := db.Conn(ctx)
		assert.NoError(t, err)
		assert.NoError(t, tracePublisher.ForceFlush(ctx))
		assertSpanAndMetric(t, exp.GetSpans(), read, "db.Conn", "", 1)

		t.Run("conn.Ping", func(t *testing.T) {
			exp.Reset()
			assert.NoError(t, conn.Ping(ctx))
			assert.NoError(t, tracePublisher.ForceFlush(ctx))
			assertSpanAndMetric(t, exp.GetSpans(), read, "conn.Ping", "", 1)
		})

		t.Run("conn.Exec", func(t *testing.T) {
			exp.Reset()
			result, err := conn.Exec(ctx, insertQuery, "patron")
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.NoError(t, tracePublisher.ForceFlush(ctx))
			assertSpanAndMetric(t, exp.GetSpans(), read, "conn.Exec", insertQuery, 1)
		})

		t.Run("conn.Query", func(t *testing.T) {
			exp.Reset()
			rows, err := conn.Query(ctx, query)
			assert.NoError(t, err)
			defer func() {
				assert.NoError(t, rows.Close())
			}()
			assert.NoError(t, tracePublisher.ForceFlush(ctx))
			assertSpanAndMetric(t, exp.GetSpans(), read, "conn.Query", query, 1)
		})

		t.Run("conn.QueryRow", func(t *testing.T) {
			exp.Reset()
			row := conn.QueryRow(ctx, query)
			var id int
			var name string
			assert.NoError(t, row.Scan(&id, &name))
			assert.NoError(t, tracePublisher.ForceFlush(ctx))
			assertSpanAndMetric(t, exp.GetSpans(), read, "conn.QueryRow", query, 1)
		})

		t.Run("conn.Prepare", func(t *testing.T) {
			exp.Reset()
			stmt, err := conn.Prepare(ctx, query)
			assert.NoError(t, err)
			assert.NoError(t, tracePublisher.ForceFlush(ctx))
			assertSpanAndMetric(t, exp.GetSpans(), read, "conn.Prepare", query, 1)
			exp.Reset()
			assert.NoError(t, stmt.Close(ctx))
			assert.NoError(t, tracePublisher.ForceFlush(ctx))
			assertSpanAndMetric(t, exp.GetSpans(), read, "stmt.Close", "", 1)
		})

		t.Run("conn.BeginTx", func(t *testing.T) {
			exp.Reset()
			tx, err := conn.BeginTx(ctx, nil)
			assert.NoError(t, err)
			assert.NoError(t, tracePublisher.ForceFlush(ctx))
			assertSpanAndMetric(t, exp.GetSpans(), read, "conn.BeginTx", "", 1)
			exp.Reset()
			assert.NoError(t, tx.Commit(ctx))
			assert.NoError(t, tracePublisher.ForceFlush(ctx))
			assertSpanAndMetric(t, exp.GetSpans(), read, "tx.Commit", "", 1)
		})

		exp.Reset()
		assert.NoError(t, conn.Close(ctx))
		assert.NoError(t, tracePublisher.ForceFlush(ctx))
		assertSpanAndMetric(t, exp.GetSpans(), read, "conn.Close", "", 1)
	})

	t.Run("tx", func(t *testing.T) {
		exp.Reset()
		tx, err := db.BeginTx(ctx, nil)
		assert.NoError(t, err)
		assert.NotNil(t, tx)
		assert.NoError(t, tracePublisher.ForceFlush(ctx))
		assertSpanAndMetric(t, exp.GetSpans(), read, "db.BeginTx", "", 1)

		t.Run("tx.Exec", func(t *testing.T) {
			exp.Reset()
			result, err := tx.Exec(ctx, insertQuery, "patron")
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.NoError(t, tracePublisher.ForceFlush(ctx))
			assertSpanAndMetric(t, exp.GetSpans(), read, "tx.Exec", insertQuery, 1)
		})

		t.Run("tx.Query", func(t *testing.T) {
			exp.Reset()
			rows, err := tx.Query(ctx, query)
			assert.NoError(t, err)
			defer func() {
				assert.NoError(t, rows.Close())
			}()
			assert.NoError(t, tracePublisher.ForceFlush(ctx))
			assertSpanAndMetric(t, exp.GetSpans(), read, "tx.Query", query, 1)
		})

		t.Run("tx.QueryRow", func(t *testing.T) {
			exp.Reset()
			row := tx.QueryRow(ctx, query)
			var id int
			var name string
			assert.NoError(t, row.Scan(&id, &name))
			assert.NoError(t, tracePublisher.ForceFlush(ctx))
			assertSpanAndMetric(t, exp.GetSpans(), read, "tx.QueryRow", query, 1)
		})

		t.Run("tx.Prepare", func(t *testing.T) {
			exp.Reset()
			stmt, err := tx.Prepare(ctx, query)
			assert.NoError(t, err)
			assert.NoError(t, tracePublisher.ForceFlush(ctx))
			assertSpanAndMetric(t, exp.GetSpans(), read, "tx.Prepare", query, 1)
			exp.Reset()
			assert.NoError(t, stmt.Close(ctx))
			assert.NoError(t, tracePublisher.ForceFlush(ctx))
			assertSpanAndMetric(t, exp.GetSpans(), read, "stmt.Close", "", 1)
		})

		t.Run("tx.Stmt", func(t *testing.T) {
			exp.Reset()
			stmt, err := db.Prepare(ctx, query)
			assert.NoError(t, err)
			assert.NoError(t, tracePublisher.ForceFlush(ctx))
			assertSpanAndMetric(t, exp.GetSpans(), read, "db.Prepare", query, 1)
			exp.Reset()
			txStmt := tx.Stmt(ctx, stmt)
			assert.NoError(t, tracePublisher.ForceFlush(ctx))
			assertSpanAndMetric(t, exp.GetSpans(), read, "tx.Stmt", query, 1)
			exp.Reset()
			assert.NoError(t, txStmt.Close(ctx))
			assert.NoError(t, tracePublisher.ForceFlush(ctx))
			assertSpanAndMetric(t, exp.GetSpans(), read, "stmt.Close", "", 1)
			exp.Reset()
			assert.NoError(t, stmt.Close(ctx))
			assert.NoError(t, tracePublisher.ForceFlush(ctx))
			assertSpanAndMetric(t, exp.GetSpans(), read, "stmt.Close", "", 1)
		})

		t.Run("tx.Rollback", func(t *testing.T) {
			exp.Reset()
			tx, err := db.BeginTx(ctx, nil)
			assert.NoError(t, err)
			assert.NotNil(t, db)
			assert.NoError(t, tracePublisher.ForceFlush(ctx))
			assertSpanAndMetric(t, exp.GetSpans(), read, "db.BeginTx", "", 1)
			exp.Reset()
			row := tx.QueryRow(ctx, query)
			assert.NoError(t, tracePublisher.ForceFlush(ctx))
			assertSpanAndMetric(t, exp.GetSpans(), read, "tx.QueryRow", query, 1)
			var id int
			var name string
			assert.NoError(t, row.Scan(&id, &name))
			exp.Reset()
			assert.NoError(t, tx.Rollback(ctx))
			assert.NoError(t, tracePublisher.ForceFlush(ctx))
			assertSpanAndMetric(t, exp.GetSpans(), read, "tx.Rollback", "", 1)
		})

		exp.Reset()
		assert.NoError(t, tx.Commit(ctx))
		assert.NoError(t, tracePublisher.ForceFlush(ctx))
		assertSpanAndMetric(t, exp.GetSpans(), read, "tx.Commit", "", 1)
		exp.Reset()
		assert.NoError(t, db.Close(ctx))
		assert.NoError(t, tracePublisher.ForceFlush(ctx))
		assertSpanAndMetric(t, exp.GetSpans(), read, "db.Close", "", 1)
	})
}

func assertSpanAndMetric(t *testing.T, spans tracetest.SpanStubs, read *metric.ManualReader, opName, statement string, metricCount int) {
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

	// Metrics
	collectedMetrics := &metricdata.ResourceMetrics{}
	assert.NoError(t, read.Collect(context.Background(), collectedMetrics))
	assert.Equal(t, metricCount, len(collectedMetrics.ScopeMetrics))

	// TODO: Fix metric collection.
	// assert.Equal(t, metricCount, testutil.CollectAndCount(opDurationMetrics, "client_sql_cmd_duration_seconds"))
	// opDurationMetrics.Reset()
}
