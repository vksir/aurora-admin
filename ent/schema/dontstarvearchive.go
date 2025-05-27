package schema

import (
	"aurora-admin/internal/entity/dstety"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// DontStarveArchive holds the schema definition for the DontStarveArchive entity.
type DontStarveArchive struct {
	ent.Schema
}

// Fields of the DontStarveArchive.
func (DontStarveArchive) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").DefaultFunc(uuid.NewString).Unique(),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
		field.String("remark").Optional(),
		field.String("cluster_name").Optional(),
		field.String("cluster_password").Optional(),
		field.String("cluster_description").Optional(),
		field.Int("max_players").Optional(),
		field.Bool("pvp").Optional(),
		field.JSON("world", []dstety.DontStarveWorld{}).Optional(),
	}
}

// Edges of the DontStarveArchive.
func (DontStarveArchive) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("mods", DontStarveMod.Type),
	}
}
