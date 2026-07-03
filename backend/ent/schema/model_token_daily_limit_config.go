package schema

import (
	"fmt"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"
)

// ModelTokenDailyLimitConfig stores the daily token limit configuration per upstream model.
// Limits are stored independently from daily usage records so they survive day boundaries.
type ModelTokenDailyLimitConfig struct {
	ent.Schema
}

func (ModelTokenDailyLimitConfig) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "model_token_daily_limit_configs"}}
}

func (ModelTokenDailyLimitConfig) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}}
}

func (ModelTokenDailyLimitConfig) Fields() []ent.Field {
	nonNegative := func(v int64) error {
		if v < 0 {
			return fmt.Errorf("token value must be non-negative")
		}
		return nil
	}
	return []ent.Field{
		field.String("model").MaxLen(255).NotEmpty(),
		field.Int64("daily_limit_tokens").Optional().Nillable().Validate(nonNegative),
	}
}

func (ModelTokenDailyLimitConfig) Indexes() []ent.Index {
	return []ent.Index{index.Fields("model").Unique()}
}
