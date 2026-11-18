package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// GrowthUserTask tracks one user's progress within an active task cycle.
type GrowthUserTask struct {
	ent.Schema
}

func (GrowthUserTask) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "growth_user_tasks"}}
}

func (GrowthUserTask) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}}
}

func (GrowthUserTask) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id"),
		field.Int64("task_id"),
		field.Int64("rule_id"),
		field.String("cycle_key").MaxLen(64),
		field.Int64("progress").Default(0).Validate(growthNonNegative("progress")),
		field.Int64("target").Default(1).Validate(growthPositive("target")),
		field.Int64("completed_count").Default(0).Validate(growthNonNegative("completed_count")),
		field.String("status").MaxLen(32).Default("IN_PROGRESS"),
		field.Time("completed_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.MySQL: "datetime(6)"}),
		field.Time("rewarded_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.MySQL: "datetime(6)"}),
	}
}

func (GrowthUserTask) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("growth_user_tasks").
			Field("user_id").
			Unique().
			Required().
			Annotations(entsql.OnDelete(entsql.Restrict)),
	}
}

func (GrowthUserTask) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "task_id", "rule_id", "cycle_key").Unique(),
		index.Fields("user_id", "created_at"),
		index.Fields("task_id", "cycle_key"),
	}
}
