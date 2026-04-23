# SchemaSync — SPEC.md

## 1. Concept & Vision

SchemaSync is a CLI tool that compares database schemas across environments (dev/staging/prod) and automatically generates migration files with proper rollback support. It works with PostgreSQL, MySQL, and SQLite. The tool detects schema drift before it becomes a crisis and generates safe, reversible migration scripts.

## 2. Design Language

- **Aesthetic**: Clean developer tool with terminal-first UX. Minimal, functional, zero bloat.
- **Color Palette**: Standard terminal colors (no custom palette)
- **Typography**: Monospace (system default terminal font)
- **Output Format**: Structured text output with clear diffs and migration suggestions

## 3. Features & Interactions

### Core Features

1. **Schema Extraction** — Connect to a database and extract current schema (tables, columns, indexes, foreign keys, constraints)
2. **Schema Comparison** — Diff two schemas and produce a list of changes (additions, deletions, modifications)
3. **Migration Generation** — Generate SQL migration files (up + down/rollback) for the detected changes
4. **Multi-DB Support** — Works with PostgreSQL, MySQL, SQLite via a unified interface
5. **Environment Tracking** — Track schemas across dev/staging/prod environments via config file

### CLI Commands

```
schema-sync init                  # Initialize project with config
schema-sync snapshot <env>         # Capture current schema snapshot
schema-sync diff <env1> <env2>    # Compare two environments
schema-sync migrate <env>          # Generate migration for target env
schema-sync status                # Show tracked environments and last sync
```

### Supported Schema Elements
- Tables (create, drop, rename)
- Columns (add, drop, modify type, rename)
- Indexes (create, drop)
- Foreign keys (add, drop)
- Constraints (NOT NULL, UNIQUE, DEFAULT, CHECK)

### Error Handling
- Connection failures: clear error with connection string hints
- Unsupported migrations: warn and skip with explanation
- Empty diff: report no changes needed

## 4. Technical Approach

### Stack
- **Language**: Go 1.21+
- **Database Drivers**: 
  - PostgreSQL: `github.com/lib/pq`
  - MySQL: `github.com/go-sql-driver/mysql`
  - SQLite: `github.com/mattn/go-sqlite3`
- **Architecture**: 
  - `cmd/` — CLI entry points (cobra)
  - `internal/` — Core logic
    - `schema/` — Schema extraction per DB
    - `diff/` — Schema diff engine
    - `migrate/` — Migration file generator
    - `config/` — Config management
  - `migrations/` — Generated migration files (user-editable)

### Data Model
- Schema represented as `[]Table` where `Table` has `[]Column`, `[]Index`, `[]ForeignKey`
- Snapshots stored as JSON files in `.schema-sync/` directory
- Config stored as `.schema-sync/config.yaml`

## 5. Acceptance Criteria

1. `schema-sync init` creates `.schema-sync/config.yaml` in current directory
2. `schema-sync snapshot postgres <connection_string> <env>` captures schema to JSON
3. `schema-sync diff <env1> <env2>` shows human-readable schema differences
4. `schema-sync migrate <env>` generates `migrations/YYYYMMDDHHMMSS_<env>.sql` with UP and DOWN blocks
5. PostgreSQL, MySQL, and SQLite connections work correctly
6. Generated migrations are valid SQL syntax
7. Tests cover core diff and migration generation logic
8. Comprehensive README with usage examples
