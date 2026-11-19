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

// GrowthExchangeOrder is a product purchase with an immutable version snapshot.
type GrowthExchangeOrder struct {
	ent.Schema
}

func (GrowthExchangeOrder) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "growth_exchange_orders"}}
}

func (GrowthExchangeOrder) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}}
}

func (GrowthExchangeOrder) Fields() []ent.Field {
	return []ent.Field{
		field.String("order_no").MaxLen(64).NotEmpty(),
		field.Int64("user_id"),
		field.Int64("product_id"),
		field.Int("product_version").Validate(growthPositiveInt("product_version")),
		field.Text("product_snapshot").Optional(),
		field.Int64("points_cost").Validate(growthPositive("points_cost")),
		field.String("order_status").MaxLen(20).Default("CREATED").Validate(growthEnum("order_status", "CREATED", "FULFILLING", "SUCCEEDED", "FAILED", "REFUNDED")),
		field.String("idempotency_key").MaxLen(128).NotEmpty(),
		field.Int("expected_version").Optional().Nillable(),
		field.Int64("transfer_id").Optional().Nillable(),
		field.Int64("refund_transfer_id").Optional().Nillable(),
		field.String("failure_reason").MaxLen(1024).Default(""),
		field.Time("fulfilled_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.MySQL: "datetime(6)"}),
		field.Time("refunded_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.MySQL: "datetime(6)"}),
	}
}

func (GrowthExchangeOrder) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("growth_exchange_orders").
			Field("user_id").
			Unique().
			Required().
			Annotations(entsql.OnDelete(entsql.Restrict)),
	}
}

func (GrowthExchangeOrder) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("order_no").Unique(),
		index.Fields("user_id", "idempotency_key").Unique(),
		index.Fields("user_id", "created_at"),
		index.Fields("product_id", "order_status"),
		index.Fields("order_status", "created_at"),
	}
}
