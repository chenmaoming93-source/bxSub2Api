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

// GrowthProductVersion is an immutable price and benefit snapshot.
type GrowthProductVersion struct {
	ent.Schema
}

func (GrowthProductVersion) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "growth_product_versions"}}
}

func (GrowthProductVersion) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}}
}

func (GrowthProductVersion) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("product_id"),
		field.Int("version").Validate(growthPositiveInt("version")),
		field.Int64("points_cost").Validate(growthPositive("points_cost")),
		field.String("benefit_summary").MaxLen(512).Default(""),
		field.Text("benefit_detail").Optional(),
		field.Int64("token_amount").Default(0).Validate(growthNonNegative("token_amount")),
		field.Int("per_user_limit").Default(0).Validate(growthNonNegativeInt("per_user_limit")),
		field.Int64("created_by"),
		field.Time("published_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.MySQL: "datetime(6)"}),
		field.Time("effective_from").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.MySQL: "datetime(6)"}),
		field.Time("effective_to").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.MySQL: "datetime(6)"}),
	}
}

func (GrowthProductVersion) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("product_id", "version").Unique(),
		index.Fields("product_id", "published_at"),
	}
}
