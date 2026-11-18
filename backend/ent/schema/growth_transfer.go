package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// GrowthTransfer is the idempotent header for a balanced point transfer.
type GrowthTransfer struct {
	ent.Schema
}

func (GrowthTransfer) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "growth_transfers"}}
}

func (GrowthTransfer) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}}
}

func (GrowthTransfer) Fields() []ent.Field {
	return []ent.Field{
		field.String("biz_type").MaxLen(64).NotEmpty(),
		field.String("idempotency_key").MaxLen(128).NotEmpty(),
		field.String("from_account_type").MaxLen(32).Validate(growthEnum("from_account_type", "TASK", "WALLET", "REDEEMED", "SYSTEM")),
		field.String("to_account_type").MaxLen(32).Validate(growthEnum("to_account_type", "TASK", "WALLET", "REDEEMED", "SYSTEM")),
		field.Int64("from_user_id").Optional().Nillable(),
		field.Int64("to_user_id").Optional().Nillable(),
		field.Int64("task_id").Optional().Nillable(),
		field.Int64("order_id").Optional().Nillable(),
		field.Int64("points").Validate(growthPositive("points")),
		field.String("status").MaxLen(20).Default("PENDING").Validate(growthEnum("status", "PENDING", "SUCCEEDED", "FAILED", "REVERSED")),
		field.String("failure_reason").MaxLen(512).Default(""),
		field.Time("completed_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.MySQL: "datetime(6)"}),
	}
}

func (GrowthTransfer) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("biz_type", "idempotency_key").Unique(),
		index.Fields("from_user_id", "created_at"),
		index.Fields("to_user_id", "created_at"),
		index.Fields("task_id"),
		index.Fields("order_id"),
		index.Fields("status"),
	}
}
