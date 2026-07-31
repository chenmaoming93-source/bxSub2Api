package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"
)

type TokenStatProjectionMetric struct{ ent.Schema }

func (TokenStatProjectionMetric) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "token_stat_projection_metrics"}}
}

func (TokenStatProjectionMetric) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}}
}

func (TokenStatProjectionMetric) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("projection_id").Positive(),
		field.String("metric_code").MaxLen(64).NotEmpty(),
		field.String("status").MaxLen(20).NotEmpty().Default("ACTIVE"),
		field.Time("enabled_at").Optional().Nillable().SchemaType(map[string]string{dialect.MySQL: "datetime(6)"}),
		field.Time("disabled_at").Optional().Nillable().SchemaType(map[string]string{dialect.MySQL: "datetime(6)"}),
	}
}

func (TokenStatProjectionMetric) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("projection_id", "metric_code").Unique(),
		index.Fields("metric_code", "status"),
	}
}
