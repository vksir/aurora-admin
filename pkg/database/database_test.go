package database

import (
	"aurora-admin/ent/dontstarvearchive"
	"aurora-admin/internal/entity/dstety"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestUpdateEdges(t *testing.T) {
	client := Init("")

	mod1 := client.DontStarveMod.
		Create().
		SetWorkshopID("mod_id1").
		SaveX(t.Context())
	mod2 := client.DontStarveMod.
		Create().
		SetWorkshopID("mod_id2").
		SaveX(t.Context())
	originArch := client.DontStarveArchive.
		Create().
		SetClusterName("cluster1").
		SetWorld([]dstety.DontStarveWorld{
			{
				Type: dstety.WorldTypeMaster,
			},
		}).
		AddMods(mod1).
		SaveX(t.Context())

	queryArch1 := client.DontStarveArchive.
		Query().
		Where(dontstarvearchive.ID(originArch.ID)).
		WithMods().
		OnlyX(t.Context())
	assert.Len(t, queryArch1.Edges.Mods, 1)
	assert.Equal(t, mod1.ID, queryArch1.Edges.Mods[0].ID)
	assert.Equal(t, "mod_id1", queryArch1.Edges.Mods[0].WorkshopID)

	// 替换 Edges，被替换的 Mod 条目不会删除
	updateArch2 := client.DontStarveArchive.
		UpdateOneID(originArch.ID).
		ClearMods().
		AddMods(mod2).
		SaveX(t.Context())
	queryArch2 := client.DontStarveArchive.
		Query().
		Where(dontstarvearchive.ID(originArch.ID)).
		WithMods().
		OnlyX(t.Context())
	queryMod1 := client.DontStarveMod.GetX(t.Context(), mod1.ID)
	assert.Len(t, updateArch2.Edges.Mods, 0)
	assert.Len(t, queryArch2.Edges.Mods, 1)
	assert.Equal(t, mod2.ID, queryArch2.Edges.Mods[0].ID)
	assert.Equal(t, "mod_id1", queryMod1.WorkshopID)
	assert.Equal(t, "mod_id2", queryArch2.Edges.Mods[0].WorkshopID)
}
