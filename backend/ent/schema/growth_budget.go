package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// GrowthBudget is the single-row authoritative growth budget account.
type GrowthBudget struct {
	ent.Schema
}

func (GrowthBudget) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "growth_budgets"},
		entsql.Check("total_amount >= 0 AND allocated_amount >= 0 AND distributed_amount >= 0 AND allocated_amount <= total_amount AND distributed_amount <= allocated_amount"),
	}
}

func (GrowthBudget) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}}
}

func (GrowthBudget) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("total_amount").Default(0).Validate(growthNonNegative("total_amount")),
		field.Int64("allocated_amount").Default(0).Validate(growthNonNegative("allocated_amount")),
		field.Int64("distributed_amount").Default(0).Validate(growthNonNegative("distributed_amount")),
		field.Int64("version").Default(1).Validate(growthPositive("version")),
	}
}
