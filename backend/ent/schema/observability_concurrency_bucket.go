package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type ObservabilityConcurrencyBucket struct {
	ent.Schema
}

func (ObservabilityConcurrencyBucket) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "observability_concurrency_bucket"}}
}

func (ObservabilityConcurrencyBucket) Fields() []ent.Field {
	return []ent.Field{
		field.Int("window_seconds"),
		field.Time("bucket_start").SchemaType(map[string]string{dialect.MySQL: "datetime(6)"}),
		field.String("sample_kind").MaxLen(16),
		field.String("dimension_hash").MaxLen(64),
		field.Int64("account_id"),
		field.Int64("group_id").Default(0),
		field.String("route_alias").MaxLen(255).Optional().Nillable(),
		field.Int("current_peak").Default(0),
		field.Int64("current_sum").Default(0),
		field.Int64("sample_count").Default(0),
		field.Int("effective_limit").Default(0),
		field.Int("waiting_peak").Default(0),
		field.Int("peak_ratio_actual").Default(0),
		field.Int("peak_ratio_limit").Default(0),
		field.Time("peak_ratio_sample_at").Optional().Nillable().SchemaType(map[string]string{dialect.MySQL: "datetime(6)"}),
		field.Int64("wait_sum_ms").Default(0),
		field.Int64("wait_sample_count").Default(0),
		field.Time("first_sampled_at").SchemaType(map[string]string{dialect.MySQL: "datetime(6)"}),
		field.Time("last_sampled_at").SchemaType(map[string]string{dialect.MySQL: "datetime(6)"}),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now).SchemaType(map[string]string{dialect.MySQL: "datetime(6)"}),
	}
}

func (ObservabilityConcurrencyBucket) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("window_seconds", "bucket_start", "dimension_hash").Unique(),
		index.Fields("window_seconds", "bucket_start"),
		index.Fields("group_id", "window_seconds", "route_alias", "bucket_start"),
		index.Fields("account_id", "window_seconds", "bucket_start"),
	}
}
