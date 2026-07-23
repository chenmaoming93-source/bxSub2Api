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

type RBACRolePermission struct{ ent.Schema }

func (RBACRolePermission) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "rbac_role_permissions"}}
}
func (RBACRolePermission) Mixin() []ent.Mixin { return []ent.Mixin{mixins.TimeMixin{}} }
func (RBACRolePermission) Fields() []ent.Field {
	return []ent.Field{field.Int64("role_id"), field.Int64("permission_id")}
}
func (RBACRolePermission) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("role", RBACRole.Type).Ref("role_permissions").Field("role_id").Required().Unique(),
		edge.From("permission", RBACPermission.Type).Ref("role_permissions").Field("permission_id").Required().Unique(),
	}
}
func (RBACRolePermission) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("role_id", "permission_id").Unique(),
		index.Fields("permission_id"),
	}
}
