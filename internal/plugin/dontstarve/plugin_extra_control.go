package dontstarve

import (
	"context"
	"fmt"
	"github.com/vksir/vkiss-lib/pkg/service"
	"github.com/vksir/vkiss-lib/pkg/util/errutil"
)

func (p *Plugin) GetPlayers(ctx context.Context) ([]string, error) {
	var players []string
	err := p.svc.Control(func(process map[string]*service.SubProcess) error {
		for _, pp := range process {
			out, err := p.RunCmd(ctx, pp, "c_listallplayers()")
			if err != nil {
				return errutil.Wrap(err)
			}
			players = append(players, out)
		}
		return nil
	})
	return players, err
}

func (p *Plugin) Announce(ctx context.Context, msg string) error {
	return p.svc.Control(func(process map[string]*service.SubProcess) error {
		cmd := fmt.Sprintf("c_announce(\"%s\")", msg)
		_, err := p.RunCmd(ctx, process[ShardMaster], cmd)
		if err != nil {
			return errutil.Wrap(err)
		}
		return nil
	})
}

func (p *Plugin) Regenerate(ctx context.Context) error {
	return p.svc.Control(func(process map[string]*service.SubProcess) error {
		_, err := p.RunCmd(ctx, process[ShardMaster], "c_regenerateworld()")
		if err != nil {
			return errutil.Wrap(err)
		}
		return nil
	})
}

func (p *Plugin) Rollback(ctx context.Context, days int) error {
	return p.svc.Control(func(process map[string]*service.SubProcess) error {
		cmd := fmt.Sprintf("c_rollback(%d)", days)
		_, err := p.RunCmd(ctx, process[ShardMaster], cmd)
		if err != nil {
			return errutil.Wrap(err)
		}
		return nil
	})
}
