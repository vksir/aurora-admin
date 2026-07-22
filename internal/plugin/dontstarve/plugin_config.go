package dontstarve

import (
	"aurora-admin/internal/entity/dstety"
	"aurora-admin/pkg/util"
	"context"
)

func (p *Plugin) GetConfig(ctx context.Context) dstety.Config {
	return dstety.Config{
		EnabledArchiveId: p.state.DontStarve.EnabledArchiveID,
		Token:            p.state.DontStarve.Token,
		TickRate:         p.state.DontStarve.TickRate,
		Password:         p.state.DontStarve.PassWord,
	}
}

func (p *Plugin) UpdateConfig(ctx context.Context, c dstety.Config) dstety.Config {
	changed := util.UpdateIfChanged(c.EnabledArchiveId, &p.state.DontStarve.EnabledArchiveID)
	changed = util.UpdateIfChanged(c.Token, &p.state.DontStarve.Token) || changed
	changed = util.UpdateIfChanged(c.TickRate, &p.state.DontStarve.TickRate) || changed
	changed = util.UpdateIfChanged(c.Password, &p.state.DontStarve.PassWord) || changed
	if changed {
		p.state.Save()
	}
	return p.GetConfig(ctx)
}
