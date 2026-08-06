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

type TokenStatProjection struct{ ent.Schema }

func (TokenStatProjection) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "token_stat_projections"}}
}

func (TokenStatProjection) Mixin() []ent.Mixin { return []ent.Mixin{mixins.TimeMixin{}} }

func (TokenStatProjection) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").MaxLen(128).NotEmpty(),
		field.JSON("dimension_codes", []string{}).SchemaType(map[string]string{dialect.MySQL: "json"}),
		field.String("dimension_signature").MaxLen(512).NotEmpty().Unique(),
		field.String("status").MaxLen(20).NotEmpty().Default("DRAFT"),
		field.Uint64("config_version").Default(0),
		field.Time("published_at").Optional().Nillable().SchemaType(map[string]string{dialect.MySQL: "datetime(6)"}),
		field.Time("enabled_at").Optional().Nillable().SchemaType(map[string]string{dialect.MySQL: "datetime(6)"}),
		field.Time("disabled_at").Optional().Nillable().SchemaType(map[string]string{dialect.MySQL: "datetime(6)"}),
		field.Uint64("created_by"),
	}
}

func (TokenStatProjection) Indexes() []ent.Index {
	return []ent.Index{index.Fields("status")}
}
