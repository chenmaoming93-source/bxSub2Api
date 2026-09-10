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

type TokenStatAggregate struct{ ent.Schema }

func (TokenStatAggregate) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "token_stat_aggregates"}}
}

func (TokenStatAggregate) Mixin() []ent.Mixin { return []ent.Mixin{mixins.TimeMixin{}} }

func (TokenStatAggregate) Fields() []ent.Field {
	nonNegative := func(v int64) error {
		if v < 0 {
			return fmt.Errorf("value must be non-negative")
		}
		return nil
	}
	return []ent.Field{
		field.String("period_type").MaxLen(1).NotEmpty(),
		field.Time("period_start").SchemaType(map[string]string{dialect.MySQL: "datetime(6)"}),
		field.Time("period_end").SchemaType(map[string]string{dialect.MySQL: "datetime(6)"}),
		field.Int64("projection_id").Positive(),
		field.Bytes("dimension_hash").SchemaType(map[string]string{dialect.MySQL: "binary(16)"}),
		field.JSON("dimension_values", map[string]any{}).SchemaType(map[string]string{dialect.MySQL: "json"}),
		field.String("metric_code").MaxLen(64).NotEmpty(),
		field.Int64("metric_value").Default(0).Validate(nonNegative),
		field.Int64("source_version").Default(0).Validate(nonNegative),
		field.Int64("user_id").Optional().Nillable(),
		field.Int64("api_key_id").Optional().Nillable(),
		field.Int64("group_id").Optional().Nillable(),
		field.String("route_alias").MaxLen(255).Optional().Nillable(),
		field.Int64("account_id").Optional().Nillable(),
		field.String("upstream_model").MaxLen(255).Optional().Nillable(),
		field.String("department").MaxLen(255).Optional().Nillable(),
		field.Time("last_synced_at").SchemaType(map[string]string{dialect.MySQL: "datetime(6)"}),
	}
}

func (TokenStatAggregate) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("period_type", "period_start", "projection_id", "dimension_hash", "metric_code").Unique(),
		index.Fields("projection_id", "metric_code", "period_type", "period_start"),
		index.Fields("user_id"),
		index.Fields("api_key_id"),
		index.Fields("group_id"),
		index.Fields("account_id"),
		index.Fields("department"),
	}
}
