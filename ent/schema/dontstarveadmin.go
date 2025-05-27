package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
	"time"
)

// DontStarveAdmin holds the schema definition for the DontStarveAdmin entity.
type DontStarveAdmin struct {
	ent.Schema
}

// Fields of the DontStarveAdmin.
func (DontStarveAdmin) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").DefaultFunc(uuid.NewString).Unique(),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
		field.String("name").Optional(),
		field.String("klei_id").Unique(),
	}
}

// Edges of the DontStarveAdmin.
func (DontStarveAdmin) Edges() []ent.Edge {
	return nil
}
