# SchemaSync

Compare database schemas across environments and generate migration files with rollback support. Works with PostgreSQL, MySQL, and SQLite.

## Features

- **Schema Extraction** — Connect to any supported database and extract the current schema
- **Schema Comparison** — Diff two schemas and produce a clear list of changes
- **Migration Generation** — Generate SQL migration files with UP and DOWN (rollback) blocks
- **Multi-DB Support** — Works with PostgreSQL, MySQL, and SQLite via a unified interface
- **Environment Tracking** — Track schemas across dev/staging/prod environments

## Installation

```bash
go install github.com/fuleinist/schema-sync@latest
```

Or build from source:

```bash
git clone https://github.com/fuleinist/schema-sync.git
cd schema-sync
go build -o schema-sync ./cmd
```

## Quick Start

### 1. Initialize a project

```bash
schema-sync init
```

This creates a `.schema-sync/config.yaml` file in your current directory.

### 2. Capture a schema snapshot

```bash
# PostgreSQL
schema-sync snapshot postgres "host=localhost port=5432 user=admin password=secret dbname=mydb" dev

# MySQL
schema-sync snapshot mysql "user:password@tcp(localhost:3306)/mydb" dev

# SQLite
schema-sync snapshot sqlite "./mydb.sqlite" dev
```

### 3. Compare environments

```bash
schema-sync diff dev staging
```

### 4. Generate migrations

```bash
schema-sync migrate prod
```

This creates a migration file in the `migrations/` directory with UP and DOWN blocks.

## Commands

| Command | Description |
|---------|-------------|
| `init` | Initialize SchemaSync in the current directory |
| `snapshot <dbtype> <connstr> <env>` | Capture schema snapshot for an environment |
| `diff <env1> <env2>` | Compare two environment schemas |
| `migrate <env>` | Generate migration file for target environment |
| `status` | Show tracked environments and last sync times |

## Configuration

SchemaSync stores configuration in `.schema-sync/config.yaml`:

```yaml
snapshotDir: .schema-sync/snapshots
outputDir: migrations
```

## Supported Databases

| Database | Connection String Format |
|----------|-------------------------|
| PostgreSQL | `host=localhost port=5432 user=admin password=secret dbname=mydb` |
| MySQL | `user:password@tcp(localhost:3306)/mydb` |
| SQLite | `./path/to/database.sqlite` |

## Migration Format

Generated migrations use a simple format with UP and DOWN blocks:

```sql
-- +migrate Up
-- +migrate StatementBegin
CREATE TABLE users (
  id INTEGER NOT NULL PRIMARY KEY,
  email VARCHAR(255) NOT NULL
);
-- +migrate StatementEnd

-- +migrate Down
-- +migrate StatementBegin
DROP TABLE IF EXISTS users;
-- +migrate StatementEnd
```

## License

MIT
