package schema

import (
	"fmt"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"
)

// UserModelTokenDailyUsage stores one token quota window per user, upstream model and day.
type UserModelTokenDailyUsage struct {
	ent.Schema
}

func (UserModelTokenDailyUsage) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "user_model_token_daily_usages"}}
}

func (UserModelTokenDailyUsage) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}}
}

func (UserModelTokenDailyUsage) Fields() []ent.Field {
	nonNegative := func(v int64) error {
		if v < 0 {
			return fmt.Errorf("token value must be non-negative")
		}
		return nil
	}
	return []ent.Field{
		field.Int64("user_id"),
		field.String("model").MaxLen(255).NotEmpty(),
		field.Time("usage_date").SchemaType(map[string]string{dialect.MySQL: "date"}),
		field.Int64("used_tokens").Default(0).Validate(nonNegative),
		field.Int64("daily_limit_tokens").Optional().Nillable().Validate(nonNegative),
	}
}

func (UserModelTokenDailyUsage) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("model_token_daily_usages").
			Field("user_id").
			Unique().
			Required(),
	}
}

func (UserModelTokenDailyUsage) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "model", "usage_date").Unique(),
		index.Fields("user_id"),
	}
}
