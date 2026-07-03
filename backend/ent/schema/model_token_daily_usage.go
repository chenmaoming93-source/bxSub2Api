package schema

import (
	"fmt"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"
)

// ModelTokenDailyUsage stores one global token quota window per upstream model and day.
type ModelTokenDailyUsage struct {
	ent.Schema
}

func (ModelTokenDailyUsage) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "model_token_daily_usages"}}
}

func (ModelTokenDailyUsage) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}}
}

func (ModelTokenDailyUsage) Fields() []ent.Field {
	nonNegative := func(v int64) error {
		if v < 0 {
			return fmt.Errorf("token value must be non-negative")
		}
		return nil
	}
	return []ent.Field{
		field.String("model").MaxLen(255).NotEmpty(),
		field.Time("usage_date").SchemaType(map[string]string{dialect.MySQL: "date"}),
		field.Int64("used_tokens").Default(0).Validate(nonNegative),
	}
}

func (ModelTokenDailyUsage) Indexes() []ent.Index {
	return []ent.Index{index.Fields("model", "usage_date").Unique()}
}
