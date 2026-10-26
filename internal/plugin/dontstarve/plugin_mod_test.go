package dontstarve

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"

	"aurora-admin/internal/store"
	"aurora-admin/pkg/steam"
	"github.com/stretchr/testify/assert"
)

const mockModOverridesLua = `return {
  ["workshop-1392778117"]={
    configuration_options={
      Language="chinese",
      LilyBushSpacing=1,
      ["UI相关"]=false
    },
    enabled=true 
  },
  ["workshop-1991746508"]={ configuration_options={ Language="A", ShowBuff=true }, enabled=true }
}`

func TestParseAndGenModOverrides(t *testing.T) {
	workshopID2Config, err := parseModLuaByRegex(mockModOverridesLua)
	assert.Nil(t, err)
	assert.Len(t, workshopID2Config, 2)
	for id, block := range workshopID2Config {
		assert.True(t, strings.HasPrefix(block, `["workshop-`+id+`"]`), block)
		assert.True(t, enabledRegex.MatchString(block), block)
	}

	res := genModOverrideContent(slices.Collect(maps.Values(workshopID2Config)))
	fmt.Println(res)
}

// newTestPlugin 构造仅带内存 DB 的 Plugin，用于模组导入相关单测。
func newTestPlugin(t testing.TB) *Plugin {
	t.Helper()
	db, err := store.OpenMemory()
	assert.Nil(t, err)
	assert.Nil(t, db.Migrate(context.Background()))
	return &Plugin{db: db}
}

// fakeQueryPublishedFileDetails 按预置详情应答 Workshop 查询；
// 未预置的 ID 返回 result=9（不存在），与 Steam 实际行为一致。
func fakeQueryPublishedFileDetails(details ...steam.Publishedfiledetail) func(...string) (steam.GetPublishedFileDetailsResponse, error) {
	return func(ids ...string) (steam.GetPublishedFileDetailsResponse, error) {
		var resp steam.GetPublishedFileDetailsResponse
		for _, id := range ids {
			matched := steam.Publishedfiledetail{Publishedfileid: id, Result: 9}
			for _, d := range details {
				if d.Publishedfileid == id {
					matched = d
					break
				}
			}
			resp.Response.Publishedfiledetails = append(resp.Response.Publishedfiledetails, matched)
		}
		return resp, nil
	}
}

// withFakeSteamDetails 在测试期间替换 Workshop 查询为假实现。
func withFakeSteamDetails(t *testing.T, details ...steam.Publishedfiledetail) {
	t.Helper()
	orig := queryPublishedFileDetails
	t.Cleanup(func() { queryPublishedFileDetails = orig })
	queryPublishedFileDetails = fakeQueryPublishedFileDetails(details...)
}

func TestNormalizeModConfig(t *testing.T) {
	const id = "1392778117"
	defaultCfg := `["workshop-1392778117"]={ configuration_options={}, enabled=true }`

	// 空配置 → 默认配置
	cfg, err := normalizeModConfig(id, "")
	assert.Nil(t, err)
	assert.Equal(t, defaultCfg, cfg)

	cfg, err = normalizeModConfig(id, "  \n ")
	assert.Nil(t, err)
	assert.Equal(t, defaultCfg, cfg)

	// 完整配置块 → 原样保留
	full := `["workshop-1392778117"]={ configuration_options={ Language="chinese" }, enabled=true }`
	cfg, err = normalizeModConfig(id, full)
	assert.Nil(t, err)
	assert.Equal(t, full, cfg)

	// enabled=false 的完整块 → 尊重用户原样
	disabled := `["workshop-1392778117"]={ configuration_options={}, enabled=false }`
	cfg, err = normalizeModConfig(id, disabled)
	assert.Nil(t, err)
	assert.Equal(t, disabled, cfg)

	// 整个 modoverrides.lua 文件（含多个模组）→ 提取对应项
	cfg, err = normalizeModConfig(id, mockModOverridesLua)
	assert.Nil(t, err)
	assert.Equal(t, parseModLuaBlocks(mockModOverridesLua)[id], cfg)
	assert.Contains(t, cfg, `["workshop-1392778117"]`)

	// 配置属于其他模组 → 报错
	_, err = normalizeModConfig("123456", mockModOverridesLua)
	assert.ErrorIs(t, err, ErrInvalidModConfig)

	// 仅模组内部配置项 → 自动包装键名
	cfg, err = normalizeModConfig(id, `configuration_options={ Language="chinese" }, enabled=true`)
	assert.Nil(t, err)
	assert.Equal(t, `["workshop-1392778117"]={ configuration_options={ Language="chinese" }, enabled=true }`, cfg)

	// 内部配置项缺 enabled → 自动补 enabled=true
	cfg, err = normalizeModConfig(id, `configuration_options={ Language="chinese" }`)
	assert.Nil(t, err)
	assert.Equal(t, `["workshop-1392778117"]={ configuration_options={ Language="chinese" }, enabled=true }`, cfg)

	// 带 return {...} 外壳的内部配置项
	cfg, err = normalizeModConfig(id, "return {\n  configuration_options={ Language=\"chinese\" }, enabled=true\n}")
	assert.Nil(t, err)
	assert.Equal(t, `["workshop-1392778117"]={ configuration_options={ Language="chinese" }, enabled=true }`, cfg)

	// 带 {...} 外壳的内部配置项
	cfg, err = normalizeModConfig(id, `{ configuration_options={ Language="chinese" }, enabled=true }`)
	assert.Nil(t, err)
	assert.Equal(t, `["workshop-1392778117"]={ configuration_options={ Language="chinese" }, enabled=true }`, cfg)

	// 垃圾输入 → 报错
	_, err = normalizeModConfig(id, "hello world")
	assert.ErrorIs(t, err, ErrInvalidModConfig)
}

