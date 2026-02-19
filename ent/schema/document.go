package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Document is the searchable unit.
// Classification: (year, depth1, depth2, title).
type Document struct {
	ent.Schema
}

func (Document) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.Int("year"),
		field.String("depth1").NotEmpty().MaxLen(80),
		field.String("depth2").NotEmpty().MaxLen(80),
		field.String("title").NotEmpty().MaxLen(300),
		field.Text("content").Optional(),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
		field.UUID("created_by_user_id", uuid.UUID{}),
	}
}

func (Document) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("created_by", User.Type).Ref("documents_created").Field("created_by_user_id").Unique().Required(),
		edge.To("aliases", DocumentAlias.Type),
		edge.To("embedding", DocumentEmbedding.Type).Unique(),
		edge.To("acls", DocumentACL.Type),
	}
}

func (Document) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("year", "depth1", "depth2", "title"),
		index.Fields("year"),
		index.Fields("depth1", "depth2"),
	}
}
