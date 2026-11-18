package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// GrowthAccountLedger is an immutable double-entry ledger row.
type GrowthAccountLedger struct {
	ent.Schema
}

func (GrowthAccountLedger) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "growth_account_ledger"},
		entsql.Check("points > 0 AND (balance_after IS NULL OR balance_after >= 0)"),
	}
}

func (GrowthAccountLedger) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}}
}

func (GrowthAccountLedger) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("transfer_id"),
		field.Int64("user_id").Optional().Nillable(),
		field.String("account_type").MaxLen(32).Validate(growthEnum("account_type", "TASK", "WALLET", "REDEEMED", "SYSTEM")),
		field.String("direction").MaxLen(8).Validate(growthEnum("direction", "IN", "OUT")),
		field.Int64("points").Validate(growthPositive("points")),
		field.Int64("balance_after").Optional().Nillable(),
		field.String("biz_type").MaxLen(64).NotEmpty(),
		field.String("title").MaxLen(255).NotEmpty(),
		field.Int64("reference_id").Optional().Nillable(),
	}
}

func (GrowthAccountLedger) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("transfer_id"),
		index.Fields("user_id", "created_at", "id"),
		index.Fields("biz_type", "created_at"),
	}
}
