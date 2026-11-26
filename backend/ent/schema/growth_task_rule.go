package schema

import (
	"fmt"

	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// GrowthTaskRule is an immutable published rule version.
type GrowthTaskRule struct {
	ent.Schema
}

func (GrowthTaskRule) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "growth_task_rules"}}
}

func (GrowthTaskRule) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}}
}

func (GrowthTaskRule) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("task_id"),
		field.Int("version").Validate(growthPositiveInt("version")),
		field.String("trigger_event").MaxLen(64).Default("COMPLETION"),
		field.JSON("conditions", map[string]any{}).Optional(),
		field.Int64("target").Default(1).Validate(growthPositive("target")),
		field.Int64("reward_points").Validate(growthPositive("reward_points")),
		field.String("period_type").MaxLen(20).Validate(growthEnum("period_type", "DAILY", "ONCE", "WEEKLY")),
		field.Int("daily_limit").Default(0).Validate(func(value int) error {
			if value < 0 {
				return fmt.Errorf("daily_limit must be non-negative")
			}
			return nil
		}),
		field.Int("per_user_limit").Default(0).Validate(func(value int) error {
			if value < 0 {
				return fmt.Errorf("per_user_limit must be non-negative")
			}
			return nil
		}),
		field.Int64("created_by"),
		field.Time("effective_from").SchemaType(map[string]string{dialect.MySQL: "datetime(6)"}),
		field.Time("effective_to").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.MySQL: "datetime(6)"}),
		field.Time("published_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.MySQL: "datetime(6)"}),
	}
}

func (GrowthTaskRule) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("task_id", "version").Unique(),
		index.Fields("task_id", "effective_from", "effective_to"),
		index.Fields("published_at"),
	}
}
