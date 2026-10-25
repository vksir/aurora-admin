package dontstarve

import (
	"context"
	"fmt"
	"regexp"
	"sort"

	"aurora-admin/pkg/service"
	"aurora-admin/pkg/util/errutil"
)

// playerIDRegex 匹配 DST 玩家 userid（KU_ 开头的 Klei 账号 ID）。
// c_listallplayers() 的输出形如 "[1] Wilson (KU_xxxxxxxx)"，
// 以 userid 为准比解析具体格式更稳健。
var playerIDRegex = regexp.MustCompile(`KU_[0-9A-Za-z]+`)

// extractPlayerIDs 从 c_listallplayers() 的控制台输出中提取去重后的玩家 userid 列表。
func extractPlayerIDs(output string) []string {
	seen := make(map[string]struct{})
	for _, id := range playerIDRegex.FindAllString(output, -1) {
		seen[id] = struct{}{}
	}
	ids := make([]string, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// GetPlayerIDs 返回当前在线玩家的 userid 列表（跨 Master/Caves shard 去重）。
// 服务未运行时返回 ErrNotStartedYet。
func (p *Plugin) GetPlayerIDs(ctx context.Context) ([]string, error) {
	seen := make(map[string]struct{})
	err := p.svc.Control(func(process map[string]*service.SubProcess) error {
		for _, pp := range process {
			out, err := p.RunCmd(ctx, pp, "c_listallplayers()")
			if err != nil {
				return errutil.Wrap(err)
			}
			for _, id := range extractPlayerIDs(out) {
				seen[id] = struct{}{}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids, nil
}

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
	if players == nil {
		players = []string{}
	}
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
