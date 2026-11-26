package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// GrowthInventoryLedger is an append-only inventory change record.
type GrowthInventoryLedger struct {
	ent.Schema
}

func (GrowthInventoryLedger) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "growth_inventory_ledger"}}
}

func (GrowthInventoryLedger) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}}
}

func (GrowthInventoryLedger) Fields() []ent.Field {
	return []ent.Field{
		field.String("business_key").MaxLen(128).NotEmpty(),
		field.Int64("product_id"),
		field.Int64("inventory_id"),
		field.String("operation_type").MaxLen(32).Validate(growthEnum("operation_type", "STOCK_IN", "RESERVE", "SELL", "RELEASE", "ADJUST")),
		field.Int64("delta_available").Default(0),
		field.Int64("delta_reserved").Default(0),
		field.Int64("delta_sold").Default(0),
		field.Int64("available_after").Validate(growthNonNegative("available_after")),
		field.Int64("reserved_after").Validate(growthNonNegative("reserved_after")),
		field.Int64("sold_after").Validate(growthNonNegative("sold_after")),
		field.Int64("order_id").Optional().Nillable(),
		field.Int64("operator_id").Optional().Nillable(),
		field.String("reason").MaxLen(512).Default(""),
	}
}

func (GrowthInventoryLedger) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("business_key").Unique(),
		index.Fields("product_id", "created_at"),
		index.Fields("order_id"),
	}
}
