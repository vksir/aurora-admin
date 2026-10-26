package dontstarve

import (
	"aurora-admin/internal/store"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"

	"aurora-admin/pkg/log"
	"aurora-admin/pkg/util/errutil"
	"aurora-admin/pkg/steam"
)

// 预编译正则，避免每次解析 mod 配置时重复编译。
var (
	modEntryRegex     = regexp.MustCompile(`\[["']workshop-(\d+)["']\]\s*=\s*\{`)
	enabledRegex      = regexp.MustCompile(`\benabled\s*=\s*true`)
	returnPrefixRegex = regexp.MustCompile(`^return\b`)
	workshopIDRegex   = regexp.MustCompile(`^\d+$`)
)

// queryPublishedFileDetails 查询 Steam Workshop 模组详情；以变量形式注入，便于测试替换。
var queryPublishedFileDetails = steam.GetPublishedFileDetails

// 模组添加相关的错误，供 API 层区分客户端输入错误（400）与内部错误（500）。
var (
	ErrInvalidWorkshopID = errors.New("invalid workshop id, expect digits only")
	ErrModNotFound       = errors.New("workshop mod not found")
	ErrInvalidModConfig  = errors.New("invalid mod config")
	ErrNoEnabledMods     = errors.New("no enabled mods found in modoverrides.lua")
)

func (p *Plugin) createModFromModOverrides(ctx context.Context, content []byte) ([]*store.Mod, error) {
	workshopID2Config, err := parseModLuaByRegex(string(content))
	if err != nil {
		return nil, errutil.Wrap(err)
	}
	return p.importMods(ctx, workshopID2Config)
}

// importMods 将解析出的 workshopID -> 原始 Lua 配置批量 upsert 入库。
// Steam 元信息获取失败时仅记日志降级（无元信息入库），与存档导入行为一致。
func (p *Plugin) importMods(ctx context.Context, workshopID2Config map[string]string) ([]*store.Mod, error) {
	workshopID2Info, err := fetchModInfoFromAPI(slices.Collect(maps.Keys(workshopID2Config)))
	if err != nil {
		log.ErrorC(ctx, "failed to fetch mod info from API", "err", err)
	}

	var newMods []*store.Mod
	err = p.db.WithTx(ctx, func(ctx context.Context, _ *sql.Tx) error {
		for workshopID, modConfig := range workshopID2Config {
			mod, err := p.db.GetModByWorkshopID(ctx, workshopID)
			if err == sql.ErrNoRows {
				mod, err = p.db.CreateMod(ctx, &store.Mod{WorkshopID: workshopID})
				if err != nil {
					return errutil.Wrap(err)
				}
			} else if err != nil {
				return errutil.Wrap(err)
			}

			mod.RawConfig = modConfig
			info, ok := workshopID2Info[workshopID]
			if ok {
				applyModMetaInfo(mod, info)
			}

			mod, err = p.db.UpdateMod(ctx, mod)
			if err != nil {
				return errutil.Wrap(err)
			}
			newMods = append(newMods, mod)
		}
		return nil
	})
	if err != nil {
		return nil, errutil.Wrap(err)
	}
	return newMods, nil
}

// CreateMod 按创意工坊 ID 添加模组。
// rawConfig 为空时：新模组使用默认配置，已存在的模组保留原配置；
// 提供时支持完整配置块、整个 modoverrides.lua 文件或仅模组内部配置项三种形式。
func (p *Plugin) CreateMod(ctx context.Context, workshopID, rawConfig string) (*store.Mod, error) {
	workshopID = strings.TrimSpace(workshopID)
	if !workshopIDRegex.MatchString(workshopID) {
		return nil, fmt.Errorf("%w: %q", ErrInvalidWorkshopID, workshopID)
	}

	// 按单个 ID 添加前必须经 Workshop 校验，避免拼写错误静默入库
	workshopID2Info, err := fetchModInfoFromAPI([]string{workshopID})
	if err != nil {
		return nil, errutil.Wrap(err)
	}
	info, ok := workshopID2Info[workshopID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrModNotFound, workshopID)
	}

	var result *store.Mod
	err = p.db.WithTx(ctx, func(ctx context.Context, _ *sql.Tx) error {
		mod, err := p.db.GetModByWorkshopID(ctx, workshopID)
		if err == sql.ErrNoRows {
			rawConfig, err = normalizeModConfig(workshopID, rawConfig)
			if err != nil {
				return err
			}
			mod = &store.Mod{WorkshopID: workshopID, RawConfig: rawConfig}
			applyModMetaInfo(mod, info)
			mod, err = p.db.CreateMod(ctx, mod)
			if err != nil {
				return errutil.Wrap(err)
			}
			result = mod
			return nil
		}
		if err != nil {
			return errutil.Wrap(err)
		}

		// 已存在的模组：未提供配置时保留原配置
		if strings.TrimSpace(rawConfig) != "" {
			mod.RawConfig, err = normalizeModConfig(workshopID, rawConfig)
			if err != nil {
				return err
			}
		}
		applyModMetaInfo(mod, info)
		mod, err = p.db.UpdateMod(ctx, mod)
		if err != nil {
			return errutil.Wrap(err)
		}
		result = mod
		return nil
	})
	if err != nil {
		return nil, errutil.Wrap(err)
	}
	return result, nil
}

