package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// GrowthRewardClaim is the final business idempotency record for an award.
type GrowthRewardClaim struct {
	ent.Schema
}

func (GrowthRewardClaim) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "growth_reward_claims"}}
}

func (GrowthRewardClaim) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}}
}

func (GrowthRewardClaim) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id"),
		field.Int64("task_id"),
		field.Int64("rule_id"),
		field.String("cycle_key").MaxLen(64),
		field.Int("sequence").Default(1).Validate(growthPositiveInt("sequence")),
		field.String("idempotency_key").MaxLen(128).NotEmpty(),
		field.Int64("transfer_id"),
		field.Int64("points").Validate(growthPositive("points")),
		field.String("claim_status").MaxLen(20).Default("SUCCEEDED"),
		field.Time("claimed_at").SchemaType(map[string]string{dialect.MySQL: "datetime(6)"}),
	}
}

func (GrowthRewardClaim) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("growth_reward_claims").
			Field("user_id").
			Unique().
			Required().
			Annotations(entsql.OnDelete(entsql.Restrict)),
	}
}

func (GrowthRewardClaim) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "idempotency_key").Unique(),
		index.Fields("user_id", "task_id", "cycle_key", "sequence").Unique(),
		index.Fields("transfer_id"),
		index.Fields("user_id", "claimed_at"),
	}
}
