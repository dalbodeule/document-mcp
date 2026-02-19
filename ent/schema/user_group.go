package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// UserGroup represents a membership relation.
type UserGroup struct {
	ent.Schema
}

func (UserGroup) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.UUID("user_id", uuid.UUID{}),
		field.UUID("group_id", uuid.UUID{}),
	}
}

func (UserGroup) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("user_groups").Field("user_id").Unique().Required(),
		edge.From("group", Group.Type).Ref("user_groups").Field("group_id").Unique().Required(),
	}
}

func (UserGroup) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "group_id").Unique(),
	}
}
