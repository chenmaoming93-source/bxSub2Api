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

type RBACPermission struct{ ent.Schema }

func (RBACPermission) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "rbac_permissions"}}
}
func (RBACPermission) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}, mixins.SoftDeleteMixin{}}
}
func (RBACPermission) Fields() []ent.Field {
	return []ent.Field{
		field.String("code").MaxLen(128).NotEmpty(),
		field.String("name").MaxLen(100).NotEmpty(),
		field.String("module").MaxLen(64).NotEmpty(),
		field.String("description").MaxLen(500).Default(""),
		field.String("risk_level").MaxLen(16).NotEmpty(),
		field.Bool("is_system").Default(true),
		field.String("status").MaxLen(20).Default("active"),
	}
}
func (RBACPermission) Edges() []ent.Edge {
	return []ent.Edge{edge.To("role_permissions", RBACRolePermission.Type)}
}
func (RBACPermission) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("code").Unique(),
		index.Fields("module"),
		index.Fields("status"),
		index.Fields("deleted_at"),
	}
}
