package dontstarve

import (
	"aurora-admin/internal/entity/dstety"
	"aurora-admin/pkg/cache"
	"aurora-admin/pkg/util"
	"context"
)

func (p *Plugin) GetConfig(ctx context.Context) dstety.Config {
	return dstety.Config{
		EnabledArchiveId: cache.G.DontStarve.EnabledArchiveID,
		Token:            cache.G.DontStarve.Token,
		TickRate:         cache.G.DontStarve.TickRate,
		Password:         cache.G.DontStarve.PassWord,
	}
}

func (p *Plugin) UpdateConfig(ctx context.Context, c dstety.Config) dstety.Config {
	changed := util.UpdateIfChanged(c.EnabledArchiveId, &cache.G.DontStarve.EnabledArchiveID)
	changed = util.UpdateIfChanged(c.Token, &cache.G.DontStarve.Token) || changed
	changed = util.UpdateIfChanged(c.TickRate, &cache.G.DontStarve.TickRate) || changed
	changed = util.UpdateIfChanged(c.Password, &cache.G.DontStarve.PassWord) || changed
	if changed {
		cache.Save()
	}
	return p.GetConfig(ctx)
}
