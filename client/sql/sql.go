// Package sql provides a client with included tracing capabilities.
package sql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"regexp"
	"time"

	patrontrace "github.com/beatlabs/patron/observability/trace"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// TODO: MEtrics
// var opDurationMetrics *prometheus.HistogramVec

// func init() {
// 	opDurationMetrics = prometheus.NewHistogramVec(
// 		prometheus.HistogramOpts{
// 			Namespace: "client",
// 			Subsystem: "sql",
// 			Name:      "cmd_duration_seconds",
// 			Help:      "SQL commands completed by the client.",
// 		},
// 		[]string{"op", "success"},
// 	)
// 	prometheus.MustRegister(opDurationMetrics)
// }

type connInfo struct {
	userAttr     attribute.KeyValue
	instanceAttr attribute.KeyValue
	dbNameAttr   attribute.KeyValue
}

func (c *connInfo) startSpan(ctx context.Context, opName, stmt string) (context.Context, trace.Span) {
	return patrontrace.Tracer().Start(ctx, opName,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(c.userAttr, c.instanceAttr, c.dbNameAttr, attribute.String("db.statement", stmt)))
}

// Conn represents a single database connection.
type Conn struct {
	connInfo
	conn *sql.Conn
}

// DSNInfo contains information extracted from a valid
// connection string. Additional parameters provided are discarded.
type DSNInfo struct {
	Driver   string
	DBName   string
	Address  string
	User     string
	Protocol string
}

// BeginTx starts a transaction.
func (c *Conn) BeginTx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	op := "conn.BeginTx"
	ctx, sp := c.startSpan(ctx, op, "")
	defer sp.End()

	start := time.Now()
	tx, err := c.conn.BeginTx(ctx, opts)
	observeDuration(ctx, start, op, err)
	if err != nil {
		return nil, err
	}

	return &Tx{tx: tx}, nil
}

// Close returns the connection to the connection pool.
func (c *Conn) Close(ctx context.Context) error {
	op := "conn.Close"
	ctx, sp := c.startSpan(ctx, op, "")
	defer sp.End()
	start := time.Now()
	err := c.conn.Close()
	observeDuration(ctx, start, op, err)
	return err
}

// Exec executes a query without returning any rows.
func (c *Conn) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	op := "conn.Exec"
	ctx, sp := c.startSpan(ctx, op, query)
	defer sp.End()
	start := time.Now()
	res, err := c.conn.ExecContext(ctx, query, args...)
	observeDuration(ctx, start, op, err)
	return res, err
}

// Ping verifies the connection to the database is still alive.
func (c *Conn) Ping(ctx context.Context) error {
	op := "conn.Ping"
	ctx, sp := c.startSpan(ctx, op, "")
	defer sp.End()
	start := time.Now()
	err := c.conn.PingContext(ctx)
	observeDuration(ctx, start, op, err)
	return err
}

// Prepare creates a prepared statement for later queries or executions.
func (c *Conn) Prepare(ctx context.Context, query string) (*Stmt, error) {
	op := "conn.Prepare"
	ctx, sp := c.startSpan(ctx, op, query)
	defer sp.End()
	start := time.Now()
	stmt, err := c.conn.PrepareContext(ctx, query)
	observeDuration(ctx, start, op, err)
	if err != nil {
		return nil, err
	}
	return &Stmt{stmt: stmt}, nil
}

// Query executes a query that returns rows.
func (c *Conn) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	op := "conn.Query"
	ctx, sp := c.startSpan(ctx, op, query)
	defer sp.End()
	start := time.Now()
	rows, err := c.conn.QueryContext(ctx, query, args...)
	observeDuration(ctx, start, op, err)
	if err != nil {
		return nil, err
	}

	return rows, nil
}

// QueryRow executes a query that is expected to return at most one row.
func (c *Conn) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	op := "conn.QueryRow"
	ctx, sp := c.startSpan(ctx, op, query)
	defer sp.End()
	start := time.Now()
	row := c.conn.QueryRowContext(ctx, query, args...)
	observeDuration(ctx, start, op, nil)
	return row
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
	info := parseDSN(dataSourceName)
	connInfo := connInfo{
		userAttr:     attribute.String("db.user", info.User),
		instanceAttr: attribute.String("db.instance", info.Address),
		dbNameAttr:   attribute.String("db.name", info.DBName),
	}

	return &DB{connInfo: connInfo, db: db}, nil
}

// OpenDB opens a database.
func OpenDB(c driver.Connector) *DB {
	db := sql.OpenDB(c)
	return &DB{db: db}
}

// FromDB wraps an opened db. This allows to support libraries that provide
// *sql.DB like sqlmock.
func FromDB(db *sql.DB) *DB {
	return &DB{db: db}
}

