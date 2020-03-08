// +build integration

package sql

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
)

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
			got, err := Open(tt.args.driverName, "patron:test123@tcp/patrondb?parseTime=true")

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

func TestOpenPingClose(t *testing.T) {
	ctx := context.Background()
	db, err := Open("mysql", "patron:test123@tcp/patrondb?parseTime=true")
	assert.NoError(t, err)
	assert.NotNil(t, db)
	db.SetConnMaxLifetime(time.Minute)
	db.SetMaxIdleConns(10)
	db.SetMaxOpenConns(10)

	t.Run("db.Ping", func(t *testing.T) {
		assert.NoError(t, db.Ping(ctx))
	})

	t.Run("db.Stats", func(t *testing.T) {
		stats := db.Stats(ctx)
		expected := sql.DBStats{MaxOpenConnections: 10, OpenConnections: 1, InUse: 0, Idle: 1, WaitCount: 0, WaitDuration: 0, MaxIdleClosed: 0, MaxLifetimeClosed: 0}
		assert.Equal(t, expected, stats)
	})

	t.Run("db.Exec", func(t *testing.T) {
		result, err := db.Exec(ctx, "CREATE TABLE IF NOT EXISTS employee(id int NOT NULL AUTO_INCREMENT PRIMARY KEY,name VARCHAR(255) NOT NULL)")
		assert.NoError(t, err)
		count, err := result.RowsAffected()
		assert.NoError(t, err)
		assert.True(t, count >= 0)
		result, err = db.Exec(ctx, "INSERT INTO employee(name) value (?)", "patron")
		assert.NoError(t, err)
		count, err = result.RowsAffected()
		assert.NoError(t, err)
		assert.True(t, count == 1)
	})

	t.Run("db.Query", func(t *testing.T) {
		rows, err := db.Query(ctx, "SELECT * FROM employee LIMIT 1")
		assert.NoError(t, err)
		defer func() {
			assert.NoError(t, rows.Close())
		}()
		for rows.Next() {
			var id int
			var name string
			assert.NoError(t, rows.Scan(&id, &name))
			assert.True(t, id > 0)
			assert.Equal(t, "patron", name)
		}
		assert.NoError(t, rows.Err())
	})

	t.Run("db.QueryRow", func(t *testing.T) {
		row := db.QueryRow(ctx, "SELECT * FROM employee LIMIT 1")
		var id int
		var name string
		assert.NoError(t, row.Scan(&id, &name))
		assert.True(t, id > 0)
		assert.Equal(t, "patron", name)
	})

	t.Run("db.Driver", func(t *testing.T) {
		drv := db.Driver(ctx)
		assert.NotNil(t, drv)
	})

	t.Run("stmt", func(t *testing.T) {
		stmt, err := db.Prepare(ctx, "SELECT * FROM employee LIMIT 1")
		assert.NoError(t, err)

		t.Run("stmt.Exec", func(t *testing.T) {
			result, err := stmt.Exec(ctx)
			assert.NoError(t, err)
			count, err := result.RowsAffected()
			assert.NoError(t, err)
			assert.True(t, count >= 0)
		})

		t.Run("stmt.Query", func(t *testing.T) {
			rows, err := stmt.Query(ctx)
			assert.NoError(t, err)
			defer func() {
				assert.NoError(t, rows.Close())
			}()
			for rows.Next() {
				var id int
				var name string
				assert.NoError(t, rows.Scan(&id, &name))
				assert.True(t, id > 0)
				assert.Equal(t, "patron", name)
			}
			assert.NoError(t, rows.Err())
		})

		t.Run("stmt.QueryRow", func(t *testing.T) {
			row := stmt.QueryRow(ctx)
			var id int
			var name string
			assert.NoError(t, row.Scan(&id, &name))
			assert.True(t, id > 0)
			assert.Equal(t, "patron", name)
		})

		assert.NoError(t, stmt.Close(ctx))
	})

	t.Run("cnn", func(t *testing.T) {
		cnn, err := db.Conn(ctx)
		assert.NoError(t, err)

		t.Run("cnn.Ping", func(t *testing.T) {
			assert.NoError(t, cnn.Ping(ctx))
		})

		t.Run("cnn.Exec", func(t *testing.T) {
			result, err := db.Exec(ctx, "INSERT INTO employee(name) value (?)", "patron")
			assert.NoError(t, err)
			count, err := result.RowsAffected()
			assert.NoError(t, err)
			assert.True(t, count == 1)
		})

		t.Run("cnn.Query", func(t *testing.T) {
			rows, err := cnn.Query(ctx, "SELECT * FROM employee LIMIT 1")
			assert.NoError(t, err)
			defer func() {
				assert.NoError(t, rows.Close())
			}()
			for rows.Next() {
				var id int
				var name string
				assert.NoError(t, rows.Scan(&id, &name))
				assert.True(t, id > 0)
				assert.Equal(t, "patron", name)
			}
			assert.NoError(t, rows.Err())
		})

		t.Run("cnn.QueryRow", func(t *testing.T) {
			row := cnn.QueryRow(ctx, "SELECT * FROM employee LIMIT 1")
			var id int
			var name string
			assert.NoError(t, row.Scan(&id, &name))
			assert.True(t, id > 0)
			assert.Equal(t, "patron", name)
		})

		t.Run("cnn.QueryRow", func(t *testing.T) {
			stmt, err := cnn.Prepare(ctx, "SELECT * FROM employee LIMIT 1")
			assert.NoError(t, err)
			assert.NoError(t, stmt.Close(ctx))
		})

		t.Run("cnn.BeginTx", func(t *testing.T) {
			tx, err := cnn.BeginTx(ctx, nil)
			assert.NoError(t, err)
			assert.NoError(t, tx.Commit(ctx))
		})

		assert.NoError(t, cnn.Close(ctx))
	})

	t.Run("tx", func(t *testing.T) {
		tx, err := db.BeginTx(ctx, nil)
		assert.NoError(t, err)
		assert.NotNil(t, db)

		t.Run("tx.Exec", func(t *testing.T) {
			result, err := tx.Exec(ctx, "INSERT INTO employee(name) value (?)", "patron")
			assert.NoError(t, err)
			count, err := result.RowsAffected()
			assert.NoError(t, err)
			assert.True(t, count == 1)
		})

		t.Run("tx.Query", func(t *testing.T) {
			rows, err := tx.Query(ctx, "SELECT * FROM employee LIMIT 1")
			assert.NoError(t, err)
			defer func() {
				assert.NoError(t, rows.Close())
			}()
			for rows.Next() {
				var id int
				var name string
				assert.NoError(t, rows.Scan(&id, &name))
				assert.True(t, id > 0)
				assert.Equal(t, "patron", name)
			}
			assert.NoError(t, rows.Err())
		})

		t.Run("tx.QueryRow", func(t *testing.T) {
			row := tx.QueryRow(ctx, "SELECT * FROM employee LIMIT 1")
			var id int
			var name string
			assert.NoError(t, row.Scan(&id, &name))
			assert.True(t, id > 0)
			assert.Equal(t, "patron", name)
		})

		t.Run("tx.Prepare", func(t *testing.T) {
			stmt, err := tx.Prepare(ctx, "SELECT * FROM employee LIMIT 1")
			assert.NoError(t, err)
			assert.NoError(t, stmt.Close(ctx))
		})

		t.Run("tx.Stmt", func(t *testing.T) {
			stmt, err := db.Prepare(ctx, "SELECT * FROM employee LIMIT 1")
			assert.NoError(t, err)
			txStmt := tx.Stmt(ctx, stmt)
			assert.NoError(t, txStmt.Close(ctx))
			assert.NoError(t, stmt.Close(ctx))
		})

		assert.NoError(t, tx.Commit(ctx))

		t.Run("tx.Rollback", func(t *testing.T) {
			tx, err := db.BeginTx(ctx, nil)
			assert.NoError(t, err)
			assert.NotNil(t, db)

			row := tx.QueryRow(ctx, "SELECT * FROM employee LIMIT 1")
			var id int
			var name string
			assert.NoError(t, row.Scan(&id, &name))
			assert.True(t, id > 0)
			assert.Equal(t, "patron", name)

			assert.NoError(t, tx.Rollback(ctx))
		})
	})

	assert.NoError(t, db.Close(ctx))
}
