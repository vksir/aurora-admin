package dstapi

import (
	"aurora-admin/internal/entity/dstety"
	"aurora-admin/internal/entity/svcety"
	"aurora-admin/internal/plugin/donstarve"
	"context"
	"github.com/vksir/vkiss-lib/pkg/endpoint"
)

type Api struct {
	plugin *donstarve.Plugin
}

func NewApi(dst *donstarve.Plugin) *Api {
	return &Api{plugin: dst}
}

func (d *Api) Resource() []endpoint.Resource {
	return []endpoint.Resource{
		{
			RelativePath: "/dontstarve/status",
			Handler: []endpoint.Handler{
				endpoint.GetResource[statusResp](d.getStatus),
			},
		},
		{
			RelativePath: "/dontstarve/control",
			Handler: []endpoint.Handler{
				endpoint.CreateResource[createControlReq, controlResp](d.createControl),
			},
		},
		{
			RelativePath: "/dontstarve/config",
			Handler: []endpoint.Handler{
				endpoint.GetResource[configResp](d.getConfig),
				endpoint.UpdateResource[updateConfigReq, configResp](d.updateConfig),
			},
		},
		{
			RelativePath: "/dontstarve/archive",
			Handler: []endpoint.Handler{
				endpoint.IndexResource[archiveResp](d.indexArchive),
				endpoint.ListResource[listArchiveReq, archiveResp](d.listArchive),
				endpoint.CreateResource[createArchiveReq, archiveResp](d.createArchive),
				endpoint.UpdateResource[updateArchiveReq, archiveResp](d.updateArchive),
				endpoint.DeleteResource(d.deleteArchive),
			},
		},
	}
}

type statusResp = svcety.Status

// @Summary			获取状态
// @Tags			dontstarve_status
// @Accept			json
// @Produce			json
// @Success			200 {object} httputil.Response
// @Failure			500 {object} httputil.Response
// @Router			/api/dontstarve/status [get]
func (d *Api) getStatus(ctx context.Context) (statusResp, error) {
	return d.plugin.Status(ctx), nil
}

type controlResp struct{}
type createControlReq = svcety.Control

// @Summary			服务控制
// @Tags			dontstarve_control
// @Accept			json
// @Produce			json
// @Param			request body createControlReq true "body"
// @Success			200 {object} httputil.Response
// @Failure			500 {object} httputil.Response
// @Router			/api/dontstarve/control [post]
func (d *Api) createControl(ctx context.Context, req createControlReq) (controlResp, error) {
	var err error
	switch req.Action {
	case "start":
		err = d.plugin.Start(ctx)
	case "stop":
		err = d.plugin.Stop(ctx)
	case "restart":
		err = d.plugin.Restart(ctx)
	case "install":
		err = d.plugin.Install(ctx)
	case "uninstall":
		err = d.plugin.Uninstall(ctx)
	case "upgrade":
		err = d.plugin.Upgrade(ctx)
	}
	return controlResp{}, err
}

type configResp = dstety.Config
type updateConfigReq = dstety.Config

// @Summary			获取配置
// @Tags			dontstarve_config
// @Accept			json
// @Produce			json
// @Success			200 {object} httputil.Response
// @Failure			500 {object} httputil.Response
// @Router			/api/dontstarve/config [get]
func (d *Api) getConfig(ctx context.Context) (configResp, error) {
	return d.plugin.GetConfig(ctx), nil
}

// @Summary			更新配置
// @Tags			dontstarve_config
// @Accept			json
// @Produce			json
// @Param			id path string true "id"
// @Param			request body updateConfigReq true "body"
// @Success			200 {object} httputil.Response
// @Failure			500 {object} httputil.Response
// @Router			/api/dontstarve/config [put]
func (d *Api) updateConfig(ctx context.Context, id string, req updateConfigReq) (configResp, error) {
	return d.plugin.UpdateConfig(ctx, req), nil
}

type archiveResp = dstety.DontStarveArchive
type listArchiveReq struct{}
type createArchiveReq struct{}
type updateArchiveReq = dstety.DontStarveArchive

// @Summary			获取存档
// @Tags			dontstarve_archive
// @Accept			json
// @Produce			json
// @Param			id path string true "id"
// @Success			200 {object} httputil.Response
// @Failure			500 {object} httputil.Response
// @Router			/api/dontstarve/archive/{id} [get]
func (d *Api) indexArchive(ctx context.Context, id string) (archiveResp, error) {
	return d.plugin.GetArchive(ctx, id)
}

// @Summary			获取存档列表
// @Tags			dontstarve_archive
// @Accept			json
// @Produce			json
// @Param			request query listArchiveReq true "query"
// @Success			200 {object} httputil.Response
// @Failure			500 {object} httputil.Response
// @Router			/api/dontstarve/archive [get]
func (d *Api) listArchive(ctx context.Context, req listArchiveReq) ([]archiveResp, error) {
	return d.plugin.ListArchive(ctx)
}

// @Summary			创建存档
// @Tags			dontstarve_archive
// @Accept			json
// @Produce			json
// @Param			request body createArchiveReq true "body"
// @Success			200 {object} httputil.Response
// @Failure			500 {object} httputil.Response
// @Router			/api/dontstarve/archive [post]
func (d *Api) createArchive(ctx context.Context, req createArchiveReq) (archiveResp, error) {
	return d.plugin.CreateArchive(ctx)
}

// @Summary			更新存档
// @Tags			dontstarve_archive
// @Accept			json
// @Produce			json
// @Param			id path string true "id"
// @Param			request body updateArchiveReq true "body"
// @Success			200 {object} httputil.Response
// @Failure			500 {object} httputil.Response
// @Router			/api/dontstarve/archive [put]
func (d *Api) updateArchive(ctx context.Context, id string, req updateArchiveReq) (archiveResp, error) {
	req.ID = id
	return d.plugin.UpdateArchive(ctx, req)
}

// @Summary			删除存档
// @Tags			dontstarve_archive
// @Accept			json
// @Produce			json
// @Param			id path string true "id"
// @Success			200 {object} httputil.Response
// @Failure			500 {object} httputil.Response
// @Router			/api/dontstarve/archive/{id} [delete]
func (d *Api) deleteArchive(ctx context.Context, id string) error {
	return d.plugin.DeleteArchive(ctx, id)
}
