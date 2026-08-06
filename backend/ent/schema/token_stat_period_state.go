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

type TokenStatPeriodState struct{ ent.Schema }

func (TokenStatPeriodState) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "token_stat_period_states"}}
}

func (TokenStatPeriodState) Mixin() []ent.Mixin { return []ent.Mixin{mixins.TimeMixin{}} }

func (TokenStatPeriodState) Fields() []ent.Field {
	return []ent.Field{
		field.String("period_type").MaxLen(1).NotEmpty(),
		field.Time("period_start").SchemaType(map[string]string{dialect.MySQL: "datetime(6)"}),
		field.Time("period_end").SchemaType(map[string]string{dialect.MySQL: "datetime(6)"}),
		field.String("state").MaxLen(20).NotEmpty().Default("OPEN"),
		field.Int64("final_sync_version").Default(0),
		field.Time("closed_at").Optional().Nillable().SchemaType(map[string]string{dialect.MySQL: "datetime(6)"}),
		field.Time("persisted_at").Optional().Nillable().SchemaType(map[string]string{dialect.MySQL: "datetime(6)"}),
		field.Time("deleted_at").Optional().Nillable().SchemaType(map[string]string{dialect.MySQL: "datetime(6)"}),
		field.String("last_error").Optional().Nillable().SchemaType(map[string]string{dialect.MySQL: "text"}),
	}
}

func (TokenStatPeriodState) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("period_type", "period_start").Unique(),
		index.Fields("state", "period_end"),
	}
}
