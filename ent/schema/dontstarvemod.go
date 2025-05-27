package schema

import (
	"aurora-admin/internal/entity/dstety"
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"

	// "entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
	"time"
)

type DontStarveMod struct {
	ent.Schema
}

func (DontStarveMod) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").DefaultFunc(uuid.NewString).Unique(),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
		field.String("workshop_id").Unique(),
		field.String("name").Optional(),
		field.Int("time_created").Optional(),
		field.Int("time_updated").Optional(),
		field.String("image").Optional(),
		field.String("description").Optional(),
		field.String("author").Optional(),
		field.String("version").Optional(),
		field.Bool("downloaded").Optional(),
		field.JSON("config", []dstety.DontStarveModConfig{}).Optional(),
		field.String("raw_config").Optional(),
	}
}

func (DontStarveMod) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("archives", DontStarveArchive.Type).
			Ref("mods"),
	}
}