// DB returns the underlying db. This is useful for SQL code that does not
// require tracing.
func (db *DB) DB() *sql.DB {
	return db.db
}

// BeginTx starts a transaction.
func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	op := "db.BeginTx"
	ctx, sp := db.startSpan(ctx, op, "")
	defer sp.End()
	start := time.Now()
	tx, err := db.db.BeginTx(ctx, opts)
	observeDuration(ctx, start, op, err)
	if err != nil {
		return nil, err
	}
	return &Tx{tx: tx, connInfo: db.connInfo}, nil
}

// Close closes the database, releasing any open resources.
func (db *DB) Close(ctx context.Context) error {
	op := "db.Close"
	ctx, sp := db.startSpan(ctx, op, "")
	defer sp.End()
	start := time.Now()
	err := db.db.Close()
	observeDuration(ctx, start, op, err)
	return err
}

// Conn returns a connection.
func (db *DB) Conn(ctx context.Context) (*Conn, error) {
	op := "db.Conn"
	ctx, sp := db.startSpan(ctx, op, "")
	defer sp.End()
	start := time.Now()
	conn, err := db.db.Conn(ctx)
	observeDuration(ctx, start, op, err)
	if err != nil {
		return nil, err
	}

	return &Conn{conn: conn, connInfo: db.connInfo}, nil
}

// Driver returns the database's underlying driver.
func (db *DB) Driver(ctx context.Context) driver.Driver {
	op := "db.Driver"
	ctx, sp := db.startSpan(ctx, op, "")
	defer sp.End()
	start := time.Now()
	drv := db.db.Driver()
	observeDuration(ctx, start, op, nil)
	return drv
}

// Exec executes a query without returning any rows.
func (db *DB) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	op := "db.Exec"
	ctx, sp := db.startSpan(ctx, op, query)
	defer sp.End()
	start := time.Now()
	res, err := db.db.ExecContext(ctx, query, args...)
	observeDuration(ctx, start, op, err)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// Ping verifies a connection to the database is still alive, establishing a connection if necessary.
func (db *DB) Ping(ctx context.Context) error {
	op := "db.Ping"
	ctx, sp := db.startSpan(ctx, op, "")
	defer sp.End()
	start := time.Now()
	err := db.db.PingContext(ctx)
	observeDuration(ctx, start, op, err)
	return err
}

// Prepare creates a prepared statement for later queries or executions.
func (db *DB) Prepare(ctx context.Context, query string) (*Stmt, error) {
	op := "db.Prepare"
	ctx, sp := db.startSpan(ctx, op, query)
	defer sp.End()
	start := time.Now()
	stmt, err := db.db.PrepareContext(ctx, query)
	observeDuration(ctx, start, op, err)
	if err != nil {
		return nil, err
	}

	return &Stmt{stmt: stmt, connInfo: db.connInfo, query: query}, nil
}

// Query executes a query that returns rows.
func (db *DB) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	op := "db.Query"
	ctx, sp := db.startSpan(ctx, op, query)
	defer sp.End()
	start := time.Now()
	rows, err := db.db.QueryContext(ctx, query, args...)
	observeDuration(ctx, start, op, err)
	if err != nil {
		return nil, err
	}

	return rows, err
}

// QueryRow executes a query that is expected to return at most one row.
func (db *DB) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	op := "db.QueryRow"
	ctx, sp := db.startSpan(ctx, op, query)
	defer sp.End()
	start := time.Now()
	row := db.db.QueryRowContext(ctx, query, args...)
	observeDuration(ctx, start, op, nil)
	return row
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
	op := "db.Stats"
	ctx, sp := db.startSpan(ctx, op, "")
	defer sp.End()
	start := time.Now()
	stats := db.db.Stats()
	observeDuration(ctx, start, op, nil)
	return stats
}

// Stmt is a prepared statement.
type Stmt struct {
	connInfo
	query string
	stmt  *sql.Stmt
}

// Close closes the statement.
func (s *Stmt) Close(ctx context.Context) error {
	op := "stmt.Close"
	ctx, sp := s.startSpan(ctx, op, "")
	defer sp.End()
	start := time.Now()
	err := s.stmt.Close()
	observeDuration(ctx, start, op, err)
	return err
}

