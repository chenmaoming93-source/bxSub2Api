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

// ObservabilityRequestBucket stores idempotently recomputable request facts for
// observability. window_seconds is 60, 300 or 3600; source facts are never
// scanned for 7-day trends.
type ObservabilityRequestBucket struct {
	ent.Schema
}

func (ObservabilityRequestBucket) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "observability_request_bucket"}}
}

func (ObservabilityRequestBucket) Fields() []ent.Field {
	return []ent.Field{
		field.Int("window_seconds"),
		field.Time("bucket_start").
			SchemaType(map[string]string{dialect.MySQL: "datetime(6)"}),
		field.String("dimension_hash").MaxLen(64),
		field.Int64("user_id").Default(0),
		field.Int64("api_key_id").Default(0),
		field.Int64("group_id").Default(0),
		field.String("route_alias").MaxLen(255).Optional().Nillable(),
		field.Int64("account_id").Default(0),
		field.Int64("request_count").Default(0),
		field.Int64("success_count").Default(0),
		field.Int64("error_count").Default(0),
		field.Int64("sla_error_count").Default(0),
		field.Int64("business_limited_count").Default(0),
		field.Int64("rate_limited_count").Default(0),
		field.Int64("legacy_inferred_count").Default(0),
		field.Int64("input_tokens").Default(0),
		field.Int64("output_tokens").Default(0),
		field.Int64("cache_creation_tokens").Default(0),
		field.Int64("cache_read_tokens").Default(0),
		field.Int64("total_tokens").Default(0),
		field.Int64("duration_sample_count").Default(0),
		field.Int64("sum_duration_ms").Default(0),
		field.Int64("min_duration_ms").Default(0),
		field.Int64("max_duration_ms").Default(0),
		field.Int64("duration_le_100").Default(0),
		field.Int64("duration_le_250").Default(0),
		field.Int64("duration_le_500").Default(0),
		field.Int64("duration_le_1000").Default(0),
		field.Int64("duration_le_2000").Default(0),
		field.Int64("duration_le_5000").Default(0),
		field.Int64("duration_le_10000").Default(0),
		field.Int64("duration_gt_10000").Default(0),
		field.Int64("ttft_sample_count").Default(0),
		field.Int64("sum_ttft_ms").Default(0),
		field.Int64("min_ttft_ms").Default(0),
		field.Int64("max_ttft_ms").Default(0),
		field.Int64("ttft_le_100").Default(0),
		field.Int64("ttft_le_250").Default(0),
		field.Int64("ttft_le_500").Default(0),
		field.Int64("ttft_le_1000").Default(0),
		field.Int64("ttft_le_2000").Default(0),
		field.Int64("ttft_le_5000").Default(0),
		field.Int64("ttft_le_10000").Default(0),
		field.Int64("ttft_gt_10000").Default(0),
		field.Time("last_fact_at").SchemaType(map[string]string{dialect.MySQL: "datetime(6)"}),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			SchemaType(map[string]string{dialect.MySQL: "datetime(6)"}),
	}
}

func (ObservabilityRequestBucket) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("window_seconds", "bucket_start", "dimension_hash").Unique(),
		index.Fields("window_seconds", "bucket_start"),
		index.Fields("group_id", "window_seconds", "route_alias", "bucket_start"),
		index.Fields("user_id", "window_seconds", "bucket_start"),
		index.Fields("api_key_id", "window_seconds", "bucket_start"),
		index.Fields("account_id", "window_seconds", "bucket_start"),
	}
}