// UploadModOverrides 解析上传的 modoverrides.lua 内容，导入其中所有启用的模组。
func (p *Plugin) UploadModOverrides(ctx context.Context, content []byte) ([]*store.Mod, error) {
	workshopID2Config, err := parseModLuaByRegex(string(content))
	if err != nil {
		return nil, errutil.Wrap(err)
	}
	if len(workshopID2Config) == 0 {
		return nil, ErrNoEnabledMods
	}
	return p.importMods(ctx, workshopID2Config)
}

func (p *Plugin) UpdateMod(ctx context.Context, id string, req *store.Mod) (*store.Mod, error) {
	mod, err := p.db.GetMod(ctx, id)
	if err != nil {
		return nil, err
	}
	mod.RawConfig = req.RawConfig
	return p.db.UpdateMod(ctx, mod)
}

func (p *Plugin) DeleteMod(ctx context.Context, ids ...string) error {
	for _, id := range ids {
		mod, err := p.db.GetModWithArchives(ctx, id)
		if err == sql.ErrNoRows {
			continue
		} else if err != nil {
			return errutil.Wrap(err)
		}

		if len(mod.Archives) != 0 {
			var usedArchives []string
			for _, arch := range mod.Archives {
				usedArchives = append(usedArchives, arch.ID)
			}
			return fmt.Errorf("mod %s is used by archives: %v", id, usedArchives)
		}

		err = p.db.DeleteMod(ctx, mod.ID)
		if err != nil {
			return errutil.Wrap(err)
		}
	}
	return nil
}

func (p *Plugin) GetMod(ctx context.Context, id string) (*store.Mod, error) {
	mod, err := p.db.GetMod(ctx, id)
	if err != nil {
		return nil, errutil.Wrap(err)
	}
	return mod, nil
}

func (p *Plugin) ListMod(ctx context.Context) ([]*store.Mod, error) {
	mods, err := p.db.ListMods(ctx)
	if err != nil {
		return nil, errutil.Wrap(err)
	}
	return mods, nil
}

// applyModMetaInfo 用 Workshop 详情覆盖模组元信息（含 time_updated，作为自动更新检测的基准）。
// 区别于 modupdate.go 的 applyModInfo：那里刻意不写 time_updated（由更新成功后回写）。
func applyModMetaInfo(mod *store.Mod, info steam.Publishedfiledetail) {
	mod.Name = info.Title
	mod.Description = info.Description
	mod.Image = info.PreviewUrl
	mod.TimeCreated = int64(info.TimeCreated)
	mod.TimeUpdated = int64(info.TimeUpdated)
}

func fetchModInfoFromAPI(workshopIDs []string) (map[string]steam.Publishedfiledetail, error) {
	if len(workshopIDs) == 0 {
		return make(map[string]steam.Publishedfiledetail), nil
	}

	log.Info("begin fetch mod info from API", "workshopIds", workshopIDs)
	resp, err := queryPublishedFileDetails(workshopIDs...)
	if err != nil {
		return nil, errutil.Wrap(err)
	}

	// Steam 对不存在的 ID 也会返回条目（result != 1），按 publishedfileid 建映射并过滤
	workshopID2Info := make(map[string]steam.Publishedfiledetail, len(workshopIDs))
	for i := range resp.Response.Publishedfiledetails {
		detail := resp.Response.Publishedfiledetails[i]
		if detail.Result != 1 {
			log.Warn("query workshop item failed, skip",
				"workshop_id", detail.Publishedfileid, "result", detail.Result)
			continue
		}
		workshopID2Info[detail.Publishedfileid] = detail
	}
	return workshopID2Info, nil
}