func TestStripLuaTableShell(t *testing.T) {
	assert.Equal(t, `a=1`, stripLuaTableShell("return { a=1 }"))
	assert.Equal(t, `a=1`, stripLuaTableShell("{ a=1 }"))
	assert.Equal(t, `a=1`, stripLuaTableShell("  a=1  "))
	// 首 { 与尾 } 不配对（并非包裹整个内容）→ 不剥离
	assert.Equal(t, `{ a=1 }, { b=2 }`, stripLuaTableShell(`{ a=1 }, { b=2 }`))
	// 括号不配对 → 原样返回
	assert.Equal(t, `{ a=1`, stripLuaTableShell(`{ a=1`))
	// return 单词不应被误剥（returner）
	assert.Equal(t, `returner`, stripLuaTableShell(`returner`))
}

func TestCreateMod(t *testing.T) {
	const id = "1392778117"
	withFakeSteamDetails(t, steam.Publishedfiledetail{
		Publishedfileid: id, Result: 1, Title: "Chinese Language Pack",
		Description: "desc", PreviewUrl: "https://example.com/p.png",
		TimeCreated: 100, TimeUpdated: 200,
	})

	p := newTestPlugin(t)
	ctx := context.Background()

	// 新建：默认配置 + Workshop 元信息
	mod, err := p.CreateMod(ctx, id, "")
	assert.Nil(t, err)
	assert.Equal(t, id, mod.WorkshopID)
	assert.Equal(t, "Chinese Language Pack", mod.Name)
	assert.Equal(t, "https://example.com/p.png", mod.Image)
	assert.Equal(t, int64(200), mod.TimeUpdated)
	assert.Equal(t, `["workshop-1392778117"]={ configuration_options={}, enabled=true }`, mod.RawConfig)

	// 已存在且未提供配置 → 保留原配置
	again, err := p.CreateMod(ctx, id, "")
	assert.Nil(t, err)
	assert.Equal(t, mod.ID, again.ID)
	assert.Equal(t, mod.RawConfig, again.RawConfig)

	// 已存在且提供配置 → 覆盖
	updated, err := p.CreateMod(ctx, id, `configuration_options={ Language="english" }`)
	assert.Nil(t, err)
	assert.Equal(t, `["workshop-1392778117"]={ configuration_options={ Language="english" }, enabled=true }`, updated.RawConfig)

	// Workshop 上不存在 → 报错
	_, err = p.CreateMod(ctx, "404404404", "")
	assert.ErrorIs(t, err, ErrModNotFound)

	// 非法 ID → 报错
	_, err = p.CreateMod(ctx, "abc", "")
	assert.ErrorIs(t, err, ErrInvalidWorkshopID)
	_, err = p.CreateMod(ctx, "workshop-123", "")
	assert.ErrorIs(t, err, ErrInvalidWorkshopID)

	// 提供的配置属于其他模组 → 报错
	_, err = p.CreateMod(ctx, id, `["workshop-666"]={ enabled=true }`)
	assert.ErrorIs(t, err, ErrInvalidModConfig)
}

func TestUploadModOverrides(t *testing.T) {
	withFakeSteamDetails(t, detail("1392778117", 200), detail("1991746508", 300))

	p := newTestPlugin(t)
	ctx := context.Background()

	mods, err := p.UploadModOverrides(ctx, []byte(mockModOverridesLua))
	assert.Nil(t, err)
	assert.Len(t, mods, 2)

	// 重复导入 → upsert，不产生重复记录
	mods, err = p.UploadModOverrides(ctx, []byte(mockModOverridesLua))
	assert.Nil(t, err)
	assert.Len(t, mods, 2)
	list, err := p.db.ListMods(ctx)
	assert.Nil(t, err)
	assert.Len(t, list, 2)

	// 元信息已补全
	byWorkshopID := map[string]*store.Mod{}
	for _, m := range list {
		byWorkshopID[m.WorkshopID] = m
	}
	assert.Equal(t, "mod-1392778117", byWorkshopID["1392778117"].Name)
	assert.Equal(t, int64(200), byWorkshopID["1392778117"].TimeUpdated)
	assert.Equal(t, "mod-1991746508", byWorkshopID["1991746508"].Name)

	// 无启用的模组 → 报错
	_, err = p.UploadModOverrides(ctx, []byte(`return { ["workshop-123"]={ enabled=false } }`))
	assert.ErrorIs(t, err, ErrNoEnabledMods)

	// 不包含模组的文件 → 报错
	_, err = p.UploadModOverrides(ctx, []byte("return {}"))
	assert.ErrorIs(t, err, ErrNoEnabledMods)
}
