package sql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
)

func TestParseDSN(t *testing.T) {
	tests := map[string]struct {
		dsn  string
		want DSNInfo
	}{
		"generic case":          {"username:password@protocol(address)/dbname?param=value", DSNInfo{"", "dbname", "address", "username", "protocol"}},
		"empty DSN":             {"/", DSNInfo{"", "", "", "", ""}},
		"dbname only":           {"/dbname", DSNInfo{"", "dbname", "", "", ""}},
		"multiple @":            {"user:p@/ssword@/", DSNInfo{"", "", "", "user", ""}},
		"driver and multiple @": {"postgresql://user:p@/ssword@/", DSNInfo{"postgresql://", "", "", "user", ""}},
		"unix socket":           {"user@unix(/path/to/socket)/dbname?charset=utf8", DSNInfo{"", "dbname", "/path/to/socket", "user", "unix"}},
		"params added":          {"user:password@/dbname?param1=val1&param2=val2&param3=val3", DSNInfo{"", "dbname", "", "user", ""}},
		"IP as address":         {"bruce:hunter2@tcp(127.0.0.1)/arkhamdb?param=value", DSNInfo{"", "arkhamdb", "127.0.0.1", "bruce", "tcp"}},
		"@ in path to socker":   {"user@unix(/path/to/mydir@/socket)/dbname?charset=utf8", DSNInfo{"", "dbname", "/path/to/mydir@/socket", "user", "unix"}},
		"port in address":       {"user:password@tcp(localhost:5555)/dbname?charset=utf8&tls=true", DSNInfo{"", "dbname", "localhost:5555", "user", "tcp"}},
		"multiple ':'":          {"us:er:name:password@memory(localhost:5555)/dbname?charset=utf8&tls=true", DSNInfo{"", "dbname", "localhost:5555", "us", "memory"}},
		"IPv6 provided":         {"user:p@ss(word)@tcp([c023:9350:225b:671a:2cdd:3d83:7c19:ca42]:80)/dbname?loc=Local", DSNInfo{"", "dbname", "[c023:9350:225b:671a:2cdd:3d83:7c19:ca42]:80", "user", "tcp"}},
		"empty string":          {"", DSNInfo{"", "", "", "", ""}},
		"non-matching string":   {"rosebud", DSNInfo{"", "", "", "", ""}},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := parseDSN(tt.dsn)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFromDB(t *testing.T) {
	want := &sql.DB{}
	db := FromDB(want)
	got := db.DB()
	assert.Equal(t, want, got)
}

// Mock driver for testing.
type mockDriver struct{}

func (m mockDriver) Open(_ string) (driver.Conn, error) {
	return &mockConn{}, nil
}

type mockConn struct{}

func (m *mockConn) Prepare(_ string) (driver.Stmt, error) {
	return &mockStmt{}, nil
}

func (m *mockConn) Close() error {
	return nil
}

func (m *mockConn) Begin() (driver.Tx, error) {
	return &mockTx{}, nil
}

type mockStmt struct{}

func (m *mockStmt) Close() error {
	return nil
}

func (m *mockStmt) NumInput() int {
	return 0
}

func (m *mockStmt) Exec(_ []driver.Value) (driver.Result, error) {
	return &mockResult{}, nil
}

func (m *mockStmt) Query(_ []driver.Value) (driver.Rows, error) {
	return &mockRows{}, nil
}

type mockResult struct{}

func (m *mockResult) LastInsertId() (int64, error) {
	return 1, nil
}

func (m *mockResult) RowsAffected() (int64, error) {
	return 1, nil
}

type mockRows struct {
	closed bool
}

func (m *mockRows) Columns() []string {
	return []string{"id", "name"}
}

func (m *mockRows) Close() error {
	m.closed = true
	return nil
}

func (m *mockRows) Next(_ []driver.Value) error {
	return errors.New("no more rows")
}

type mockTx struct{}

func (m *mockTx) Commit() error {
	return nil
}

func (m *mockTx) Rollback() error {
	return nil
}

type failingDriver struct {
	conn *failingConn
}

func (d *failingDriver) Open(_ string) (driver.Conn, error) {
	return d.conn, nil
}

type failingConn struct {
	beginTxErr error
	closeErr   error
	execErr    error
	pingErr    error
	prepareErr error
	queryErr   error
	tx         *failingTx
	stmt       *failingStmt
}

func (c *failingConn) Prepare(_ string) (driver.Stmt, error) {
	if c.prepareErr != nil {
		return nil, c.prepareErr
	}

	if c.stmt != nil {
		return c.stmt, nil
	}

	return &failingStmt{}, nil
}

func (c *failingConn) Close() error {
	return c.closeErr
}

func (c *failingConn) Begin() (driver.Tx, error) {
	if c.beginTxErr != nil {
		return nil, c.beginTxErr
	}

	if c.tx != nil {
		return c.tx, nil
	}

	return &failingTx{}, nil
}

func (c *failingConn) BeginTx(_ context.Context, _ driver.TxOptions) (driver.Tx, error) {
	return c.Begin()
}

func (c *failingConn) Ping(_ context.Context) error {
	return c.pingErr
}

func (c *failingConn) ExecContext(_ context.Context, _ string, _ []driver.NamedValue) (driver.Result, error) {
	if c.execErr != nil {
		return nil, c.execErr
	}

	return &mockResult{}, nil
}

func (c *failingConn) QueryContext(_ context.Context, _ string, _ []driver.NamedValue) (driver.Rows, error) {
	if c.queryErr != nil {
		return nil, c.queryErr
	}

	return &mockRows{}, nil
}

type failingStmt struct {
	closeErr error
	execErr  error
	queryErr error
}

func (s *failingStmt) Close() error {
	return s.closeErr
}

func (s *failingStmt) NumInput() int {
	return -1
}

func (s *failingStmt) Exec(_ []driver.Value) (driver.Result, error) {
	if s.execErr != nil {
		return nil, s.execErr
	}

	return &mockResult{}, nil
}

func (s *failingStmt) Query(_ []driver.Value) (driver.Rows, error) {
	if s.queryErr != nil {
		return nil, s.queryErr
	}

	return &mockRows{}, nil
}

func (s *failingStmt) ExecContext(_ context.Context, _ []driver.NamedValue) (driver.Result, error) {
	if s.execErr != nil {
		return nil, s.execErr
	}

	return &mockResult{}, nil
}

func (s *failingStmt) QueryContext(_ context.Context, _ []driver.NamedValue) (driver.Rows, error) {
	if s.queryErr != nil {
		return nil, s.queryErr
	}

	return &mockRows{}, nil
}

type failingTx struct {
	commitErr   error
	rollbackErr error
	execErr     error
	prepareErr  error
	queryErr    error
	stmt        *failingStmt
}

func (tx *failingTx) Commit() error {
	return tx.commitErr
}

func (tx *failingTx) Rollback() error {
	return tx.rollbackErr
}

func (tx *failingTx) ExecContext(_ context.Context, _ string, _ []driver.NamedValue) (driver.Result, error) {
	if tx.execErr != nil {
		return nil, tx.execErr
	}

	return &mockResult{}, nil
}

func (tx *failingTx) QueryContext(_ context.Context, _ string, _ []driver.NamedValue) (driver.Rows, error) {
	if tx.queryErr != nil {
		return nil, tx.queryErr
	}

	return &mockRows{}, nil
}

func (tx *failingTx) PrepareContext(_ context.Context, _ string) (driver.Stmt, error) {
	if tx.prepareErr != nil {
		return nil, tx.prepareErr
	}

	if tx.stmt != nil {
		return tx.stmt, nil
	}

	return &failingStmt{}, nil
}

func TestDSNInfo(t *testing.T) {
	info := DSNInfo{
		Driver:   "mysql://",
		DBName:   "testdb",
		Address:  "localhost:3306",
		User:     "testuser",
		Protocol: "tcp",
	}

	assert.Equal(t, "mysql://", info.Driver)
	assert.Equal(t, "testdb", info.DBName)
	assert.Equal(t, "localhost:3306", info.Address)
	assert.Equal(t, "testuser", info.User)
	assert.Equal(t, "tcp", info.Protocol)
}

func TestConnInfo_Attributes(t *testing.T) {
	user := "admin"
	instance := "db.example.com:5432"
	dbName := "production"

	ci := connInfo{
		userAttr:     attribute.String("db.user", user),
		instanceAttr: attribute.String("db.instance", instance),
		dbNameAttr:   attribute.String("db.name", dbName),
	}

	assert.Equal(t, "db.user", string(ci.userAttr.Key))
	assert.Equal(t, user, ci.userAttr.Value.AsString())

	assert.Equal(t, "db.instance", string(ci.instanceAttr.Key))
	assert.Equal(t, instance, ci.instanceAttr.Value.AsString())

	assert.Equal(t, "db.name", string(ci.dbNameAttr.Key))
	assert.Equal(t, dbName, ci.dbNameAttr.Value.AsString())
}

func TestOpenDB(t *testing.T) {
	// Register mock driver
	sql.Register("mock-driver", &mockDriver{})

	// Create a mock connector
	mockConnector := dsnConnector{dsn: "", driver: &mockDriver{}}
	db := OpenDB(mockConnector)

	assert.NotNil(t, db)
	assert.NotNil(t, db.db)

	// Clean up
	_ = db.db.Close()
}

// dsnConnector implements driver.Connector for testing.
type dsnConnector struct {
	dsn    string
	driver driver.Driver
}

func (t dsnConnector) Connect(_ context.Context) (driver.Conn, error) {
	return t.driver.Open(t.dsn)
}

func (t dsnConnector) Driver() driver.Driver {
	return t.driver
}

func TestDB_DB(t *testing.T) {
	mockDB := &sql.DB{}
	db := &DB{db: mockDB}

	result := db.DB()
	assert.Equal(t, mockDB, result)
}

func TestDB_SetConnMaxLifetime(t *testing.T) {
	sql.Register("mock-test-lifetime", &mockDriver{})
	stdDB, err := sql.Open("mock-test-lifetime", "")
	require.NoError(t, err)
	defer stdDB.Close()

	db := FromDB(stdDB)
	duration := 5 * time.Minute

	// Should not panic
	db.SetConnMaxLifetime(duration)
}

func TestDB_SetMaxIdleConns(t *testing.T) {
	sql.Register("mock-test-idle", &mockDriver{})
	stdDB, err := sql.Open("mock-test-idle", "")
	require.NoError(t, err)
	defer stdDB.Close()

	db := FromDB(stdDB)

	// Should not panic
	db.SetMaxIdleConns(10)
}

func TestDB_SetMaxOpenConns(t *testing.T) {
	sql.Register("mock-test-open", &mockDriver{})
	stdDB, err := sql.Open("mock-test-open", "")
	require.NoError(t, err)
	defer stdDB.Close()

	db := FromDB(stdDB)

	// Should not panic
	db.SetMaxOpenConns(20)
}

func TestOperationAttr(t *testing.T) {
	op := "db.Query"
	attr := operationAttr(op)

	assert.Equal(t, "op", string(attr.Key))
	assert.Equal(t, op, attr.Value.AsString())
}

func TestObserveDuration(_ *testing.T) {
	ctx := context.Background()
	start := time.Now()
	op := "test.operation"

	// Should not panic
	observeDuration(ctx, start, op, nil)
	observeDuration(ctx, start, op, errors.New("test error"))
}

func TestParseDSN_EdgeCases(t *testing.T) {
	tests := map[string]struct {
		dsn  string
		want DSNInfo
	}{
		"very long dbname": {
			dsn: "user@/verylongdatabasenamewithnospaces",
			want: DSNInfo{
				DBName: "verylongdatabasenamewithnospaces",
				User:   "user",
			},
		},
		"special chars in password": {
			dsn: "user:p@ssw0rd!@#@tcp(localhost)/db",
			want: DSNInfo{
				DBName:   "db",
				Address:  "localhost",
				User:     "user",
				Protocol: "tcp",
			},
		},
		"no protocol": {
			dsn: "user:password@/dbname",
			want: DSNInfo{
				DBName: "dbname",
				User:   "user",
			},
		},
		"driver only": {
			dsn: "postgres:///dbname",
			want: DSNInfo{
				Driver: "postgres://",
				DBName: "dbname",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := parseDSN(tt.dsn)
			assert.Equal(t, tt.want.Driver, got.Driver)
			assert.Equal(t, tt.want.DBName, got.DBName)
			assert.Equal(t, tt.want.User, got.User)
			if tt.want.Address != "" {
				assert.Equal(t, tt.want.Address, got.Address)
			}
			if tt.want.Protocol != "" {
				assert.Equal(t, tt.want.Protocol, got.Protocol)
			}
		})
	}
}

func TestOpen_InvalidDriver(t *testing.T) {
	db, err := Open("nonexistent-driver", "invalid-dsn")

	// Should return error for unknown driver
	require.Error(t, err)
	assert.Nil(t, db)
	assert.ErrorContains(t, err, "db.Open")
}

func TestOpen_Success(t *testing.T) {
	// Register a test driver
	driverName := "test-driver-open"
	sql.Register(driverName, &mockDriver{})

	dsn := "testuser:testpass@tcp(localhost:3306)/testdb"
	db, err := Open(driverName, dsn)

	require.NoError(t, err)
	assert.NotNil(t, db)
	assert.NotNil(t, db.db)

	// Verify parsed info
	assert.Equal(t, "testuser", db.userAttr.Value.AsString())
	assert.Equal(t, "localhost:3306", db.instanceAttr.Value.AsString())
	assert.Equal(t, "testdb", db.dbNameAttr.Value.AsString())

	// Clean up
	err = db.db.Close()
	assert.NoError(t, err)
}

func TestPackageNameConstant(t *testing.T) {
	assert.Equal(t, "sql", packageName)
}

func TestDurationHistogramInitialized(t *testing.T) {
	assert.NotNil(t, durationHistogram)
}

func TestWrapError(t *testing.T) {
	err := errors.New("boom")

	wrapped := wrapError("db.Exec", err)

	require.Error(t, wrapped)
	require.ErrorContains(t, wrapped, "db.Exec")
	require.ErrorIs(t, wrapped, err)
	assert.NoError(t, wrapError("db.Exec", nil))
}

func TestDBExec_WrapsError(t *testing.T) {
	t.Parallel()

	baseErr := errors.New("exec failed")
	driverName := "failing-driver-db-exec"
	sql.Register(driverName, &failingDriver{conn: &failingConn{execErr: baseErr}})
	stdDB, err := sql.Open(driverName, "")
	require.NoError(t, err)
	defer stdDB.Close()

	db := FromDB(stdDB)
	res, err := db.Exec(context.Background(), "INSERT INTO test VALUES (?)", 1)

	assert.Nil(t, res)
	require.Error(t, err)
	require.ErrorContains(t, err, "db.Exec")
	require.ErrorIs(t, err, baseErr)
}

func TestConnPrepare_WrapsError(t *testing.T) {
	t.Parallel()

	baseErr := errors.New("prepare failed")
	driverName := "failing-driver-conn-prepare"
	sql.Register(driverName, &failingDriver{conn: &failingConn{prepareErr: baseErr}})
	stdDB, err := sql.Open(driverName, "")
	require.NoError(t, err)
	defer stdDB.Close()

	db := FromDB(stdDB)
	conn, err := db.Conn(context.Background())
	require.NoError(t, err)
	defer func() {
		require.NoError(t, conn.Close(context.Background()))
	}()

	stmt, err := conn.Prepare(context.Background(), "SELECT 1")

	assert.Nil(t, stmt)
	require.Error(t, err)
	require.ErrorContains(t, err, "conn.Prepare")
	require.ErrorIs(t, err, baseErr)
}

func TestStmtExec_WrapsError(t *testing.T) {
	t.Parallel()

	baseErr := errors.New("stmt exec failed")
	driverName := "failing-driver-stmt-exec"
	sql.Register(driverName, &failingDriver{conn: &failingConn{stmt: &failingStmt{execErr: baseErr}}})
	stdDB, err := sql.Open(driverName, "")
	require.NoError(t, err)
	defer stdDB.Close()

	db := FromDB(stdDB)
	stmt, err := db.Prepare(context.Background(), "UPDATE test SET value = 1")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, stmt.Close(context.Background()))
	}()

	res, err := stmt.Exec(context.Background())

	assert.Nil(t, res)
	require.Error(t, err)
	require.ErrorContains(t, err, "stmt.Exec")
	require.ErrorIs(t, err, baseErr)
}

