package donstarve

import (
	"aurora-admin/internal/entity/dstety"
	"aurora-admin/thirdparty/steam"
	"context"
	"fmt"
	"github.com/vksir/vkiss-lib/pkg/log"
	"github.com/vksir/vkiss-lib/pkg/util/errutil"
	"github.com/yuin/gopher-lua"
	"regexp"
	"strings"
)

func parseModLua(content string) ([]*dstety.DontStarveMod, error) {
	L := lua.NewState()
	defer L.Close()
	err := L.DoString(content)
	if err != nil {
		return nil, errutil.Wrap(err)
	}

	modIdRegex := regexp.MustCompile(`(?s)\["workshop-(\d+)"]=\{.*?\{.*?}.*?enabled.*?}`)
	modIds := modIdRegex.FindAllStringSubmatch(content, -1)
	configRegex := regexp.MustCompile(`(?s)\["workshop-\d+"]=\{.*?\{.*?}.*?enabled.*?}`)
	configs := configRegex.FindAllString(content, -1)
	if len(modIds) != len(configs) {
		return nil, fmt.Errorf("regex not match")
	}
	idToConfig := make(map[string]string)
	for i := range len(modIds) {
		idToConfig[modIds[i][1]] = configs[i]
	}

	var mods []*dstety.DontStarveMod
	modIdRegex = regexp.MustCompile(`\d+`)
	data := L.Get(-1).(*lua.LTable)
	L.Pop(1)
	data.ForEach(func(workShopId lua.LValue, config lua.LValue) {
		modId := modIdRegex.FindString(workShopId.String())
		if modId == "" {
			err = fmt.Errorf("regex modId failed: %s", workShopId.String())
		}

		configTable := config.(*lua.LTable)
		enabled := configTable.RawGetString("enabled").(lua.LBool)
		if enabled {
			conf, ok := idToConfig[modId]
			if !ok {
				err = fmt.Errorf("mod config not found: %s", modId)
				return
			}
			mods = append(mods, &dstety.DontStarveMod{
				WorkshopId: modId,
				Config:     conf,
			})
		}
	})
	if err != nil {
		return nil, errutil.Wrap(err)
	}
	return mods, nil
}

func generateModLua(mods []dstety.DontStarveModInArchive) string {
	content := strings.Builder{}
	content.WriteString("return {\n")

	for i, mod := range mods {
		var s string
		if i == len(mods)-1 {
			s = fmt.Sprintf("  %s\n", mod.CurrentConfig)
		} else {
			s = fmt.Sprintf("  %s,\n", mod.CurrentConfig)
		}
		content.WriteString(s)
	}

	content.WriteString("}")
	return content.String()
}

func generateModSetup(mods []dstety.DontStarveModInArchive) string {
	content := strings.Builder{}
	for _, mod := range mods {
		content.WriteString(fmt.Sprintf("ServerModSetup(\"%s\")\n", mod.Mod.WorkshopId))
	}
	return content.String()
}

func fetchModsName(ctx context.Context, mods []dstety.DontStarveMod) []dstety.DontStarveMod {
	var modIds []string
	for _, m := range mods {
		modIds = append(modIds, m.WorkshopId)
	}
	infos, err := steam.GetWorkShopItemInfos(modIds)
	if err != nil {
		log.ErrorC(ctx, "steam get work shop item info failed: %v", err)
		return mods
	}

	modMap := make(map[string]dstety.DontStarveMod)
	for _, m := range mods {
		modMap[m.WorkshopId] = m
	}

	for _, info := range infos {
		mod := modMap[info.Id]
		mod.Name = info.Name

		modMap[info.Id] = mod
	}

	var newMods []dstety.DontStarveMod
	for _, m := range modMap {
		newMods = append(newMods, m)
	}
	return newMods
}
