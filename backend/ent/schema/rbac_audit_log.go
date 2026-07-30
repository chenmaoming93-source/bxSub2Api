package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"time"
)

type RBACAuditLog struct{ ent.Schema }

func (RBACAuditLog) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "rbac_audit_logs"}}
}
func (RBACAuditLog) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("actor_user_id").Optional().Nillable(),
		field.String("action").MaxLen(64).NotEmpty(),
		field.String("target_type").MaxLen(32).NotEmpty(),
		field.String("target_id").MaxLen(128).NotEmpty(),
		field.JSON("before_data", map[string]any{}).Optional().SchemaType(map[string]string{dialect.MySQL: "json"}),
		field.JSON("after_data", map[string]any{}).Optional().SchemaType(map[string]string{dialect.MySQL: "json"}),
		field.String("request_id").MaxLen(128).Default(""),
		field.String("ip_address").MaxLen(64).Default(""),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}
func (RBACAuditLog) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("actor", User.Type).Ref("rbac_audit_logs").Field("actor_user_id").Unique(),
	}
}
func (RBACAuditLog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("actor_user_id", "created_at"),
		index.Fields("target_type", "target_id", "created_at"),
		index.Fields("created_at"),
	}
}
