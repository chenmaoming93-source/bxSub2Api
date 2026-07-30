package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"
)

type RBACUserRole struct{ ent.Schema }

func (RBACUserRole) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "rbac_user_roles"}}
}
func (RBACUserRole) Mixin() []ent.Mixin { return []ent.Mixin{mixins.TimeMixin{}} }
func (RBACUserRole) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id"),
		field.Int64("role_id"),
		field.Int64("assigned_by").Optional().Nillable(),
	}
}
func (RBACUserRole) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("rbac_user_roles").Field("user_id").Required().Unique(),
		edge.From("role", RBACRole.Type).Ref("user_roles").Field("role_id").Required().Unique(),
		edge.From("assigner", User.Type).Ref("assigned_rbac_user_roles").Field("assigned_by").Unique(),
	}
}
func (RBACUserRole) Indexes() []ent.Index {
	return []ent.Index{index.Fields("user_id", "role_id").Unique(), index.Fields("role_id")}
}
