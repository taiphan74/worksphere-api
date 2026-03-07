# migrations/

**Role:** Database migration files (schema versioning).

All DDL changes (CREATE TABLE, ALTER TABLE, DROP INDEX, …) are tracked here as
sequential, numbered migration files so that the database schema can be evolved
in a controlled, reproducible way across environments.

## Tooling
Planned migration tool: **golang-migrate** (`github.com/golang-migrate/migrate`)

## File naming convention
```
{version}_{description}.{up|down}.sql

# Example
000001_create_users_table.up.sql
000001_create_users_table.down.sql
000002_create_workspaces_table.up.sql
000002_create_workspaces_table.down.sql
```

## Common commands (to be added to Makefile)
```bash
# Apply all pending migrations
migrate -path ./migrations -database "postgres://..." up

# Roll back the last migration
migrate -path ./migrations -database "postgres://..." down 1
```
