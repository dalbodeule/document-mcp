package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// DocumentAlias is an additional keyword for a document (many per document).
type DocumentAlias struct {
	ent.Schema
}

func (DocumentAlias) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.String("alias").NotEmpty().MaxLen(120),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.UUID("document_id", uuid.UUID{}),
	}
}

func (DocumentAlias) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("document", Document.Type).Ref("aliases").Field("document_id").Unique().Required(),
	}
}

func (DocumentAlias) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("alias"),
		index.Fields("document_id"),
		index.Fields("document_id", "alias").Unique(),
	}
}
