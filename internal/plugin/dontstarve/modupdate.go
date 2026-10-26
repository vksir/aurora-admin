package dontstarve

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"time"

	"aurora-admin/internal/store"
	"aurora-admin/pkg/log"
	"aurora-admin/pkg/registry"
	"aurora-admin/pkg/steam"
	"aurora-admin/pkg/util/errutil"
	"aurora-admin/pkg/util/fileutil"
)

// workshopQueryBatchSize Workshop 详情查询接口的单批数量上限
const workshopQueryBatchSize = 50

type modUpdateHelper struct {
	plugin *Plugin
	logger *log.Logger
}

func newModUpdateHelper(p *Plugin, logger *log.Logger) *modUpdateHelper {
	return &modUpdateHelper{plugin: p, logger: logger}
}

func (h *modUpdateHelper) RegisterCronJob() {
	registry.RegisterCronJob(ModUpdateCronJobName, time.Second*ModUpdateCronJobInterval, h.run)
}

// ModUpdateStatus 模组更新检测结果（API 对外展示用）。
// 字段名与前端 `apps/web-ele/src/api/dontstarve/index.ts` 的 ModUpdateStatus 保持一致。
type ModUpdateStatus struct {
	WorkshopID     string `json:"workshop_id"`
	Name           string `json:"name"`
	CurrentUpdated int64  `json:"current_time"`
	RemoteUpdated  int64  `json:"latest_time"`
	// NeedUpdate 当前实现只返回需要更新的模组，因此恒为 true
	NeedUpdate bool `json:"need_update"`
}

// modUpdate 待更新模组：DB 记录 + Workshop 最新详情
type modUpdate struct {
	mod    *store.Mod
	detail steam.Publishedfiledetail
}

func (h *modUpdateHelper) run(ctx context.Context) error {
	outdated, err := h.check(ctx)
	if err != nil {
		return errutil.Wrap(err)
	}
	if len(outdated) == 0 {
		return nil
	}

	for _, u := range outdated {
		h.logger.InfoC(ctx, "mod update detected",
			"workshop_id", u.mod.WorkshopID,
			"name", u.mod.Name,
			"current", u.mod.TimeUpdated, "remote", u.detail.TimeUpdated)
	}

	if !h.plugin.tryBeginUpdate() {
		h.logger.InfoC(ctx, "another update flow is running, skip this round")
		return nil
	}
	defer h.plugin.endUpdate()

	// 有玩家时只公告等待，玩家全部下线后再更新重启
	err = h.plugin.waitNoPlayers(ctx, msgModUpdateLogout)
	if err != nil {
		return errutil.Wrap(err)
	}

	err = h.downloadAndInstall(ctx, outdated)
	if err != nil {
		return errutil.Wrap(err)
	}

	// 重启使新模组生效（重启时 DeployBeforeStart 会重新生成 mods setup）
	if h.plugin.svc.Running() {
		err = h.plugin.restartService(ctx)
		if err != nil {
			return errutil.Wrap(err)
		}
	}

	// 成功后回写远端 time_updated，避免下轮重复检测
	for _, u := range outdated {
		u.mod.TimeUpdated = int64(u.detail.TimeUpdated)
		_, err = h.plugin.db.UpdateMod(ctx, u.mod)
		if err != nil {
			h.logger.ErrorC(ctx, "update mod record failed", "workshop_id", u.mod.WorkshopID, "err", err)
		}
	}

	h.logger.InfoC(ctx, "mod update success", "count", len(outdated))
	return nil
}

