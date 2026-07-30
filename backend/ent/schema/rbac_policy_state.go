package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"time"
)

type RBACPolicyState struct{ ent.Schema }

func (RBACPolicyState) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "rbac_policy_state"}}
}
func (RBACPolicyState) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("policy_version").Default(1),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}
