// Package ent provides the generated ORM code for database entities.
package ent

// Enable sql/execquery for raw SQL passthrough and sql/lock for FOR UPDATE row locks.
//go:generate go run -mod=mod entgo.io/ent/cmd/ent generate --feature sql/upsert,intercept,sql/execquery,sql/lock --idtype int64 ./schema
