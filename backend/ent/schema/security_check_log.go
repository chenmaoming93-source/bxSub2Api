package schema

import (
	"github.com/Wei-Shaw/sub2api/internal/domain"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// SecurityCheckLog stores an asynchronously collected request safety decision.
type SecurityCheckLog struct {
	ent.Schema
}

func (SecurityCheckLog) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "security_check_logs"}}
}

func (SecurityCheckLog) Fields() []ent.Field {
	return []ent.Field{
		field.String("event_id").MaxLen(64),
		field.String("request_id").MaxLen(64).Optional().Nillable(),
		field.String("client_request_id").MaxLen(64).Optional().Nillable(),
		field.Int64("user_id").Optional().Nillable(),
		field.Int64("api_key_id").Optional().Nillable(),
		field.String("api_key_name").MaxLen(100).Optional().Nillable(),
		field.Int64("group_id").Optional().Nillable(),
		field.String("group_name").MaxLen(100).Optional().Nillable(),
		field.String("model").MaxLen(100).Optional().Nillable(),
		field.String("provider").MaxLen(50).Optional().Nillable(),
		field.String("protocol").MaxLen(32).Optional().Nillable(),
		field.String("endpoint").MaxLen(255).Optional().Nillable(),
		field.Int64("config_version").Default(1),
		field.JSON("rules_snapshot", []domain.SecurityCheckRule{}).Default([]domain.SecurityCheckRule{}),
		field.Bytes("request_body").Optional(),
		field.Int64("request_body_original_bytes").Default(0),
		field.Int64("request_body_stored_bytes").Default(0),
		field.Bool("request_body_truncated").Default(false),
		field.String("singguard_response").Optional().Nillable().SchemaType(map[string]string{"mysql": "mediumtext"}),
		field.String("check_status").MaxLen(16),
		field.String("decision").MaxLen(16),
		field.Bool("is_unsafe").Default(false),
		field.JSON("triggered_rules", []domain.SecurityCheckTriggeredRule{}).Default([]domain.SecurityCheckTriggeredRule{}),
		field.Int("latency_ms").Optional().Nillable(),
		field.Int("singguard_latency_ms").Optional().Nillable(),
		field.Int("queue_delay_ms").Optional().Nillable(),
		field.String("exception_type").MaxLen(32).Optional().Nillable(),
		field.String("exception_message").Optional().Nillable(),
		field.Time("created_at").Immutable(),
	}
}

func (SecurityCheckLog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("event_id").Unique(),
		index.Fields("created_at"),
		index.Fields("group_id", "created_at"),
		index.Fields("decision", "created_at"),
	}
}
