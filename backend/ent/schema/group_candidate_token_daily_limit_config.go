package schema

import (
	"fmt"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"
)

// GroupCandidateTokenDailyLimitConfig stores the daily token limit configuration per group routing candidate.
// Limits are stored independently from daily usage records so they survive day boundaries.
type GroupCandidateTokenDailyLimitConfig struct{ ent.Schema }

func (GroupCandidateTokenDailyLimitConfig) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "group_candidate_token_daily_limit_configs"}}
}

func (GroupCandidateTokenDailyLimitConfig) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}}
}

func (GroupCandidateTokenDailyLimitConfig) Fields() []ent.Field {
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
		field.Int64("daily_limit_tokens").Optional().Nillable().Validate(nonNegative),
	}
}

func (GroupCandidateTokenDailyLimitConfig) Edges() []ent.Edge {
	return []ent.Edge{edge.From("group", Group.Type).Ref("group_candidate_token_daily_limit_configs").Field("group_id").Unique().Required()}
}

func (GroupCandidateTokenDailyLimitConfig) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("group_id", "route_alias", "upstream_model").Unique(),
		index.Fields("group_id"),
	}
}
