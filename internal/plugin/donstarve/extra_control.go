package donstarve

import (
	"context"
	"fmt"
	"github.com/vksir/vkiss-lib/pkg/util/errutil"
)

func (d *Plugin) GetPlayers(ctx context.Context) ([]string, error) {
	var players []string
	for _, p := range d.process {
		out, err := d.RunCmd(ctx, p, "c_listallplayers()")
		if err != nil {
			return nil, errutil.Wrap(err)
		}
		players = append(players, out)
	}
	return players, nil
}

func (d *Plugin) Announce(ctx context.Context, msg string) error {
	cmd := fmt.Sprintf("c_announce(\"%s\")", msg)
	_, err := d.RunCmd(ctx, d.process["Master"], cmd)
	if err != nil {
		return errutil.Wrap(err)
	}
	return nil
}

func (d *Plugin) Regenerate(ctx context.Context) error {
	_, err := d.RunCmd(ctx, d.process["Master"], "c_regenerateworld()")
	if err != nil {
		return errutil.Wrap(err)
	}
	return nil
}

func (d *Plugin) Rollback(ctx context.Context, days int) error {
	cmd := fmt.Sprintf("c_rollback(%d)", days)
	_, err := d.RunCmd(ctx, d.process["Master"], cmd)
	if err != nil {
		return errutil.Wrap(err)
	}
	return nil
}
