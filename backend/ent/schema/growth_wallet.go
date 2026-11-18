package schema

import (
	"time"

	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// GrowthWallet is a user's spendable growth-point account.
type GrowthWallet struct {
	ent.Schema
}

func (GrowthWallet) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "growth_wallets"},
		entsql.Check("available_points >= 0 AND earned_total >= 0 AND spent_total >= 0"),
	}
}

func (GrowthWallet) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}}
}

func (GrowthWallet) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id"),
		field.Int64("available_points").Default(0).Validate(growthNonNegative("available_points")),
		field.Int64("earned_total").Default(0).Validate(growthNonNegative("earned_total")),
		field.Int64("spent_total").Default(0).Validate(growthNonNegative("spent_total")),
		field.Int64("version").Default(1).Validate(growthPositive("version")),
		field.Time("balance_updated_at").
			Default(time.Now).
			SchemaType(map[string]string{dialect.MySQL: "datetime(6)"}),
	}
}

func (GrowthWallet) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("growth_wallet").
			Field("user_id").
			Unique().
			Required().
			Annotations(entsql.OnDelete(entsql.Restrict)),
	}
}

func (GrowthWallet) Indexes() []ent.Index {
	return []ent.Index{index.Fields("user_id").Unique()}
}
