# SQL client

Wrapper around `database/sql` adding OpenTelemetry spans and query duration metrics.

- Package: `github.com/beatlabs/patron/client/sql`

## Open connections

```go
// Open with driver name and DSN
DB, err := patronsql.Open(ctx, "mysql", dsn)
// Wrap an existing *sql.DB
DB := patronsql.FromDB(existingDB)
```

The returned `*patronsql.DB` mirrors `*sql.DB` API and propagates context to statements and transactions:

```go
ctx := context.Background()
var n int
err := DB.QueryRowContext(ctx, "SELECT 1").Scan(&n)
```

- Metrics: query duration histogram labeled with driver, success/error.
- Tracing: spans per Exec/Query/Prepare/Tx with DB attributes.
