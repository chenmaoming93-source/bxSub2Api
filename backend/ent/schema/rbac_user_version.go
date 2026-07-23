package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type RBACUserVersion struct{ ent.Schema }

func (RBACUserVersion) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "rbac_user_versions"}}
}
func (RBACUserVersion) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id"),
		field.Int64("authz_version").Default(1),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now).
			SchemaType(map[string]string{dialect.MySQL: "datetime(6)"}),
	}
}
func (RBACUserVersion) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("rbac_user_version").Field("user_id").Required().Unique(),
	}
}
func (RBACUserVersion) Indexes() []ent.Index {
	return []ent.Index{index.Fields("user_id").Unique()}
}
