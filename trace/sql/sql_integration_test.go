// +build integration

package sql

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

const (
	// These values are taken from examples/docker-compose.yml
	testPsqlInfo string = "host=localhost port=5432 user=postgres password=test123 sslmode=disable"
)

func TestSQL(t *testing.T) {
	mockedCtx := context.Background()
	db := newDB(mockedCtx, t)
	conn := newConn(mockedCtx, t, db)

	createTestTableWithDB(mockedCtx, t, db)
	createTestTableWithConn(mockedCtx, t, conn)

	insertTestDataWithDB(mockedCtx, t, db)
	insertTestDataWithConn(mockedCtx, t, conn)

	queryTestDataWithDB(mockedCtx, t, db)
	queryTestDataWithConn(mockedCtx, t, conn)

	stats := db.Stats(mockedCtx)
	assert.NotNil(t, stats)

	err := conn.Close(mockedCtx)
	assert.NoError(t, err)
	err = db.Close(mockedCtx)
	assert.NoError(t, err)
}

func newDB(ctx context.Context, t *testing.T) *DB {
	db, err := Open("postgres", testPsqlInfo)
	require.NoError(t, err)

	err = db.Ping(ctx)
	assert.NoError(t, err)

	return db
}

func newConn(ctx context.Context, t *testing.T, db *DB) *Conn {
	conn, err := db.Conn(ctx)
	require.NoError(t, err)

	err = conn.Ping(ctx)
	require.NoError(t, err)

	return conn
}

func createTestTableWithDB(ctx context.Context, t *testing.T, db *DB) {
	// test table is created
	dstmt, err := db.Prepare(ctx, "DROP TABLE IF EXISTS test")
	require.NoError(t, err)
	_, err = dstmt.Exec(ctx)
	require.NoError(t, err)
	err = dstmt.Close(ctx)
	require.NoError(t, err)
	stmt, err := db.Prepare(ctx, "CREATE TABLE test(id int, column1 VARCHAR (50))")
	require.NoError(t, err)
	_, err = stmt.Exec(ctx)
	require.NoError(t, err)
	err = stmt.Close(ctx)
	require.NoError(t, err)
}

func createTestTableWithConn(ctx context.Context, t *testing.T, conn *Conn) {
	// test table is created
	dstmt, err := conn.Prepare(ctx, "DROP TABLE IF EXISTS test")
	require.NoError(t, err)
	_, err = dstmt.Exec(ctx)
	require.NoError(t, err)
	err = dstmt.Close(ctx)
	require.NoError(t, err)
	stmt, err := conn.Prepare(ctx, "CREATE TABLE test(id int, column1 VARCHAR (50))")
	require.NoError(t, err)
	_, err = stmt.Exec(ctx)
	require.NoError(t, err)
	err = stmt.Close(ctx)
	require.NoError(t, err)
}

func insertTestDataWithDB(ctx context.Context, t *testing.T, db *DB) {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{})
	require.NoError(t, err)

	_, err = tx.Exec(ctx, "INSERT INTO test VALUES (1, 'value1')")
	require.NoError(t, err)
	err = tx.Commit(ctx)
	require.NoError(t, err)
}

func insertTestDataWithConn(ctx context.Context, t *testing.T, conn *Conn) {
	tx, err := conn.BeginTx(ctx, &sql.TxOptions{})
	require.NoError(t, err)

	_, err = tx.Exec(ctx, "INSERT INTO test VALUES (2, 'value2')")
	require.NoError(t, err)
	err = tx.Commit(ctx)
	require.NoError(t, err)
}

func queryTestDataWithDB(ctx context.Context, t *testing.T, db *DB) {
	stmt, err := db.Prepare(ctx, "SELECT column1 FROM test where id = 1")
	require.NoError(t, err)

	var column1 string
	row := stmt.QueryRow(ctx)
	err = row.Scan(&column1)
	require.NoError(t, err)

	require.Equal(t, column1, "value1")

	err = stmt.Close(ctx)
	require.NoError(t, err)
}

func queryTestDataWithConn(ctx context.Context, t *testing.T, conn *Conn) {
	stmt, err := conn.Prepare(ctx, "SELECT column1 FROM test where id = 1")
	require.NoError(t, err)

	var column1 string
	row := stmt.QueryRow(ctx)
	err = row.Scan(&column1)
	require.NoError(t, err)

	require.Equal(t, column1, "value1")

	err = stmt.Close(ctx)
	require.NoError(t, err)
}