func TestTxCommit_WrapsError(t *testing.T) {
	t.Parallel()

	baseErr := errors.New("commit failed")
	driverName := "failing-driver-tx-commit"
	sql.Register(driverName, &failingDriver{conn: &failingConn{tx: &failingTx{commitErr: baseErr}}})
	stdDB, err := sql.Open(driverName, "")
	require.NoError(t, err)
	defer stdDB.Close()

	db := FromDB(stdDB)
	tx, err := db.BeginTx(context.Background(), nil)
	require.NoError(t, err)

	err = tx.Commit(context.Background())

	require.Error(t, err)
	require.ErrorContains(t, err, "tx.Commit")
	require.ErrorIs(t, err, baseErr)
}

func TestDBQueryRow_PreservesErrNoRows(t *testing.T) {
	t.Parallel()

	driverName := "failing-driver-query-row"
	sql.Register(driverName, &failingDriver{conn: &failingConn{queryErr: sql.ErrNoRows}})
	stdDB, err := sql.Open(driverName, "")
	require.NoError(t, err)
	defer stdDB.Close()

	db := FromDB(stdDB)
	row := db.QueryRow(context.Background(), "SELECT 1")

	err = row.Scan(new(int))

	require.Error(t, err)
	assert.ErrorIs(t, err, sql.ErrNoRows)
}
