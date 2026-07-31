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

type TokenStatQuotaRule struct{ ent.Schema }

func (TokenStatQuotaRule) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "token_stat_quota_rules"}}
}

func (TokenStatQuotaRule) Mixin() []ent.Mixin { return []ent.Mixin{mixins.TimeMixin{}} }

func (TokenStatQuotaRule) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").MaxLen(128).NotEmpty(),
		field.Int64("projection_id").Positive(),
		field.Bytes("dimension_hash").SchemaType(map[string]string{dialect.MySQL: "binary(16)"}),
		field.JSON("dimension_values", map[string]any{}).SchemaType(map[string]string{dialect.MySQL: "json"}),
		field.String("metric_code").MaxLen(64).NotEmpty(),
		field.String("period_type").MaxLen(1).NotEmpty(),
		field.Int64("limit_value").Positive().Validate(func(v int64) error {
			if v <= 0 {
				return fmt.Errorf("limit_value must be positive")
			}
			return nil
		}),
		field.String("enforcement_mode").MaxLen(20).NotEmpty().Default("REJECT"),
		field.String("status").MaxLen(20).NotEmpty().Default("DISABLED"),
		field.Time("effective_from").Optional().Nillable().SchemaType(map[string]string{dialect.MySQL: "datetime(6)"}),
		field.Time("effective_until").Optional().Nillable().SchemaType(map[string]string{dialect.MySQL: "datetime(6)"}),
		field.Uint64("created_by"),
	}
}

func (TokenStatQuotaRule) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("projection_id", "metric_code", "period_type", "status"),
		index.Fields("dimension_hash"),
	}
}
