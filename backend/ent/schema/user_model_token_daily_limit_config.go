package schema

import (
	"fmt"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"
)

// UserModelTokenDailyLimitConfig stores the daily token limit configuration per user and upstream model.
// Limits are stored independently from daily usage records so they survive day boundaries.
type UserModelTokenDailyLimitConfig struct {
	ent.Schema
}

func (UserModelTokenDailyLimitConfig) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "user_model_token_daily_limit_configs"}}
}

func (UserModelTokenDailyLimitConfig) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}}
}

func (UserModelTokenDailyLimitConfig) Fields() []ent.Field {
	nonNegative := func(v int64) error {
		if v < 0 {
			return fmt.Errorf("token value must be non-negative")
		}
		return nil
	}
	return []ent.Field{
		field.Int64("user_id"),
		field.String("model").MaxLen(255).NotEmpty(),
		field.Int64("daily_limit_tokens").Optional().Nillable().Validate(nonNegative),
	}
}

func (UserModelTokenDailyLimitConfig) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("user_model_token_daily_limit_configs").
			Field("user_id").
			Unique().
			Required(),
	}
}

func (UserModelTokenDailyLimitConfig) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "model").Unique(),
		index.Fields("user_id"),
	}
}
