package sql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/thebeatapp/patron/trace"
)

type connInfo struct {
	instance, user string
}

func (c *connInfo) startSpan(
	ctx context.Context,
	opName, stmt string,
) (opentracing.Span, context.Context) {
	return trace.SQLSpan(ctx, opName, "sql", "RDBMS", c.instance, c.user, stmt)
}

// Conn represents a single database connection.
type Conn struct {
	connInfo
	conn *sql.Conn
}

// BeginTx starts a transaction.
func (c *Conn) BeginTx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	sp, _ := c.startSpan(ctx, "conn.BeginTx", "")
	tx, err := c.conn.BeginTx(ctx, opts)
	if err != nil {
		trace.SpanError(sp)
		return nil, err
	}

	trace.SpanSuccess(sp)
	return &Tx{tx: tx}, nil
}

// Close returns the connection to the connection pool.
func (c *Conn) Close(ctx context.Context) error {
	sp, _ := c.startSpan(ctx, "conn.Close", "")
	err := c.conn.Close()
	if err != nil {
		trace.SpanError(sp)
		return err
	}
	trace.SpanSuccess(sp)
	return nil
}

// Exec executes a query without returning any rows.
func (c *Conn) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	sp, _ := c.startSpan(ctx, "conn.Exec", query)
	res, err := c.conn.ExecContext(ctx, query, args...)
	if err != nil {
		trace.SpanError(sp)
		return nil, err
	}
	trace.SpanSuccess(sp)
	return res, nil
}

// Ping verifies the connection to the database is still alive.
func (c *Conn) Ping(ctx context.Context) error {
	sp, _ := c.startSpan(ctx, "conn.Ping", "")
	err := c.conn.PingContext(ctx)
	if err != nil {
		trace.SpanError(sp)
		return err
	}
	trace.SpanSuccess(sp)
	return nil
}

// Prepare creates a prepared statement for later queries or executions.
func (c *Conn) Prepare(ctx context.Context, query string) (*Stmt, error) {
	sp, _ := c.startSpan(ctx, "conn.Prepare", query)
	stmt, err := c.conn.PrepareContext(ctx, query)
	if err != nil {
		trace.SpanError(sp)
		return nil, err
	}
	trace.SpanSuccess(sp)
	return &Stmt{stmt: stmt}, nil
}

// Query executes a query that returns rows.
func (c *Conn) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	sp, _ := c.startSpan(ctx, "conn.Query", query)
	rows, err := c.conn.QueryContext(ctx, query, args...)
	if err != nil {
		trace.SpanError(sp)
		return nil, err
	}
	trace.SpanSuccess(sp)
	return rows, nil
}

// QueryRow executes a query that is expected to return at most one row.
func (c *Conn) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	sp, _ := c.startSpan(ctx, "conn.QueryRow", query)
	defer trace.SpanSuccess(sp)
	return c.conn.QueryRowContext(ctx, query, args...)
}

// DB contains the underlying db to be traced.
type DB struct {
	connInfo
	db *sql.DB
}

// Open opens a database.
func Open(driverName, dataSourceName string) (*DB, error) {
	db, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}
	return &DB{db: db}, nil
}

// OpenDB opens a database.
func OpenDB(c driver.Connector) *DB {
	db := sql.OpenDB(c)
	return &DB{db: db}
}

// BeginTx starts a transaction.
func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	sp, _ := db.startSpan(ctx, "db.BeginTx", "")
	tx, err := db.db.BeginTx(ctx, opts)
	if err != nil {
		trace.SpanError(sp)
		return nil, err
	}
	trace.SpanSuccess(sp)
	return &Tx{tx: tx}, nil
}

// Close closes the database, releasing any open resources.
func (db *DB) Close(ctx context.Context) error {
	sp, _ := db.startSpan(ctx, "db.Close", "")
	err := db.db.Close()
	if err != nil {
		trace.SpanError(sp)
		return err
	}
	trace.SpanSuccess(sp)
	return nil
}

// Conn returns a connection.
func (db *DB) Conn(ctx context.Context) (*Conn, error) {
	sp, _ := db.startSpan(ctx, "db.Conn", "")
	conn, err := db.db.Conn(ctx)
	if err != nil {
		trace.SpanError(sp)
		return nil, err
	}
	trace.SpanSuccess(sp)
	return &Conn{conn: conn, connInfo: db.connInfo}, nil
}

// Driver returns the database's underlying driver.
func (db *DB) Driver(ctx context.Context) driver.Driver {
	sp, _ := db.startSpan(ctx, "db.Driver", "")
	defer trace.SpanSuccess(sp)
	return db.db.Driver()
}

// Exec executes a query without returning any rows.
func (db *DB) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	sp, _ := db.startSpan(ctx, "db.Exec", query)
	res, err := db.db.ExecContext(ctx, query, args...)
	if err != nil {
		trace.SpanError(sp)
		return nil, err
	}
	trace.SpanSuccess(sp)
	return res, nil
}

// Ping verifies a connection to the database is still alive, establishing a connection if necessary.
func (db *DB) Ping(ctx context.Context) error {
	sp, _ := db.startSpan(ctx, "db.Ping", "")
	err := db.db.PingContext(ctx)
	if err != nil {
		trace.SpanError(sp)
		return err
	}
	trace.SpanSuccess(sp)
	return nil
}

// Prepare creates a prepared statement for later queries or executions.
func (db *DB) Prepare(ctx context.Context, query string) (*Stmt, error) {
	sp, _ := db.startSpan(ctx, "db.Prepare", query)
	stmt, err := db.db.PrepareContext(ctx, query)
	if err != nil {
		trace.SpanError(sp)
		return nil, err
	}
	trace.SpanSuccess(sp)
	return &Stmt{stmt: stmt}, nil
}

