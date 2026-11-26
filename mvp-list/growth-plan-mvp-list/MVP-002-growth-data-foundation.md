# MVP-002: Establish the growth data model and immutable accounting invariants

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 days`
- Estimate rationale: Seventeen related tables, Ent generation, database constraints, and migration verification cross a schema boundary and require one integration pass.
- Dependencies: `MVP-001`

## Outcome

The database and Ent layer can represent budgets, task quota, wallets, transfers, immutable ledger entries, tasks/rules, completions, products, inventory, exchange orders, fulfillment outbox, and audit records with enforceable uniqueness and non-negative constraints.

## Context

The design lists 17 `growth_*` schemas and forbids UUID/soft-deleted tasks. Project convention requires a manual SQL archive and currently has maximum archive number `181`. Existing Ent generation uses int64 IDs plus upsert and lock support.

## In Scope

- Add the 17 Ent schemas under `backend/ent/schema/growth_*.go` and required `User` edges.
- Add `backend/sqlArchiving/growth/ddl/182_create_growth_tables.sql` for MySQL 8/GoldenDB with all tables, indexes, unique keys, and verification statements.
- Model budget row id `1`; task status `DRAFT/ACTIVE/OFFLINE`; no `deleted_at`; immutable task code; rule/product version uniqueness; completion request uniqueness; order/fulfillment/inventory uniqueness.
- Add Ent/repository tests for key constraints and fixed-point fields.
- Run `make generate` and confirm generated code is committed.

## Out of Scope

- No budget operations, business services, HTTP routes, UI, seed tasks, or production data backfill.
- No edits to historical `backend/migrations/` files and no UUID/snowflake primary keys.

## Implementation Notes

- Use `SchemaType(map[string]string{dialect.MySQL: "decimal(20,8)"})` for monetary/points columns where required by the design and keep persistence representation consistent across Ent and SQL.
- Encode indexes in both Ent schema and DDL so generation and manual archives agree.
- Store rule conditions and benefit snapshots as MySQL `JSON` when supported; validate application JSON before persistence.
- Test initial empty database and repeat execution of the archive.

## Acceptance Criteria

- [x] `make generate` completes and the 17 schema types are present with the required table annotations, indexes, and user associations.
- [x] SQL executes successfully twice against MySQL 8/GoldenDB test database and creates exactly the 17 planned tables.
- [x] Database rejects duplicate `task_code`, duplicate `(task_id, version)`, duplicate `(user_id, task_id, cycle_key)`, duplicate `(user_id, task_id, completion_ref)`, duplicate order/outbox business keys, and negative wallet/task quota updates.
- [x] `growth_tasks` has no delete column or foreign-key cascade that can erase accounting history.

## Verification Plan

- `cd backend && make generate`
- `go test -tags=unit ./ent/...`
- Execute `backend/sqlArchiving/growth/ddl/182_create_growth_tables.sql` twice in the configured MySQL 8/GoldenDB test database.
- `go test -tags=integration ./internal/repository/... -run Growth`

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|
| Generation | `make generate` | PASS: Ent and Wire generation completed. |
| Ent tests | `go test -tags=unit ./ent/...` | PASS across all generated Ent packages. |
| Static schema test | `go test -p=1 -gcflags=all='-N -l' -tags=unit ./ent/schema -run Growth -count=1` | PASS: 17 schemas, table annotations, unique keys, checks, and User edges verified. |
| DDL inventory | `backend/sqlArchiving/growth/ddl/182_create_growth_tables.sql` | Static inspection finds exactly 17 `CREATE TABLE IF NOT EXISTS growth_*` statements and all required unique keys/checks. |
| MySQL integration test | `GROWTH_MYSQL_TEST_DSN='root@tcp(127.0.0.1:3307)/?parseTime=true' go test -tags=integration ./internal/repository/growthmysqltest -run TestGrowthMySQLArchiveAndConstraints -count=1 -v` | PASS on MySQL 8.4.9: archive executed twice, 17 tables created, duplicate business keys and negative quotas rejected. |
| Generated paths | `backend/ent/growth*.go`, `backend/ent/growth*/`, `backend/ent/migrate/schema.go`, `backend/ent/user*.go` | Present and generated. |
| Test path | `backend/internal/repository/growthmysqltest/growth_mysql_schema_integration_test.go` | Creates and removes a disposable schema using the explicit test DSN. |

## Execution Notes

Real verification used a native local MySQL 8.4.9 test instance on `127.0.0.1:3307`; Docker Desktop's WSL engine remains unavailable on this host and was not required.
