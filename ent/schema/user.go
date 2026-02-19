package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// User represents an authenticated principal.
type User struct {
	ent.Schema
}

func (User) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.String("email").NotEmpty().MaxLen(320),
		// Store password hash as TEXT to avoid length issues when changing algorithms.
		field.Text("password_hash").Sensitive().NotEmpty(),
		field.String("name").NotEmpty().MaxLen(100),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("user_groups", UserGroup.Type),
		edge.To("documents_created", Document.Type),
		edge.To("refresh_tokens", RefreshToken.Type),
	}
}

func (User) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("email").Unique(),
	}
}
