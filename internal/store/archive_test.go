package store

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func newTestDB(t testing.TB) *DB {
	t.Helper()
	db, err := OpenMemory()
	assert.Nil(t, err)
	assert.Nil(t, db.Migrate(context.Background()))
	return db
}

func TestListArchivesJoinMods(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	// 创建两个 mod
	m1, err := db.CreateMod(ctx, &Mod{WorkshopID: "111"})
	assert.Nil(t, err)
	m2, err := db.CreateMod(ctx, &Mod{WorkshopID: "222"})
	assert.Nil(t, err)

	// 存档 A：关联 2 个 mod；存档 B：无 mod
	archA, err := db.CreateArchive(ctx, &Archive{
		Remark: "A", ClusterName: "ca",
		World: nil,
		Mods:  []*Mod{m1, m2},
	})
	assert.Nil(t, err)
	archB, err := db.CreateArchive(ctx, &Archive{Remark: "B"})
	assert.Nil(t, err)

	list, err := db.ListArchives(ctx)
	assert.Nil(t, err)
	assert.Len(t, list, 2)

	byID := map[string]*Archive{}
	for _, a := range list {
		byID[a.ID] = a
	}
	assert.Contains(t, byID, archA.ID)
	assert.Contains(t, byID, archB.ID)
	assert.Len(t, byID[archA.ID].Mods, 2)
	assert.Empty(t, byID[archB.ID].Mods)

	workshopIDs := map[string]bool{}
	for _, m := range byID[archA.ID].Mods {
		workshopIDs[m.WorkshopID] = true
	}
	assert.True(t, workshopIDs["111"] && workshopIDs["222"])
}

func TestOpenFileDBWithWAL(t *testing.T) {
	path := t.TempDir() + "/test.db"
	db, err := Open(path)
	assert.Nil(t, err)
	defer db.Close()

	assert.Nil(t, db.Migrate(context.Background()))

	ctx := context.Background()
	_, err = db.CreateArchive(ctx, &Archive{Remark: "wal"})
	assert.Nil(t, err)

	// WAL 模式生效检查
	var mode string
	err = db.QueryRow(context.Background(), `PRAGMA journal_mode`).Scan(&mode)
	assert.Nil(t, err)
	assert.Equal(t, "wal", mode)

	list, err := db.ListArchives(ctx)
	assert.Nil(t, err)
	assert.Len(t, list, 1)
}
