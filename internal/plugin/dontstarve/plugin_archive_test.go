package dontstarve

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"aurora-admin/internal/store"
	"aurora-admin/pkg/util/fileutil"
	"github.com/stretchr/testify/assert"
)

// useTempHome 把 fileutil.Home 指向临时目录，使 clusterRoot 落在测试目录下。
// clusterRoot 依赖 ~/.klei/DoNotStarveTogether，测试不能碰真实的家目录。
func useTempHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	oldHome := fileutil.Home
	fileutil.Home = home
	t.Cleanup(func() { fileutil.Home = oldHome })
	return home
}

// writeModOverrides 写入存档目录下的 Master/modoverrides.lua
func writeModOverrides(t *testing.T, home string, archID string, content string) string {
	t.Helper()
	dir := filepath.Join(home, ".klei/DoNotStarveTogether", archID, ShardMaster)
	assert.Nil(t, os.MkdirAll(dir, 0o755))
	path := filepath.Join(dir, "modoverrides.lua")
	assert.Nil(t, fileutil.Write(path, []byte(content)))
	return filepath.Join(home, ".klei/DoNotStarveTogether", archID)
}

// mustGetOrCreateMod 复用已存在的模组（内存 DB 在同一进程的测试间共享）
func mustGetOrCreateMod(t *testing.T, p *Plugin, workshopID string) *store.Mod {
	t.Helper()
	ctx := context.Background()
	mod, err := p.db.GetModByWorkshopID(ctx, workshopID)
	if err == sql.ErrNoRows {
		mod, err = p.db.CreateMod(ctx, &store.Mod{WorkshopID: workshopID})
	}
	assert.Nil(t, err)
	return mod
}

// 旧存档没有 archive_mods 关联时，GetArchive 应从磁盘的 modoverrides.lua 回填，
// 否则前端会把已启用的模组显示为全部禁用。
func TestGetArchiveBackfillsModsFromDisk(t *testing.T) {
	p := newTestPlugin(t)
	ctx := context.Background()
	home := useTempHome(t)

	arch, err := p.db.CreateArchive(ctx, &store.Archive{Remark: "legacy"})
	assert.Nil(t, err)

	root := writeModOverrides(t, home, arch.ID, mockModOverridesLua)

	got, err := p.GetArchive(ctx, arch.ID)
	assert.Nil(t, err)
	assert.Len(t, got.Mods, 2)
	for _, mod := range got.Mods {
		assert.NotEmpty(t, mod.ID)
		assert.NotEmpty(t, mod.RawConfig, "回填时应带上磁盘上的模组配置")
	}

	// 关联已落库：删掉磁盘目录后仍应能读到模组
	assert.Nil(t, fileutil.Remove(root))
	again, err := p.GetArchive(ctx, arch.ID)
	assert.Nil(t, err)
	assert.Len(t, again.Mods, 2)
}

// 已有模组关联的存档不应被磁盘内容覆盖（用户可能在面板里改过关联）。
func TestGetArchiveKeepsExistingMods(t *testing.T) {
	p := newTestPlugin(t)
	ctx := context.Background()
	home := useTempHome(t)

	arch, err := p.db.CreateArchive(ctx, &store.Archive{Remark: "current"})
	assert.Nil(t, err)

	// 先写入一个模组关联（模拟面板中保存过）
	mod := mustGetOrCreateMod(t, p, "1392778117")
	assert.Nil(t, p.db.AddArchiveMods(ctx, arch.ID, []string{mod.ID}))

	// 磁盘上有两个启用的模组
	writeModOverrides(t, home, arch.ID, mockModOverridesLua)

	got, err := p.GetArchive(ctx, arch.ID)
	assert.Nil(t, err)
	assert.Len(t, got.Mods, 1)
	assert.Equal(t, "1392778117", got.Mods[0].WorkshopID)
}

// 存档目录不存在（例如只在数据库里有记录）时不应报错，也不应写入关联。
func TestGetArchiveWithoutClusterDir(t *testing.T) {
	p := newTestPlugin(t)
	ctx := context.Background()
	useTempHome(t)

	arch, err := p.db.CreateArchive(ctx, &store.Archive{Remark: "db-only"})
	assert.Nil(t, err)

	got, err := p.GetArchive(ctx, arch.ID)
	assert.Nil(t, err)
	assert.Empty(t, got.Mods)
}
