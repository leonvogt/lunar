<p align="center">
  <picture >
    <source height="200" srcset="https://github.com/user-attachments/assets/2ae5cdae-dd08-4da0-8786-68547fcabe49">
    <img height="200" alt="Expo Orbit" src="https://github.com/user-attachments/assets/2ae5cdae-dd08-4da0-8786-68547fcabe49">
  </picture>
  <h1 align="center">Lunar</h1>
</p>

A tool for creating and restoring snapshots of PostgreSQL and SQLite databases.

## Installation

Using Homebrew:
```bash
brew install leonvogt/tap/lunar-db
```

Other platforms: Download the latest release binaries from the [Releases page](https://github.com/leonvogt/lunar/releases). Each release contains platform-specific files for manual installation.

## Quick Start

```bash
# Initialize Lunar (creates lunar.yml)
lunar init

# Create a snapshot
lunar snapshot production

# List all snapshots
lunar list

# Restore a snapshot
lunar restore production

# Replace a snapshot (delete and recreate)
lunar replace production

# Remove a snapshot
lunar remove production
```

## Configuration

Lunar uses a `lunar.yml` configuration file. Run `lunar init` to create one interactively, or create it manually.

### PostgreSQL Configuration

```yaml
database_url: postgres://localhost:5432/
database: my_database

# Optional: specify a maintenance database for administrative operations
# If not set, lunar will try 'postgres' first, then 'template1'
# maintenance_database: postgres
```

#### PostgreSQL Options

| Option | Required | Description |
|--------|----------|-------------|
| `database_url` | Yes | PostgreSQL connection URL (without database name) |
| `database` | Yes | Name of the database to snapshot |
| `maintenance_database` | No | Database to use for admin operations (default: tries `postgres`, then `template1`) |

### SQLite Configuration

```yaml
provider: sqlite
database_path: ./myapp.db
snapshot_directory: ./.lunar_snapshots
```

> **Note:** Paths are relative to the `lunar.yml` file location, making configs portable across machines and team members.

#### SQLite Options

| Option | Required | Description |
|--------|----------|-------------|
| `provider` | Yes | Must be `sqlite` for SQLite databases |
| `database_path` | Yes | Path to the SQLite database file (relative or absolute) |
| `snapshot_directory` | No | Directory to store snapshots (default: `.lunar_snapshots` next to database) |

## How It Works

### PostgreSQL
Snapshots are created using PostgreSQL's `CREATE DATABASE ... TEMPLATE` feature, which creates an efficient copy of the database. Restoring a snapshot replaces the current database with the snapshot copy.

### SQLite
Snapshots are simple file copies of the SQLite database file. The tool handles WAL (Write-Ahead Logging) files automatically for databases using WAL mode.

## Development

**Build a local binary:**

```bash
go build -o lunar .
```

This creates a `lunar` executable in the current directory that you can use to test in other projects:

```bash
# Use with full path
/path/to/lunar/lunar init

# Or install globally
go install .
```

**Run tests:**

```bash
go test ./tests
```

**Run only SQLite tests (faster, no Docker required):**

```bash
go test ./tests -run "SQLite"
```

**Run only PostgreSQL tests:**

```bash
go test ./tests -run "Postgres"
```

**Run a specific test:**

```bash
go test -run TestInit ./tests/
```