// check 对比 DB 与 Workshop 的模组更新时间，返回需要更新的模组。
// 同时顺带刷新无变化的模组的元信息（名称/描述/图标等）。
func (h *modUpdateHelper) check(ctx context.Context) ([]modUpdate, error) {
	mods, err := h.plugin.db.ListMods(ctx)
	if err != nil {
		return nil, errutil.Wrap(err)
	}
	if len(mods) == 0 {
		return nil, nil
	}

	byWorkshopID := make(map[string]*store.Mod, len(mods))
	workshopIDs := make([]string, 0, len(mods))
	for _, m := range mods {
		byWorkshopID[m.WorkshopID] = m
		workshopIDs = append(workshopIDs, m.WorkshopID)
	}

	var outdated []modUpdate
	for chunk := range slices.Chunk(workshopIDs, workshopQueryBatchSize) {
		resp, err := steam.GetPublishedFileDetails(chunk...)
		if err != nil {
			return nil, errutil.Wrap(err)
		}

		for i := range resp.Response.Publishedfiledetails {
			detail := resp.Response.Publishedfiledetails[i]
			if detail.Result != 1 {
				h.logger.WarnC(ctx, "query workshop item failed, skip",
					"workshop_id", detail.Publishedfileid, "result", detail.Result)
				continue
			}

			mod, ok := byWorkshopID[detail.Publishedfileid]
			if !ok {
				continue
			}

			// TimeUpdated 为 0 说明导入时元信息获取失败，视同需要更新以对齐远端
			needUpdate := mod.TimeUpdated == 0 || int64(detail.TimeUpdated) > mod.TimeUpdated
			if needUpdate {
				applyModInfo(mod, detail)
				outdated = append(outdated, modUpdate{mod: mod, detail: detail})
				continue
			}

			// 无更新时刷新元信息（若有变化）
			updated := *mod
			applyModInfo(&updated, detail)
			if updated.Name != mod.Name || updated.Description != mod.Description ||
				updated.Image != mod.Image || updated.TimeCreated != mod.TimeCreated {
				_, err = h.plugin.db.UpdateMod(ctx, &updated)
				if err != nil {
					h.logger.WarnC(ctx, "refresh mod info failed",
						"workshop_id", mod.WorkshopID, "err", err)
				}
			}
		}
	}
	return outdated, nil
}

// applyModInfo 将 Workshop 详情写入模组记录（不含 TimeUpdated，由更新成功后回写）。
func applyModInfo(mod *store.Mod, detail steam.Publishedfiledetail) {
	mod.Name = detail.Title
	mod.Description = detail.Description
	mod.Image = detail.PreviewUrl
	mod.TimeCreated = int64(detail.TimeCreated)
}

// CheckModUpdates 检测需要更新的模组（供手动触发/API 使用，不执行更新）。
func (p *Plugin) CheckModUpdates(ctx context.Context) ([]ModUpdateStatus, error) {
	outdated, err := p.modupdate.check(ctx)
	if err != nil {
		return nil, errutil.Wrap(err)
	}

	res := make([]ModUpdateStatus, 0, len(outdated))
	for _, u := range outdated {
		res = append(res, ModUpdateStatus{
			WorkshopID:     u.mod.WorkshopID,
			Name:           u.mod.Name,
			CurrentUpdated: u.mod.TimeUpdated,
			RemoteUpdated:  int64(u.detail.TimeUpdated),
			NeedUpdate:     true,
		})
	}
	return res, nil
}

// downloadAndInstall 用 steamcmd 下载全部待更新模组，并安装到服务器 mods 目录。
func (h *modUpdateHelper) downloadAndInstall(ctx context.Context, outdated []modUpdate) error {
	workshopIDs := make([]string, 0, len(outdated))
	for _, u := range outdated {
		workshopIDs = append(workshopIDs, u.mod.WorkshopID)
	}

	h.logger.InfoC(ctx, "begin download mods", "workshop_ids", workshopIDs)
	err := steam.NewSteamcmd(h.plugin.config.SteamCmdPath, h.plugin.logger).
		SetForceInstallDir(steam.WorkshopDir()).
		DoWorkShopDownloadItem(ctx, AppId, workshopIDs...)
	if err != nil {
		return errutil.Wrap(err)
	}

	for _, u := range outdated {
		err = h.installMod(ctx, u.mod.WorkshopID)
		if err != nil {
			return errutil.Wrap(err)
		}
	}
	return nil
}

// installMod 将 workshop 下载内容安装到服务器 mods 目录（mods/workshop-<id>），
// 替换旧版本。优先符号链接（省磁盘且与 workshop 目录保持同步），失败时回退为复制。
func (h *modUpdateHelper) installMod(ctx context.Context, workshopID string) error {
	src := steam.WorkshopItemPath(AppId, workshopID)
	if !fileutil.Exist(src) {
		return errutil.WrapF("workshop item %s not downloaded: %s", workshopID, src)
	}
	dst := filepath.Join(programDir(), "mods", "workshop-"+workshopID)

	err := os.RemoveAll(dst)
	if err != nil {
		return errutil.Wrap(err)
	}

	rel, relErr := filepath.Rel(filepath.Dir(dst), src)
	if relErr == nil {
		err = os.Symlink(rel, dst)
		if err == nil {
			h.logger.InfoC(ctx, "install mod by symlink", "dst", dst, "src", src)
			return nil
		}
		h.logger.WarnC(ctx, "symlink mod failed, fallback to copy", "dst", dst, "err", err)
	}

	err = fileutil.Cp(src, dst)
	if err != nil {
		return errutil.Wrap(err)
	}
	h.logger.InfoC(ctx, "install mod by copy", "dst", dst, "src", src)
	return nil
}
