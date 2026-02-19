package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

type ACLEffect string

const (
	ACLEffectDeny  ACLEffect = "deny"
	ACLEffectRead  ACLEffect = "read"
	ACLEffectWrite ACLEffect = "read_write"
)

func (ACLEffect) Values() []string {
	return []string{string(ACLEffectDeny), string(ACLEffectRead), string(ACLEffectWrite)}
}

// DocumentACL is a group-based per-document rule.
// Precedence suggestion (to be implemented in service layer): deny > read_write > read.
type DocumentACL struct {
	ent.Schema
}

func (DocumentACL) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.UUID("document_id", uuid.UUID{}),
		field.UUID("group_id", uuid.UUID{}),
		field.Enum("effect").GoType(ACLEffect("")).Default(string(ACLEffectRead)),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (DocumentACL) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("document", Document.Type).Ref("acls").Field("document_id").Unique().Required(),
		edge.From("group", Group.Type).Ref("document_acls").Field("group_id").Unique().Required(),
	}
}

func (DocumentACL) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("document_id", "group_id").Unique(),
		index.Fields("group_id"),
	}
}
