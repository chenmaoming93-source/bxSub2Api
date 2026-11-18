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

// GrowthTask is the durable task catalog entry. It is never deleted.
type GrowthTask struct {
	ent.Schema
}

func (GrowthTask) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "growth_tasks"},
		entsql.Check("allocated_points >= 0 AND distributed_points >= 0 AND distributed_points <= allocated_points"),
	}
}

func (GrowthTask) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}}
}

func (GrowthTask) Fields() []ent.Field {
	return []ent.Field{
		field.String("task_code").MaxLen(64).NotEmpty(),
		field.String("title").MaxLen(255).NotEmpty(),
		field.String("description").MaxLen(2048).Default(""),
		field.String("category").MaxLen(32).Validate(growthEnum("category", "CHECKIN", "DAILY", "ACHIEVEMENT")),
		field.String("task_type").MaxLen(32).Validate(growthEnum("task_type", "CHECKIN", "DAILY", "ACHIEVEMENT")),
		field.String("icon").MaxLen(128).Default(""),
		field.Int("sort_order").Default(0),
		field.String("status").MaxLen(20).Default("DRAFT").Validate(growthEnum("status", "DRAFT", "ACTIVE", "OFFLINE")),
		field.Time("effective_from").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.MySQL: "datetime(6)"}),
		field.Time("effective_to").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.MySQL: "datetime(6)"}),
		field.Int64("allocated_points").Default(0).Validate(growthNonNegative("allocated_points")),
		field.Int64("distributed_points").Default(0).Validate(growthNonNegative("distributed_points")),
	}
}

func (GrowthTask) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("task_code").Unique(),
		index.Fields("status", "sort_order"),
		index.Fields("category", "status"),
	}
}
