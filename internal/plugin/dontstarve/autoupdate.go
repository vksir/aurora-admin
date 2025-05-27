package dontstarve

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/vksir/vkiss-lib/pkg/log"
	"github.com/vksir/vkiss-lib/pkg/registry"
	"github.com/vksir/vkiss-lib/pkg/util/errutil"
	"github.com/vksir/vkiss-lib/pkg/util/fileutil"
	"github.com/vksir/vkiss-lib/thirdpkg/steam"
)

type autoupdateHelper struct {
	plugin *Plugin
	logger *log.Logger
}

func newAutoupdateHelper(p *Plugin, logger *log.Logger) *autoupdateHelper {
	return &autoupdateHelper{plugin: p, logger: logger}
}

func (h *autoupdateHelper) RegisterCronJob() {
	registry.RegisterCronJob(AutoUpdateCronJobName, time.Second*AutoupdateCronJobInterval, func(ctx context.Context) error {
		if h.Check(ctx) {
			err := h.Update(ctx)
			if err != nil {
				return errutil.Wrap(err)
			}
		}
		return nil
	})
}

func (h *autoupdateHelper) Check(ctx context.Context) bool {
	data, err := steam.GetSteamCmdAppInfo(ServerAppId)
	if err != nil {
		h.logger.ErrorC(ctx, "GetSteamCmdAppInfo failed", "err", err)
		return false
	}

	newestBuild, err := strconv.Atoi(data.Data.Field1.Depots.Branches.Public.Buildid)
	if err != nil {
		h.logger.ErrorC(ctx, "invalid build id",
			"build_id", data.Data.Field1.Depots.Branches.Public.Buildid, "err", err)
		return false
	}

	currentBuild, err := readCurrentBuildID()
	if err != nil {
		h.logger.ErrorC(ctx, "readCurrentBuildID failed", "err", err)
		return false
	}

	if newestBuild > currentBuild {
		h.logger.InfoC(ctx, "newestBuild is larger than currentBuild, need update",
			"newestBuild", newestBuild, "currentBuild", currentBuild)
		return true
	}

	h.logger.DebugC(ctx, "newestBuild is no larger than currentBuild, skip",
		"newestBuild", newestBuild, "currentBuild", currentBuild)
	return false
}

func (h *autoupdateHelper) Update(ctx context.Context) error {
	timer := time.NewTimer(time.Second * AutoupdateNotifyInterval)
	defer timer.Stop()

	notifyTimes := 0
	for !h.canUpdate(ctx) {
		h.notifyLogOut(ctx)
		notifyTimes++
		if notifyTimes >= 3 {
			h.logger.InfoC(ctx, "max notify times exceeded, force update in 15 seconds")
			h.notifyForceUpdate(ctx)
			time.Sleep(time.Second * 15)
			break
		}

		timer.Reset(time.Second * AutoupdateNotifyInterval)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
	}

	h.logger.InfoC(ctx, "begin auto update")
	return h.update(ctx)
}

func (h *autoupdateHelper) canUpdate(ctx context.Context) bool {
	if !h.plugin.Service().Running() {
		h.logger.InfoC(ctx, "service is not running, can update")
		return true
	}
	players, err := h.plugin.GetPlayers(ctx)
	if err != nil {
		h.logger.ErrorC(ctx, "get players failed", "err", err)
		return false
	}
	if len(players) == 0 {
		h.logger.InfoC(ctx, "non players, can update")
		return true
	}
	h.logger.InfoC(ctx, "exist players, cannot update", "players", players)
	return false
}

func (h *autoupdateHelper) update(ctx context.Context) error {
	return h.plugin.Update(ctx)
}

func (h *autoupdateHelper) notifyLogOut(ctx context.Context) {
	msg := fmt.Sprintf("%s The server requires an update. Please log out immediately. %s", iconEye, iconEye)
	err := h.plugin.Announce(ctx, msg)
	if err != nil {
		h.logger.ErrorC(ctx, "Announce failed", "err", err)
	}
}

func (h *autoupdateHelper) notifyForceUpdate(ctx context.Context) {
	msg := fmt.Sprintf("%s The server will forcibly update in 15 seconds. %s", iconEye, iconEye)
	err := h.plugin.Announce(ctx, msg)
	if err != nil {
		h.logger.ErrorC(ctx, "Announce failed", "err", err)
	}
}

func readCurrentBuildID() (int, error) {
	manifestPath := newPathHelper().AppManifestPath()
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
