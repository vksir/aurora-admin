package dontstarve

import (
	"context"
	"errors"
	"time"

	"aurora-admin/pkg/service"
	"aurora-admin/pkg/util/errutil"
)

const (
	// playerProbeInterval 无人探测间隔
	playerProbeInterval = time.Minute
	// playerProbeConfirmDelay 二次确认间隔，降低玩家正在进入的竞态
	playerProbeConfirmDelay = 5 * time.Second
	// playerProbeFailLimit 连续探测失败上限，超过后放弃本轮更新
	playerProbeFailLimit = 10
	// busyRetryInterval ErrBusy 重试间隔
	busyRetryInterval = 10 * time.Second
	// busyRetryLimit ErrBusy 重试上限
	busyRetryLimit = 10
)

// waitNoPlayers 阻塞等待直到服务器无人在线（或服务已停止）。
// 有玩家期间每 AutoupdateNotifyInterval 公告一次提示，每 playerProbeInterval 探测一次，
// 永不强制重启。探测失败（如服务正忙）按保守策略继续等待，
// 连续失败超过 playerProbeFailLimit 次则放弃本轮。
func (p *Plugin) waitNoPlayers(ctx context.Context, announceMsg string) error {
	if !p.svc.Running() {
		return nil
	}

	timer := time.NewTimer(0) // 立即执行首次探测
	defer timer.Stop()

	var failCount int
	var lastNotify time.Time
	notify := func() {
		if time.Since(lastNotify) < time.Second*AutoupdateNotifyInterval {
			return
		}
		lastNotify = time.Now()
		err := p.Announce(ctx, announceMsg)
		if err != nil {
			p.logger.WarnC(ctx, "announce failed", "err", err)
		}
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}

		if !p.svc.Running() {
			// 等待期间服务被停止，无需继续等待
			return nil
		}

		empty, err := p.probeNoPlayers(ctx)
		if err != nil {
			failCount++
			p.logger.WarnC(ctx, "probe players failed, keep waiting",
				"err", err, "fail_count", failCount)
			if failCount >= playerProbeFailLimit {
				return errutil.WrapF("probe players failed %d times: %w", failCount, err)
			}
			notify()
			timer.Reset(playerProbeInterval)
			continue
		}

		failCount = 0
		if empty {
			return nil
		}

		p.logger.InfoC(ctx, "players online, wait for logout before update")
		notify()
		timer.Reset(playerProbeInterval)
	}
}

// probeNoPlayers 探测服务器是否无人。首次探测无人后再延迟确认一次，
// 降低玩家正在进入（尚未出现在列表）时误判的概率。
func (p *Plugin) probeNoPlayers(ctx context.Context) (bool, error) {
	empty, err := p.noPlayersOnce(ctx)
	if err != nil || !empty {
		return empty, err
	}

	select {
	case <-ctx.Done():
		return false, ctx.Err()
	case <-time.After(playerProbeConfirmDelay):
	}
	return p.noPlayersOnce(ctx)
}

func (p *Plugin) noPlayersOnce(ctx context.Context) (bool, error) {
	ids, err := p.GetPlayerIDs(ctx)
	if err != nil {
		return false, errutil.Wrap(err)
	}
	return len(ids) == 0, nil
}

// updateService 走服务层更新（stop -> steamcmd -> start），服务正忙时退避重试。
func (p *Plugin) updateService(ctx context.Context) error {
	return retryOnBusy(ctx, func(ctx context.Context) error {
		return p.svc.Update(ctx)
	})
}

// restartService 重启服务（重启时会重新部署配置），服务正忙时退避重试。
func (p *Plugin) restartService(ctx context.Context) error {
	return retryOnBusy(ctx, func(ctx context.Context) error {
		return p.svc.Restart(ctx)
	})
}

func retryOnBusy(ctx context.Context, f func(context.Context) error) error {
	var err error
	for range busyRetryLimit {
		err = f(ctx)
		if err == nil {
			return nil
		}
		if !errors.Is(err, service.ErrBusy) {
			return errutil.Wrap(err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(busyRetryInterval):
		}
	}
	return errutil.WrapF("still busy after %d retries: %w", busyRetryLimit, err)
}

// tryBeginUpdate 尝试进入更新流程。服务器更新与模组更新共用此互斥，
// 防止两个 cron 流程并发执行 steamcmd 与重启。
func (p *Plugin) tryBeginUpdate() bool {
	return p.updateMu.TryLock()
}

func (p *Plugin) endUpdate() {
	p.updateMu.Unlock()
}
