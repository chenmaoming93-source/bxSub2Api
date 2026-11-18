package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// GrowthAdminAudit records privileged growth mutations.
type GrowthAdminAudit struct {
	ent.Schema
}

func (GrowthAdminAudit) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "growth_admin_audit"}}
}

func (GrowthAdminAudit) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}}
}

func (GrowthAdminAudit) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("operator_id"),
		field.String("action").MaxLen(64).NotEmpty(),
		field.String("resource_type").MaxLen(64).NotEmpty(),
		field.Int64("resource_id").Optional().Nillable(),
		field.JSON("before_json", map[string]any{}).Optional(),
		field.JSON("after_json", map[string]any{}).Optional(),
		field.String("reason").MaxLen(512).Default(""),
		field.String("request_id").MaxLen(128).Default(""),
	}
}

func (GrowthAdminAudit) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("operator", User.Type).
			Ref("growth_admin_audits").
			Field("operator_id").
			Unique().
			Required().
			Annotations(entsql.OnDelete(entsql.Restrict)),
	}
}

func (GrowthAdminAudit) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("operator_id", "created_at"),
		index.Fields("resource_type", "resource_id", "created_at"),
		index.Fields("action", "created_at"),
	}
}
