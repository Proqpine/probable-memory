# Probable memory
work-track v1

Using SQlite for managing Activity entries
- Create "<dbname>.db"
- Run `sqlc init`
```yaml
version: "2"
sql:
  - engine: "sqlite"
    queries: "sqlite/queries"
    schema: "migrations"
    gen:
      go:
        package: "sqlite"
        out: "sqlite"

```
- Create `sqlite/queries` directory and `<dbname>.sql` for queries
```sql
-- name: QueryActivities :many
select * from activities;

-- name: QueryActivityByProject :one
select * from activities where project=?;
```
- Run `sqlc generate`
