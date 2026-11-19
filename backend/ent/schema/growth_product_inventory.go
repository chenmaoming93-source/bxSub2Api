package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// GrowthProductInventory holds stock and optional Token-budget reservation.
type GrowthProductInventory struct {
	ent.Schema
}

func (GrowthProductInventory) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "growth_product_inventory"}}
}

func (GrowthProductInventory) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}}
}

func (GrowthProductInventory) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("product_id"),
		field.String("stock_mode").MaxLen(20).Default("LIMITED").Validate(growthEnum("stock_mode", "LIMITED", "UNLIMITED")),
		field.Int64("available").Default(0).Validate(growthNonNegative("available")),
		field.Int64("reserved").Default(0).Validate(growthNonNegative("reserved")),
		field.Int64("sold").Default(0).Validate(growthNonNegative("sold")),
		field.Int64("lock_version").Default(1).Validate(growthPositive("lock_version")),
		field.Int64("token_quota_total").Default(0).Validate(growthNonNegative("token_quota_total")),
		field.Int64("token_quota_distributed").Default(0).Validate(growthNonNegative("token_quota_distributed")),
	}
}

func (GrowthProductInventory) Indexes() []ent.Index {
	return []ent.Index{index.Fields("product_id").Unique()}
}
