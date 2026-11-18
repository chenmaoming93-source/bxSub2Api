package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// GrowthProduct is the durable exchange-product catalog entry.
type GrowthProduct struct {
	ent.Schema
}

func (GrowthProduct) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "growth_products"}}
}

func (GrowthProduct) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}}
}

func (GrowthProduct) Fields() []ent.Field {
	return []ent.Field{
		field.String("product_code").MaxLen(64).NotEmpty(),
		field.String("title").MaxLen(255).NotEmpty(),
		field.String("product_type").MaxLen(32).Validate(growthEnum("product_type", "TOKEN_QUOTA", "SUBSCRIPTION", "PHYSICAL")),
		field.String("icon").MaxLen(128).Default(""),
		field.String("fulfillment_mode").MaxLen(32).Validate(growthEnum("fulfillment_mode", "AUTOMATIC", "MANUAL")),
		field.Int("sort_order").Default(0),
		field.String("status").MaxLen(20).Default("DRAFT").Validate(growthEnum("status", "DRAFT", "ACTIVE", "PAUSED", "OFFLINE")),
		field.Int("current_version").Default(0).Validate(growthNonNegativeInt("current_version")),
	}
}

func (GrowthProduct) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("product_code").Unique(),
		index.Fields("status", "sort_order"),
		index.Fields("product_type", "status"),
	}
}