// Query executes a query that returns rows.
func (db *DB) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	sp, _ := db.startSpan(ctx, "db.Query", query)
	rows, err := db.db.QueryContext(ctx, query, args...)
	if err != nil {
		trace.SpanError(sp)
		return nil, err
	}
	trace.SpanSuccess(sp)
	return rows, err
}

// QueryRow executes a query that is expected to return at most one row.
func (db *DB) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	sp, _ := db.startSpan(ctx, "db.QueryRow", query)
	trace.SpanSuccess(sp)
	return db.db.QueryRowContext(ctx, query, args...)
}

// SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
func (db *DB) SetConnMaxLifetime(d time.Duration) {
	db.db.SetConnMaxLifetime(d)
}

// SetMaxIdleConns sets the maximum number of connections in the idle connection pool.
func (db *DB) SetMaxIdleConns(n int) {
	db.db.SetMaxIdleConns(n)
}

// SetMaxOpenConns sets the maximum number of open connections to the database.
func (db *DB) SetMaxOpenConns(n int) {
	db.db.SetMaxOpenConns(n)
}

// Stats returns database statistics.
func (db *DB) Stats(ctx context.Context) sql.DBStats {
	sp, _ := db.startSpan(ctx, "db.Stats", "")
	defer trace.SpanSuccess(sp)
	return db.db.Stats()
}

// Stmt is a prepared statement.
type Stmt struct {
	connInfo
	stmt *sql.Stmt
}

// Close closes the statement.
func (s *Stmt) Close(ctx context.Context) error {
	sp, _ := s.startSpan(ctx, "stmt.Close", "")
	err := s.stmt.Close()
	if err != nil {
		trace.SpanError(sp)
		return err
	}
	trace.SpanSuccess(sp)
	return nil
}

// Exec executes a prepared statement.
func (s *Stmt) Exec(ctx context.Context, args ...interface{}) (sql.Result, error) {
	sp, _ := s.startSpan(ctx, "stmt.Exec", "")
	res, err := s.stmt.ExecContext(ctx, args...)
	if err != nil {
		trace.SpanError(sp)
		return nil, err
	}
	trace.SpanSuccess(sp)
	return res, nil
}

// Query executes a prepared query statement.
func (s *Stmt) Query(ctx context.Context, args ...interface{}) (*sql.Rows, error) {
	sp, _ := s.startSpan(ctx, "stmt.Query", "")
	rows, err := s.stmt.QueryContext(ctx, args...)
	if err != nil {
		trace.SpanError(sp)
		return nil, err
	}
	trace.SpanSuccess(sp)
	return rows, nil
}

// QueryRow executes a prepared query statement.
func (s *Stmt) QueryRow(ctx context.Context, args ...interface{}) *sql.Row {
	sp, _ := s.startSpan(ctx, "stmt.QueryRow", "")
	defer trace.SpanSuccess(sp)
	return s.stmt.QueryRowContext(ctx, args...)
}

// Tx is an in-progress database transaction.
type Tx struct {
	connInfo
	tx *sql.Tx
}

// Commit commits the transaction.
func (tx *Tx) Commit(ctx context.Context) error {
	sp, _ := tx.startSpan(ctx, "tx.Commit", "")
	err := tx.tx.Commit()
	if err != nil {
		trace.SpanError(sp)
		return err
	}
	trace.SpanSuccess(sp)
	return nil
}

// Exec executes a query that doesn't return rows.
func (tx *Tx) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	sp, _ := tx.startSpan(ctx, "tx.Exec", query)
	res, err := tx.tx.ExecContext(ctx, query, args...)
	if err != nil {
		trace.SpanError(sp)
		return nil, err
	}
	trace.SpanSuccess(sp)
	return res, nil
}

// Prepare creates a prepared statement for use within a transaction.
func (tx *Tx) Prepare(ctx context.Context, query string) (*Stmt, error) {
	sp, _ := tx.startSpan(ctx, "tx.Prepare", query)
	stmt, err := tx.tx.PrepareContext(ctx, query)
	if err != nil {
		trace.SpanError(sp)
		return nil, err
	}
	trace.SpanSuccess(sp)
	return &Stmt{stmt: stmt}, nil
}

// Query executes a query that returns rows.
func (tx *Tx) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	sp, _ := tx.startSpan(ctx, "tx.Query", query)
	rows, err := tx.tx.QueryContext(ctx, query, args...)
	if err != nil {
		trace.SpanError(sp)
		return nil, err
	}
	trace.SpanSuccess(sp)
	return rows, nil
}

// QueryRow executes a query that is expected to return at most one row.
func (tx *Tx) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	sp, _ := tx.startSpan(ctx, "tx.QueryRow", query)
	defer trace.SpanSuccess(sp)
	return tx.tx.QueryRowContext(ctx, query, args...)
}

// Rollback aborts the transaction.
func (tx *Tx) Rollback(ctx context.Context) error {
	sp, _ := tx.startSpan(ctx, "tx.Rollback", "")
	err := tx.tx.Rollback()
	if err != nil {
		trace.SpanError(sp)
		return err
	}
	trace.SpanSuccess(sp)
	return nil
}

// Stmt returns a transaction-specific prepared statement from an existing statement.
func (tx *Tx) Stmt(ctx context.Context, stmt *Stmt) *Stmt {
	sp, _ := tx.startSpan(ctx, "tx.Stmt", "")
	defer trace.SpanSuccess(sp)
	return &Stmt{stmt: tx.tx.StmtContext(ctx, stmt.stmt)}
}
