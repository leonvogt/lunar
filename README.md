# lunar

A tool for creating and restoring snapshots of PostgreSQL databases.

## Configuration

Lunar uses a `lunar.yml` configuration file. Run `lunar init` to create one interactively, or create it manually:

```yaml
database_url: postgres://localhost:5432/
database: my_database

# Optional: specify a maintenance database for administrative operations
# If not set, lunar will try 'postgres' first, then 'template1'
# maintenance_database: postgres
```

### Configuration Options

| Option | Required | Description |
|--------|----------|-------------|
| `database_url` | Yes | PostgreSQL connection URL (without database name) |
| `database` | Yes | Name of the database to snapshot |
| `maintenance_database` | No | Database to use for admin operations (default: tries `postgres`, then `template1`) |

## Development

**Run tests:**

```bash
go test ./tests
```
