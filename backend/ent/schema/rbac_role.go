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

type RBACRole struct{ ent.Schema }

func (RBACRole) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "rbac_roles"}}
}
func (RBACRole) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}, mixins.SoftDeleteMixin{}}
}
func (RBACRole) Fields() []ent.Field {
	return []ent.Field{
		field.String("code").MaxLen(64).NotEmpty(),
		field.String("name").MaxLen(100).NotEmpty(),
		field.String("description").MaxLen(500).Default(""),
		field.Bool("is_system").Default(false),
		field.String("status").MaxLen(20).Default("active"),
	}
}
func (RBACRole) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("user_roles", RBACUserRole.Type),
		edge.To("role_permissions", RBACRolePermission.Type),
	}
}
func (RBACRole) Indexes() []ent.Index {
	return []ent.Index{index.Fields("code").Unique(), index.Fields("status"), index.Fields("deleted_at")}
}
