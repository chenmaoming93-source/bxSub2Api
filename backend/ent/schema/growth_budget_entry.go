package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// GrowthBudgetEntry is an append-only budget mutation audit row.
type GrowthBudgetEntry struct {
	ent.Schema
}

func (GrowthBudgetEntry) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "growth_budget_entries"}}
}

func (GrowthBudgetEntry) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}}
}

func (GrowthBudgetEntry) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("budget_id").Default(1),
		field.String("entry_type").MaxLen(16).Validate(growthEnum("entry_type", "TOPUP", "REDUCE", "ALLOCATE", "REVOKE")),
		field.Int64("points").Validate(growthPositive("points")),
		field.Int64("budget_after").Validate(growthNonNegative("budget_after")),
		field.Int64("allocated_after").Validate(growthNonNegative("allocated_after")),
		field.Int64("distributed_after").Default(0).Validate(growthNonNegative("distributed_after")),
		field.Int64("task_id").Optional().Nillable(),
		field.Int64("operator_id"),
		field.String("reason").MaxLen(512).NotEmpty(),
		field.String("request_id").MaxLen(128).Default(""),
	}
}

func (GrowthBudgetEntry) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("task_id"),
		index.Fields("entry_type", "created_at"),
		index.Fields("operator_id", "created_at"),
	}
}
