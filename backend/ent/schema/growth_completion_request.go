package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// GrowthCompletionRequest records a trusted completion declaration by AIWork.
type GrowthCompletionRequest struct {
	ent.Schema
}

func (GrowthCompletionRequest) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "growth_completion_requests"}}
}

func (GrowthCompletionRequest) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}}
}

func (GrowthCompletionRequest) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id"),
		field.Int64("task_id"),
		field.String("completion_ref").MaxLen(128).NotEmpty(),
		field.String("request_id").MaxLen(128).Default(""),
		field.String("result").MaxLen(32).Validate(growthEnum("result", "AWARDED", "ALREADY_COMPLETED", "NOT_ELIGIBLE")),
		field.Int64("transfer_id").Optional().Nillable(),
		field.Int64("awarded_points").Default(0).Validate(growthNonNegative("awarded_points")),
		field.Int("rule_version"),
		field.String("cycle_key").MaxLen(64),
	}
}

func (GrowthCompletionRequest) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("growth_completion_requests").
			Field("user_id").
			Unique().
			Required().
			Annotations(entsql.OnDelete(entsql.Restrict)),
	}
}

func (GrowthCompletionRequest) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "task_id", "completion_ref").Unique(),
		index.Fields("user_id", "created_at"),
		index.Fields("task_id", "cycle_key"),
		index.Fields("transfer_id"),
	}
}
