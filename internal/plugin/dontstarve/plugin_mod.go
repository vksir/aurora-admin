package dontstarve

import (
	"aurora-admin/internal/store"
	"context"
	"database/sql"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"

	"aurora-admin/pkg/log"
	"aurora-admin/pkg/util/errutil"
	"aurora-admin/pkg/steam"
)

func (p *Plugin) createModFromModOverrides(ctx context.Context, content []byte) ([]*store.Mod, error) {
	workshopID2Config, err := parseModLuaByRegex(string(content))
	if err != nil {
		return nil, errutil.Wrap(err)
	}

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
				mod.Name = info.Title
				mod.Description = info.Description
				mod.TimeCreated = int64(info.TimeCreated)
				mod.TimeUpdated = int64(info.TimeUpdated)
				mod.Image = info.PreviewUrl
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

func fetchModInfoFromAPI(workshopIDs []string) (map[string]steam.Publishedfiledetail, error) {
	if len(workshopIDs) == 0 {
		return make(map[string]steam.Publishedfiledetail), nil
	}

	log.Info("begin fetch mod info from API", "workshopIds", workshopIDs)
	resp, err := steam.GetPublishedFileDetails(workshopIDs...)
	if err != nil {
		return nil, errutil.Wrap(err)
	}
	if len(resp.Response.Publishedfiledetails) != len(workshopIDs) {
		log.Error("some mods not found",
			"expected", workshopIDs, "actual", resp.Response)
		return nil, fmt.Errorf("some mods not found")
	}

	workshopID2Info := make(map[string]steam.Publishedfiledetail, len(resp.Response.Publishedfiledetails))
	for i := range workshopIDs {
		workshopID2Info[workshopIDs[i]] = resp.Response.Publishedfiledetails[i]
	}
	return workshopID2Info, err
}

func parseModLuaByRegex(content string) (map[string]string, error) {
	result := make(map[string]string)

	// 匹配 ["workshop-XXXX"] = { 开始位置，支持单引号或双引号
	modEntryRegex := regexp.MustCompile(`\[["']workshop-(\d+)["']\]\s*=\s*\{`)
	enabledRegex := regexp.MustCompile(`\benabled\s*=\s*true`)

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

		block := strings.TrimSpace(content[match[0]:closeBracePos+1])

		if enabledRegex.MatchString(block) {
			result[workshopID] = block
		}
	}

	return result, nil
}

func genModOverrideContent(rawConfigs []string) string {
	content := "return {" + strings.Join(rawConfigs, ",\n  ") + "}"
	return content
}
