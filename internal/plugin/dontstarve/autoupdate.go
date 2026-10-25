package dontstarve

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"aurora-admin/pkg/log"
	"aurora-admin/pkg/registry"
	"aurora-admin/pkg/steam"
	"aurora-admin/pkg/util/errutil"
	"aurora-admin/pkg/util/fileutil"
)

const (
	// newestBuildFetchRetries buildid 查询重试次数（api.steamcmd.net 偶发不可用）
	newestBuildFetchRetries = 3
	// newestBuildFetchRetryInterval 查询重试间隔
	newestBuildFetchRetryInterval = 5 * time.Second
)

type autoupdateHelper struct {
	plugin *Plugin
	logger *log.Logger
}

func newAutoupdateHelper(p *Plugin, logger *log.Logger) *autoupdateHelper {
	return &autoupdateHelper{plugin: p, logger: logger}
}

func (h *autoupdateHelper) RegisterCronJob() {
	registry.RegisterCronJob(AutoUpdateCronJobName, time.Second*AutoupdateCronJobInterval, h.run)
}

func (h *autoupdateHelper) run(ctx context.Context) error {
	needDownload, needRestart, err := h.check(ctx)
	if err != nil {
		return errutil.Wrap(err)
	}
	if !needDownload && !needRestart {
		return nil
	}

	if !h.plugin.tryBeginUpdate() {
		h.logger.InfoC(ctx, "another update flow is running, skip this round")
		return nil
	}
	defer h.plugin.endUpdate()

	h.logger.InfoC(ctx, "begin auto update", "need_download", needDownload, "need_restart", needRestart)

	// 有玩家时只公告等待，玩家全部下线后才执行
	err = h.plugin.waitNoPlayers(ctx, msgServerUpdateLogout)
	if err != nil {
		return errutil.Wrap(err)
	}

	if needDownload {
		// stop -> steamcmd 更新 -> start（服务已停止则仅更新，不拉起）
		err = h.plugin.updateService(ctx)
	} else if h.plugin.svc.Running() {
		// 仅文件已更新而运行中进程仍旧版（如更新期间服务未运行的历史遗留），重启生效；
		// 服务已停止则无需处理，下次启动自然是新版
		err = h.plugin.restartService(ctx)
	}
	if err != nil {
		return errutil.Wrap(err)
	}

	h.logger.InfoC(ctx, "auto update success")
	return nil
}

// check 判断服务器是否需要更新。
// 返回 needDownload（需 steamcmd 拉取新版本）与 needRestart（文件已是新版但运行中进程仍旧版）。
func (h *autoupdateHelper) check(ctx context.Context) (needDownload, needRestart bool, err error) {
	newestBuild, err := h.fetchNewestBuildID(ctx)
	if err != nil {
		return false, false, errutil.Wrap(err)
	}

	currentBuild, err := readCurrentBuildID()
	if err != nil {
		// 尚未安装服务器等场景下 manifest 不存在，跳过本轮而非报错
		h.logger.WarnC(ctx, "readCurrentBuildID failed, skip this round", "err", err)
		return false, false, nil
	}

	if newestBuild > currentBuild {
		h.logger.InfoC(ctx, "new server version detected, need download",
			"newest", newestBuild, "current", currentBuild)
		return true, false, nil
	}

	// manifest 已是最新但运行中进程基于旧 build 启动（文件在运行期间被更新过），
	// 重启后才会生效
	lastRun := h.plugin.state.DontStarve.LastRunBuildID
	if h.plugin.svc.Running() && lastRun != 0 && lastRun != currentBuild {
		h.logger.InfoC(ctx, "server files updated but process is outdated, need restart",
			"last_run", lastRun, "current", currentBuild)
		return false, true, nil
	}

	h.logger.DebugC(ctx, "server is up to date",
		"newest", newestBuild, "current", currentBuild, "last_run", lastRun)
	return false, false, nil
}

// fetchNewestBuildID 查询最新 buildid，带重试。
func (h *autoupdateHelper) fetchNewestBuildID(ctx context.Context) (int, error) {
	var lastErr error
	for range newestBuildFetchRetries {
		data, err := steam.GetSteamCmdAppInfo(ServerAppId)
		if err != nil {
			lastErr = err
			h.logger.WarnC(ctx, "GetSteamCmdAppInfo failed, retry", "err", err)
			select {
			case <-ctx.Done():
				return 0, ctx.Err()
			case <-time.After(newestBuildFetchRetryInterval):
			}
			continue
		}

		build, err := strconv.Atoi(data.Data.Field1.Depots.Branches.Public.Buildid)
		if err != nil {
			lastErr = errutil.WrapF("invalid build id %q: %w",
				data.Data.Field1.Depots.Branches.Public.Buildid, err)
			h.logger.ErrorC(ctx, "invalid build id", "err", lastErr)
			continue
		}
		return build, nil
	}
	return 0, lastErr
}

// readCurrentBuildID 从本地 appmanifest 读取已安装的 buildid。
func readCurrentBuildID() (int, error) {
	manifestPath := appManifestPath()
	content, err := fileutil.Read(manifestPath)
	if err != nil {
		return 0, errutil.Wrap(err)
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if !strings.Contains(line, `"buildid"`) {
			continue
		}

		parts := strings.Split(line, `"`)
		if len(parts) < 4 {
			return 0, fmt.Errorf("invalid build id format: %s", line)
		}
		currentBuild, err := strconv.Atoi(parts[3])
		if err != nil {
			return 0, fmt.Errorf("invalid build id format: %s", line)
		}
		return currentBuild, nil
	}

	return 0, fmt.Errorf("build id not found: %s", string(content))
}
