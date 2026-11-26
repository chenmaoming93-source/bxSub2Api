package schema

import (
	"time"

	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// GrowthFulfillmentJob is the durable outbox for order fulfillment.
type GrowthFulfillmentJob struct {
	ent.Schema
}

func (GrowthFulfillmentJob) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "growth_fulfillment_jobs"}}
}

func (GrowthFulfillmentJob) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}}
}

func (GrowthFulfillmentJob) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("order_id"),
		field.String("job_type").MaxLen(32).Default("FULFILL"),
		field.String("job_status").MaxLen(20).Default("PENDING").Validate(growthEnum("job_status", "PENDING", "PROCESSING", "RETRY", "SUCCEEDED", "FAILED", "CANCELLED")),
		field.Int("attempt_count").Default(0).Validate(growthNonNegativeInt("attempt_count")),
		field.Int("max_attempts").Default(3).Validate(growthPositiveInt("max_attempts")),
		field.Time("next_retry_at").Default(time.Now).SchemaType(map[string]string{dialect.MySQL: "datetime(6)"}),
		field.Time("locked_until").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.MySQL: "datetime(6)"}),
		field.String("last_error").MaxLen(1024).Default(""),
		field.String("provider_reference").MaxLen(255).Default(""),
		field.Time("completed_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.MySQL: "datetime(6)"}),
	}
}

func (GrowthFulfillmentJob) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("order_id").Unique(),
		index.Fields("job_status", "next_retry_at"),
		index.Fields("locked_until"),
	}
}
