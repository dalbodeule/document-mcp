package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// RefreshToken stores a hashed refresh token for rotation/revocation.
type RefreshToken struct {
	ent.Schema
}

func (RefreshToken) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.UUID("user_id", uuid.UUID{}),
		field.String("token_hash").NotEmpty().Sensitive(),
		field.Time("expires_at"),
		field.Time("revoked_at").Optional().Nillable(),
		field.String("user_agent").Optional().MaxLen(300),
		field.String("ip").Optional().MaxLen(100),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

func (RefreshToken) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("refresh_tokens").Field("user_id").Unique().Required(),
	}
}

func (RefreshToken) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("token_hash").Unique(),
		index.Fields("user_id"),
		index.Fields("expires_at"),
	}
}
