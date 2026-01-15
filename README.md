<p align="center">
  <picture>
    <source height="200" srcset="https://github.com/user-attachments/assets/2ae5cdae-dd08-4da0-8786-68547fcabe49">
    <img height="200" alt="Lunar Logo" src="https://github.com/user-attachments/assets/2ae5cdae-dd08-4da0-8786-68547fcabe49">
  </picture>
  <h1 align="center">Lunar</h1>
</p>

<p align="center">A fast database snapshot tool for PostgreSQL and SQLite. Create, restore, and manage database snapshots.</p>

## Installation

Using Homebrew:
```bash
brew install leonvogt/tap/lunar-db
```

Other platforms: Download the latest release binaries from the [Releases page](https://github.com/leonvogt/lunar/releases).

## Background

Lunar is a reimplementation of [Stellar](https://github.com/fastmonkeys/stellar), designed to make database snapshots and restoration lightning-fast during development.  
It's perfect for quickly iterating on schema changes, switching branches, or experimenting locally.

**Key Differences from Stellar**:

- PostgreSQL and SQLite support (Stellar supports PostgreSQL and partial MySQL)
- Cross-platform binaries with no language runtime setup required

## How It Works

### PostgreSQL
Lunar leverages PostgreSQL's `CREATE DATABASE ... TEMPLATE` feature to create efficient database copies. Restoring a snapshot performs a fast rename operation rather than slow SQL dumps and imports.

### SQLite
Snapshots are simple file copies of the SQLite database. The tool automatically handles WAL (Write-Ahead Logging) files for databases using WAL mode.

> [!NOTE]  
> Snapshots are full database copies and can consume significant disk space. Monitor your snapshot count to prevent storage issues.

> [!IMPORTANT]  
> Lunar is intended for development use only. Do not use it if you cannot afford data loss.

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

# Replace an existing snapshot
lunar replace production

# Remove a snapshot
lunar remove production
```

## Configuration

Lunar uses a `lunar.yml` configuration file. Run `lunar init` to create one interactively, or create it manually.

### PostgreSQL

```yaml
database_url: postgres://localhost:5432/
database: my_database
```

#### Options

| Option | Required | Description |
|--------|----------|-------------|
| `provider` | No | Set to `postgres` (default if omitted) |
| `database_url` | Yes | PostgreSQL connection URL without database name |
| `database` | Yes | Name of the database to snapshot |
| `maintenance_database` | No | Database for admin operations (default: tries `postgres`, then `template1`) |

### SQLite

```yaml
provider: sqlite
database_path: ./myapp.db
snapshot_directory: ./.lunar_snapshots
```

#### Options

| Option | Required | Description |
|--------|----------|-------------|
| `provider` | Yes | Must be `sqlite` for SQLite databases |
| `database_path` | Yes | Path to the SQLite database file (relative or absolute) |
| `snapshot_directory` | No | Directory to store snapshots (default: `.lunar_snapshots` next to database) |

> **Note:** Paths are relative to the `lunar.yml` file location


## Development

**Build a local binary:**

```bash
go build -o lunar .
```

This creates a `lunar` executable in the current directory. You can test it in other projects by specifying the path to this binary:

```bash
/path/to/lunar/lunar init
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

---

Lunar is inspired by [Stellar](https://github.com/fastmonkeys/stellar) and its approach to fast database snapshots. Many thanks to the Stellar project for the ideas that guided Lunar's design.
