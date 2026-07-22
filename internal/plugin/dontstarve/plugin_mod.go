package dontstarve

import (
	"aurora-admin/internal/entity/dstety"
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
	lua "github.com/yuin/gopher-lua"
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

func fetchModInfoFromLua(content []byte, mod *dstety.DontStarveMod) error {
	L := lua.NewState()
	defer L.Close()
	err := L.DoString(string(content))
	if err != nil {
		return errutil.Wrap(err)
	}

	description := L.GetGlobal("description")
	if description.Type() == lua.LTString {
		mod.Description = description.String()
	}
	author := L.GetGlobal("author")
	if author.Type() == lua.LTString {
		mod.Author = author.String()
	}
	version := L.GetGlobal("version")
	if version.Type() == lua.LTString {
		mod.Version = version.String()
	}

	mod.Config = make([]dstety.DontStarveModConfig, 0)
	configs := L.GetGlobal("configuration_options")
	if configs.Type() == lua.LTTable {
		configs.(*lua.LTable).ForEach(func(_, configLV lua.LValue) {
			if configLV.Type() == lua.LTTable {
				configTable := configLV.(*lua.LTable)

				var config dstety.DontStarveModConfig
				name := configTable.RawGetString("name")
				if name.Type() == lua.LTString {
					config.Name = name.String()
				}
				label := configTable.RawGetString("label")
				if label.Type() == lua.LTString {
					config.Label = label.String()
				}
				hover := configTable.RawGetString("hover")
				if hover.Type() == lua.LTString {
					config.Hover = hover.String()
				}

				options := configTable.RawGetString("options")
				if options.Type() == lua.LTTable {
					options.(*lua.LTable).ForEach(func(_, optionLV lua.LValue) {
						if optionLV.Type() == lua.LTTable {
							optionTable := optionLV.(*lua.LTable)

							var option dstety.DontStarveModConfigOption
							opDescription := optionTable.RawGetString("description")
							if opDescription.Type() == lua.LTString {
								option.Description = opDescription.String()
							}
							opHover := optionTable.RawGetString("hover")
							if opHover.Type() == lua.LTString {
								option.Hover = opHover.String()
							}
							data := optionTable.RawGetString("data")
							option.LuaValue = dstety.NewLuaValue(data)

							config.Option = append(config.Option, option)
						}
					})
				}

				// 查找 default 值，如果找不到，则将第一个选项设置为 default 值
				defaultV := configTable.RawGetString("default")
				defaultLuaValue := dstety.NewLuaValue(defaultV)
				hasFoundDefault := false
				for i := range config.Option {
					if config.Option[i].LuaValue == defaultLuaValue {
						config.Option[i].Default = true
						hasFoundDefault = true
					}
				}
				if !hasFoundDefault && len(config.Option) > 0 {
					log.Error("not found default option, set first option as default", "mod", mod.Name, "config", config.Name)
					config.Option[0].Default = true
				}

				mod.Config = append(mod.Config, config)
			}
		})
	}
	return nil
}

func parseModLuaByRegex(content string) (map[string]string, error) {
	L := lua.NewState()
	defer L.Close()
	err := L.DoString(content)
	if err != nil {
		return nil, errutil.Wrap(err)
	}

	enableModIDs := make(map[string]struct{})
	modIDRegex := regexp.MustCompile(`\d+`)
	data := L.Get(-1).(*lua.LTable)
	L.Pop(1)
	data.ForEach(func(workShopId lua.LValue, config lua.LValue) {
		modId := modIDRegex.FindString(workShopId.String())
		if modId == "" {
			log.Error("regex mod id failed", "workshop_id_string", workShopId.String())
			return
		}

		configTable := config.(*lua.LTable)
		enabled := configTable.RawGetString("enabled").(lua.LBool)
		if enabled {
			enableModIDs[modId] = struct{}{}
		}
	})

	modIDRegex = regexp.MustCompile(`(?s)\["workshop-(\d+)"]=\{.*?\{.*?}.*?enabled.*?}`)
	modIDs := modIDRegex.FindAllStringSubmatch(content, -1)
	configRegex := regexp.MustCompile(`(?s)\["workshop-\d+"]=\{.*?\{.*?}.*?enabled.*?}`)
	configs := configRegex.FindAllString(content, -1)
	if len(modIDs) != len(configs) {
		return nil, fmt.Errorf("regex not match")
	}

	modID2Config := make(map[string]string)
	for i := range modIDs {
		id := modIDs[i][1]
		_, ok := enableModIDs[id]
		if !ok {
			continue
		}
		modID2Config[id] = configs[i]
	}
	return modID2Config, nil
}

func genModOverrideContent(rawConfigs []string) string {
	content := "return {" + strings.Join(rawConfigs, ",\n  ") + "}"
	return content
}
