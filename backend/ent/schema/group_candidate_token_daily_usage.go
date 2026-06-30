package schema

import (
	"fmt"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"
)

// GroupCandidateTokenDailyUsage stores one quota window per route candidate and day.
type GroupCandidateTokenDailyUsage struct{ ent.Schema }

func (GroupCandidateTokenDailyUsage) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "group_candidate_token_daily_usages"}}
}

func (GroupCandidateTokenDailyUsage) Mixin() []ent.Mixin { return []ent.Mixin{mixins.TimeMixin{}} }

func (GroupCandidateTokenDailyUsage) Fields() []ent.Field {
	nonNegative := func(v int64) error {
		if v < 0 {
			return fmt.Errorf("token value must be non-negative")
		}
		return nil
	}
	return []ent.Field{
		field.Int64("group_id"),
		field.String("route_alias").MaxLen(255).NotEmpty(),
		field.String("upstream_model").MaxLen(255).NotEmpty(),
		field.Time("usage_date").SchemaType(map[string]string{dialect.MySQL: "date"}),
		field.Int64("used_tokens").Default(0).Validate(nonNegative),
		field.Int64("daily_limit_tokens").Optional().Nillable().Validate(nonNegative),
	}
}

func (GroupCandidateTokenDailyUsage) Edges() []ent.Edge {
	return []ent.Edge{edge.From("group", Group.Type).Ref("candidate_token_daily_usages").Field("group_id").Unique().Required()}
}

func (GroupCandidateTokenDailyUsage) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("group_id", "route_alias", "upstream_model", "usage_date").Unique(),
		index.Fields("group_id"),
	}
}