// parseModLuaByRegex 解析 Lua 内容中所有 ["workshop-xxx"] = {...} 配置块，
// 只保留 enabled = true 的。
func parseModLuaByRegex(content string) (map[string]string, error) {
	result := parseModLuaBlocks(content)
	for workshopID, block := range result {
		if !enabledRegex.MatchString(block) {
			delete(result, workshopID)
		}
	}
	return result, nil
}

// parseModLuaBlocks 解析 Lua 内容中所有 ["workshop-xxx"] = {...} 配置块（不区分 enabled）。
func parseModLuaBlocks(content string) map[string]string {
	result := make(map[string]string)

	// 匹配 ["workshop-XXXX"] = { 开始位置，支持单引号或双引号
	matches := modEntryRegex.FindAllStringSubmatchIndex(content, -1)

	for _, match := range matches {
		workshopID := content[match[2]:match[3]]

		// match[1] 是完整匹配的结束位置，紧跟在 { 之后
		// 所以 { 的位置是 match[1] - 1
		openBracePos := match[1] - 1

		// 括号计数找到匹配的 }
		depth := 0
		closeBracePos := -1
		for i := openBracePos; i < len(content); i++ {
			if content[i] == '{' {
				depth++
			} else if content[i] == '}' {
				depth--
				if depth == 0 {
					closeBracePos = i
					break
				}
			}
		}

		if closeBracePos < 0 {
			continue
		}

		result[workshopID] = strings.TrimSpace(content[match[0]:closeBracePos+1])
	}

	return result
}

// defaultModConfig 生成模组默认配置：空 configuration_options（沿用模组自身默认值）并启用。
func defaultModConfig(workshopID string) string {
	return fmt.Sprintf(`["workshop-%s"]={ configuration_options={}, enabled=true }`, workshopID)
}

// normalizeModConfig 归一化用户提供的模组配置，返回完整的 ["workshop-<id>"]={...} 配置块。
// 支持三种输入：
//  1. 完整配置块或整个 modoverrides.lua 文件（可含多个模组，取 workshopID 对应项）；
//  2. 仅模组内部配置项（如 "configuration_options={...}, enabled=true"），自动包装键名，
//     缺少 enabled 时自动补 enabled=true；
//  3. 空串，返回默认配置。
func normalizeModConfig(workshopID, rawConfig string) (string, error) {
	rawConfig = strings.TrimSpace(rawConfig)
	if rawConfig == "" {
		return defaultModConfig(workshopID), nil
	}

	// 识别完整的 ["workshop-xxx"]={...} 配置块（含 enabled=false 的，尊重用户原样）
	blocks := parseModLuaBlocks(rawConfig)
	if len(blocks) > 0 {
		block, ok := blocks[workshopID]
		if !ok {
			return "", fmt.Errorf("%w: config does not belong to workshop mod %s",
				ErrInvalidModConfig, workshopID)
		}
		return block, nil
	}

	// 未识别到 workshop 键：视为模组内部配置项，剥离 return {...} / {...} 外壳后包装键名
	inner := stripLuaTableShell(rawConfig)
	if !strings.Contains(inner, "=") {
		return "", fmt.Errorf("%w: %q", ErrInvalidModConfig, rawConfig)
	}
	if !enabledRegex.MatchString(inner) {
		inner = strings.TrimSpace(inner) + ", enabled=true"
	}
	return fmt.Sprintf(`["workshop-%s"]={ %s }`, workshopID, inner), nil
}

// stripLuaTableShell 剥离 Lua 表外壳（return 前缀与包裹整个内容的最外层花括号）。
func stripLuaTableShell(s string) string {
	s = strings.TrimSpace(s)
	if returnPrefixRegex.MatchString(s) {
		s = strings.TrimSpace(returnPrefixRegex.ReplaceAllString(s, ""))
	}
	if !strings.HasPrefix(s, "{") || !strings.HasSuffix(s, "}") {
		return s
	}

	// 仅当首 { 与尾 } 配对（确实包裹整个内容）时才剥离
	depth := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 && i != len(s)-1 {
				return s
			}
		}
	}
	if depth == 0 {
		return strings.TrimSpace(s[1 : len(s)-1])
	}
	return s
}

func genModOverrideContent(rawConfigs []string) string {
	content := "return {" + strings.Join(rawConfigs, ",\n  ") + "}"
	return content
}