// Exec executes a prepared statement.
func (s *Stmt) Exec(ctx context.Context, args ...interface{}) (sql.Result, error) {
	op := "stmt.Exec"
	ctx, sp := s.startSpan(ctx, op, s.query)
	defer sp.End()
	start := time.Now()
	res, err := s.stmt.ExecContext(ctx, args...)
	observeDuration(ctx, start, op, err)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// Query executes a prepared query statement.
func (s *Stmt) Query(ctx context.Context, args ...interface{}) (*sql.Rows, error) {
	op := "stmt.Query"
	ctx, sp := s.startSpan(ctx, op, s.query)
	defer sp.End()
	start := time.Now()
	rows, err := s.stmt.QueryContext(ctx, args...)
	observeDuration(ctx, start, op, err)
	if err != nil {
		return nil, err
	}

	return rows, nil
}

// QueryRow executes a prepared query statement.
func (s *Stmt) QueryRow(ctx context.Context, args ...interface{}) *sql.Row {
	op := "stmt.QueryRow"
	ctx, sp := s.startSpan(ctx, op, s.query)
	defer sp.End()
	start := time.Now()
	row := s.stmt.QueryRowContext(ctx, args...)
	observeDuration(ctx, start, op, nil)
	return row
}

// Tx is an in-progress database transaction.
type Tx struct {
	connInfo
	tx *sql.Tx
}

// Commit commits the transaction.
func (tx *Tx) Commit(ctx context.Context) error {
	op := "tx.Commit"
	ctx, sp := tx.startSpan(ctx, op, "")
	defer sp.End()
	start := time.Now()
	err := tx.tx.Commit()
	observeDuration(ctx, start, op, err)
	return err
}

// Exec executes a query that doesn't return rows.
func (tx *Tx) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	op := "tx.Exec"
	ctx, sp := tx.startSpan(ctx, op, query)
	defer sp.End()
	start := time.Now()
	res, err := tx.tx.ExecContext(ctx, query, args...)
	observeDuration(ctx, start, op, err)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// Prepare creates a prepared statement for use within a transaction.
func (tx *Tx) Prepare(ctx context.Context, query string) (*Stmt, error) {
	op := "tx.Prepare"
	ctx, sp := tx.startSpan(ctx, op, query)
	defer sp.End()
	start := time.Now()
	stmt, err := tx.tx.PrepareContext(ctx, query)
	observeDuration(ctx, start, op, err)
	if err != nil {
		return nil, err
	}

	return &Stmt{stmt: stmt}, nil
}

// Query executes a query that returns rows.
func (tx *Tx) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	op := "tx.Query"
	ctx, sp := tx.startSpan(ctx, op, query)
	defer sp.End()
	start := time.Now()
	rows, err := tx.tx.QueryContext(ctx, query, args...)
	observeDuration(ctx, start, op, err)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// QueryRow executes a query that is expected to return at most one row.
func (tx *Tx) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	op := "tx.QueryRow"
	ctx, sp := tx.startSpan(ctx, op, query)
	defer sp.End()
	start := time.Now()
	row := tx.tx.QueryRowContext(ctx, query, args...)
	observeDuration(ctx, start, op, nil)
	return row
}

// Rollback aborts the transaction.
func (tx *Tx) Rollback(ctx context.Context) error {
	op := "tx.Rollback"
	ctx, sp := tx.startSpan(ctx, op, "")
	defer sp.End()
	start := time.Now()
	err := tx.tx.Rollback()
	observeDuration(ctx, start, op, err)
	return err
}

// Stmt returns a transaction-specific prepared statement from an existing statement.
func (tx *Tx) Stmt(ctx context.Context, stmt *Stmt) *Stmt {
	op := "tx.Stmt"
	ctx, sp := tx.startSpan(ctx, op, stmt.query)
	defer sp.End()
	start := time.Now()
	st := &Stmt{stmt: tx.tx.StmtContext(ctx, stmt.stmt), connInfo: tx.connInfo, query: stmt.query}
	observeDuration(ctx, start, op, nil)
	return st
}

func parseDSN(dsn string) DSNInfo {
	res := DSNInfo{}

	dsnPattern := regexp.MustCompile(
		`^(?P<driver>.*:\/\/)?(?:(?P<username>.*?)(?::(.*))?@)?` + // [driver://][user[:password]@]
			`(?:(?P<protocol>[^\(]*)(?:\((?P<address>[^\)]*)\))?)?` + // [net[(addr)]]
			`\/(?P<dbname>.*?)` + // /dbname
			`(?:\?(?P<params>[^\?]*))?$`) // [?param1=value1&paramN=valueN]

	matches := dsnPattern.FindStringSubmatch(dsn)
	fields := dsnPattern.SubexpNames()

	for i, match := range matches {
		switch fields[i] {
		case "driver":
			res.Driver = match
		case "username":
			res.User = match
		case "protocol":
			res.Protocol = match
		case "address":
			res.Address = match
		case "dbname":
			res.DBName = match
		}
	}

	return res
}

func observeDuration(ctx context.Context, start time.Time, op string, err error) {
	// patrontrace.SpanComplete(span, err)

	// durationHistogram := patrontrace.Histogram{
	// 	Observer: opDurationMetrics.WithLabelValues(op, strconv.FormatBool(err == nil)),
	// }
	// durationHistogram.Observe(ctx, time.Since(start).Seconds())
}
