package dstapi

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"

	"aurora-admin/internal/entity/comety"
	"aurora-admin/internal/entity/dstety"
	"aurora-admin/internal/plugin/dontstarve"
	"aurora-admin/internal/store"
	"aurora-admin/pkg/registry"

	"github.com/gin-gonic/gin"
	"aurora-admin/pkg/log"
)

type Api struct {
	plugin *dontstarve.Plugin
	db     *store.DB
}

func NewApi(dst *dontstarve.Plugin, db *store.DB) *Api {
	return &Api{plugin: dst, db: db}
}

func (d *Api) Load(g *gin.RouterGroup) {
	dst := g.Group("/dst")
	{
		// Config
		dst.GET("/config", d.GetConfig)
		dst.PUT("/config", d.UpdateConfig)

		// Archives
		dst.GET("/archives", d.ListArchive)
		dst.POST("/archives", d.CreateArchive)
		dst.GET("/archives/:id", d.GetArchive)
		dst.PUT("/archives/:id", d.UpdateArchive)
		dst.DELETE("/archives/:id", d.DeleteArchive)

		// Archive transfer
		dst.POST("/archives/upload", d.UploadCreateArchive)
		dst.PUT("/archives/:id/upload", d.UploadUpdateArchive)
		dst.GET("/archives/:id/download", d.DownloadArchive)

		// Archive enable
		dst.POST("/archives/:id/enable", d.EnableArchive)
		dst.GET("/archives/enabled/download", d.DownloadEnabledArchive)

		// Mods
		dst.GET("/mods", d.ListMod)
		dst.POST("/mods", d.CreateMod)
		dst.GET("/mods/:id", d.GetMod)
		dst.PUT("/mods/:id", d.UpdateMod)
		dst.DELETE("/mods/:id", d.DeleteMod)
		dst.POST("/mods/upload", d.UploadModOverrides)
		dst.POST("/mods/check-update", d.CheckModUpdate)
		dst.POST("/mods/update", d.UpdateMods)

		// Admins
		dst.GET("/admins", d.ListAdmin)
		dst.POST("/admins", d.CreateAdmin)
		dst.DELETE("/admins/:id", d.DeleteAdmin)

		// Console
		dst.POST("/console/players", d.GetPlayers)
		dst.POST("/console/announce", d.Announce)
		dst.POST("/console/regenerate", d.Regenerate)
		dst.POST("/console/rollback", d.Rollback)

		// Service log
		dst.GET("/service/log/stream", d.StreamServiceLog)
	}
}

// GetConfig
// @Summary			获取配置
// @Tags			dontstarve_config
// @Accept			json
// @Produce			json
// @Success			200 {object} comety.Response{data=dstety.Config}
// @Failure			500 {object} any
// @Router			/api/v1/dst/config [get]
func (d *Api) GetConfig(c *gin.Context) {
	c.JSON(http.StatusOK, comety.Response{Data: d.plugin.GetConfig(c)})
}

// UpdateConfig
// @Summary			更新配置
// @Tags			dontstarve_config
// @Accept			json
// @Produce			json
// @Param			request body dstety.Config true "body"
// @Success			200 {object} comety.Response{data=dstety.Config}
// @Failure			500 {object} any
// @Router			/api/v1/dst/config [put]
func (d *Api) UpdateConfig(c *gin.Context) {
	var req dstety.Config
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, comety.Response{Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, comety.Response{Data: d.plugin.UpdateConfig(c, req)})
}

// GetArchive
// @Summary			获取存档详情
// @Tags			dontstarve_archive
// @Accept			json
// @Produce			json
// @Param			id path string true "id"
// @Success			200 {object} comety.Response{data=store.Archive}
// @Failure			500 {object} any
// @Router			/api/v1/dst/archives/{id} [get]
func (d *Api) GetArchive(c *gin.Context) {
	id := c.Param("id")
	// 走 plugin.GetArchive：旧存档在 archive_mods 为空时会按存档目录的
	// Master/modoverrides.lua 回填模组关联（只补不删）
	res, err := d.plugin.GetArchive(c, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, comety.Response{Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, comety.Response{Data: res})
}

// ListArchive
// @Summary			获取存档列表
// @Tags			dontstarve_archive
// @Accept			json
// @Produce			json
// @Param			request query any true "query"
// @Success			200 {object} comety.Response{data=[]store.Archive}
// @Failure			500 {object} any
// @Router			/api/v1/dst/archives [get]
func (d *Api) ListArchive(c *gin.Context) {
	res, err := d.db.ListArchives(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, comety.Response{Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, comety.Response{Data: res})
}

// CreateArchive
// @Summary			创建存档
// @Tags			dontstarve_archive
// @Accept			json
// @Produce			json
// @Param			request body any true "body"
// @Success			200 {object} comety.Response{data=store.Archive}
// @Failure			500 {object} any
// @Router			/api/v1/dst/archives [post]
func (d *Api) CreateArchive(c *gin.Context) {
	res, err := d.plugin.CreateArchive(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, comety.Response{Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, comety.Response{Data: res})
}

// UpdateArchive
// @Summary			更新存档
// @Tags			dontstarve_archive
// @Accept			json
// @Produce			json
// @Param			id path string true "id"
// @Param			request body store.Archive true "body"
// @Success			200 {object} comety.Response{data=store.Archive}
// @Failure			500 {object} any
// @Router			/api/v1/dst/archives/{id} [put]
func (d *Api) UpdateArchive(c *gin.Context) {
	id := c.Param("id")
	var req store.Archive
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, comety.Response{Message: err.Error()})
		return
	}
	req.ID = id
	res, err := d.plugin.UpdateArchive(c, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, comety.Response{Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, comety.Response{Data: res})
}

// DeleteArchive
// @Summary			删除存档
// @Tags			dontstarve_archive
// @Accept			json
// @Produce			json
// @Param			id path string true "id"
// @Success			200 {object} comety.Response
// @Failure			500 {object} any
// @Router			/api/v1/dst/archives/{id} [delete]
func (d *Api) DeleteArchive(c *gin.Context) {
	id := c.Param("id")
	err := d.plugin.DeleteArchive(c, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, comety.Response{Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, comety.Response{})
}

// UploadCreateArchive
// @Summary			上传并创建存档
// @Tags			dontstarve_archive_transfer
// @Accept			multipart/form-data
// @Produce			json
// @Param			file formData file true "file"
// @Success			200 {object} comety.Response{data=store.Archive}
// @Failure			500 {object} any
// @Router			/api/v1/dst/archives/upload [post]
func (d *Api) UploadCreateArchive(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, comety.Response{Message: err.Error()})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, comety.Response{Message: err.Error()})
		return
	}
	defer log.Close(file)

	res, err := d.plugin.UploadCreateArchive(c, file, fileHeader.Filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, comety.Response{Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, comety.Response{Data: res})
}

// UploadUpdateArchive
// @Summary			上传并更新存档
// @Tags			dontstarve_archive_transfer
// @Accept			multipart/form-data
// @Produce			json
// @Param			id path string true "id"
// @Param			file formData file true "file"
// @Success			200 {object} comety.Response{data=store.Archive}
// @Failure			500 {object} any
// @Router			/api/v1/dst/archives/{id}/upload [put]
func (d *Api) UploadUpdateArchive(c *gin.Context) {
	id := c.Param("id")

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, comety.Response{Message: err.Error()})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, comety.Response{Message: err.Error()})
		return
	}
	defer log.Close(file)

	res, err := d.plugin.UploadUpdateArchive(c, file, id, fileHeader.Filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, comety.Response{Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, comety.Response{Data: res})
}

// DownloadArchive
// @Summary			下载存档
// @Tags			dontstarve_archive_transfer
// @Accept			json
// @Produce			octet-stream
// @Param			id path string true "id"
// @Success			200 {file} file
// @Failure			500 {object} any
// @Router			/api/v1/dst/archives/{id}/download [get]
func (d *Api) DownloadArchive(c *gin.Context) {
	id := c.Param("id")
	c.Header("Content-Disposition", "attachment; filename="+id+".zip")
	c.Header("Content-Type", "application/zip")

	err := d.plugin.DownloadArchive(c, id, c.Writer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, comety.Response{Message: err.Error()})
		return
	}
}

// CreateModRequest 按创意工坊 ID 添加模组的请求。
type CreateModRequest struct {
	// 创意工坊 ID（纯数字），如 1392778117
	WorkshopID string `json:"workshop_id" binding:"required"`
	// 模组配置，可选。支持完整配置块、整个 modoverrides.lua 内容或仅模组内部配置项；缺省使用默认配置
	RawConfig string `json:"raw_config"`
}

// isClientError 判断是否为客户端输入错误（应返回 400 而非 500）。
func isClientError(err error) bool {
	return errors.Is(err, dontstarve.ErrInvalidWorkshopID) ||
		errors.Is(err, dontstarve.ErrModNotFound) ||
		errors.Is(err, dontstarve.ErrInvalidModConfig) ||
		errors.Is(err, dontstarve.ErrNoEnabledMods)
}

// CreateMod
// @Summary			添加模组
// @Description		通过创意工坊 ID 添加模组。raw_config 可选：缺省时新模组使用默认配置，
// @Description		已存在的模组保留原配置；支持完整配置块、整个 modoverrides.lua 内容或仅模组内部配置项三种形式
// @Tags			dontstarve_mod
// @Accept			json
// @Produce			json
// @Param			request body CreateModRequest true "body"
// @Success			200 {object} comety.Response{data=store.Mod}
// @Failure			400 {object} any
// @Failure			500 {object} any
// @Router			/api/v1/dst/mods [post]
func (d *Api) CreateMod(c *gin.Context) {
	var req CreateModRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, comety.Response{Message: err.Error()})
		return
	}
	req.WorkshopID = strings.TrimSpace(req.WorkshopID)

	res, err := d.plugin.CreateMod(c, req.WorkshopID, req.RawConfig)
	if err != nil {
		if isClientError(err) {
			c.JSON(http.StatusBadRequest, comety.Response{Message: err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, comety.Response{Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, comety.Response{Data: res})
}

// UploadModOverrides
// @Summary			上传 modoverrides.lua 导入模组
// @Description		解析上传的 modoverrides.lua 文件，导入其中所有启用的模组（enabled = true）；
// @Description		已存在的模组会更新配置与元信息
// @Tags			dontstarve_mod
// @Accept			multipart/form-data
// @Produce			json
// @Param			file formData file true "modoverrides.lua 文件"
// @Success			200 {object} comety.Response{data=[]store.Mod}
// @Failure			400 {object} any
// @Failure			500 {object} any
// @Router			/api/v1/dst/mods/upload [post]
func (d *Api) UploadModOverrides(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, comety.Response{Message: err.Error()})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, comety.Response{Message: err.Error()})
		return
	}
	defer log.Close(file)

	content, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, comety.Response{Message: err.Error()})
		return
	}

	res, err := d.plugin.UploadModOverrides(c, content)
	if err != nil {
		if isClientError(err) {
			c.JSON(http.StatusBadRequest, comety.Response{Message: err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, comety.Response{Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, comety.Response{Data: res})
}

// ListMod
// @Summary			获取模组列表
// @Tags			dontstarve_mod
// @Accept			json
// @Produce			json
// @Success			200 {object} comety.Response{data=[]store.Mod}
// @Failure			500 {object} any
// @Router			/api/v1/dst/mods [get]
func (d *Api) ListMod(c *gin.Context) {
	res, err := d.plugin.ListMod(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, comety.Response{Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, comety.Response{Data: res})
}

// GetMod
// @Summary			获取模组详情
// @Tags			dontstarve_mod
// @Accept			json
// @Produce			json
// @Param			id path string true "id"
// @Success			200 {object} comety.Response{data=store.Mod}
// @Failure			500 {object} any
// @Router			/api/v1/dst/mods/{id} [get]
func (d *Api) GetMod(c *gin.Context) {
	id := c.Param("id")
	res, err := d.plugin.GetMod(c, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, comety.Response{Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, comety.Response{Data: res})
}

// UpdateMod
// @Summary			更新模组
// @Tags			dontstarve_mod
// @Accept			json
// @Produce			json
// @Param			id path string true "id"
// @Param			request body store.Mod true "body"
// @Success			200 {object} comety.Response{data=store.Mod}
// @Failure			500 {object} any
// @Router			/api/v1/dst/mods/{id} [put]
func (d *Api) UpdateMod(c *gin.Context) {
	id := c.Param("id")
	var req store.Mod
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, comety.Response{Message: err.Error()})
		return
	}
	res, err := d.plugin.UpdateMod(c, id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, comety.Response{Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, comety.Response{Data: res})
}

// DeleteMod
// @Summary			删除模组
// @Tags			dontstarve_mod
// @Accept			json
// @Produce			json
// @Param			id path string true "id"
// @Success			200 {object} comety.Response
// @Failure			500 {object} any
// @Router			/api/v1/dst/mods/{id} [delete]
func (d *Api) DeleteMod(c *gin.Context) {
	id := c.Param("id")
	err := d.plugin.DeleteMod(c, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, comety.Response{Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, comety.Response{})
}

// CheckModUpdate
// @Summary			检测模组更新
// @Description		对比 Workshop 与本地记录，返回需要更新的模组列表（不执行更新）
// @Tags			dontstarve_mod
// @Accept			json
// @Produce			json
// @Success			200 {object} comety.Response{data=[]dontstarve.ModUpdateStatus}
// @Failure			500 {object} any
// @Router			/api/v1/dst/mods/check-update [post]
func (d *Api) CheckModUpdate(c *gin.Context) {
	res, err := d.plugin.CheckModUpdates(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, comety.Response{Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, comety.Response{Data: res})
}

// UpdateMods
// @Summary			触发模组更新
// @Description		异步触发模组更新流程（检测->下载->安装->无人时重启）。有玩家在线时等待玩家下线
// @Tags			dontstarve_mod
// @Accept			json
// @Produce			json
// @Success			200 {object} comety.Response
// @Failure			500 {object} any
// @Router			/api/v1/dst/mods/update [post]
func (d *Api) UpdateMods(c *gin.Context) {
	registry.TriggerCronJob(c, dontstarve.ModUpdateCronJobName)
	c.JSON(http.StatusOK, comety.Response{})
}

// GetPlayers
// @Summary			获取玩家列表
// @Tags			dontstarve_extra_control
// @Accept			json
// @Produce			json
// @Success			200 {object} comety.Response{data=[]string}
// @Failure			500 {object} any
// @Router			/api/v1/dst/console/players [post]
func (d *Api) GetPlayers(c *gin.Context) {
	res, err := d.plugin.GetPlayers(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, comety.Response{Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, comety.Response{Data: res})
}

type AnnounceRequest struct {
	Msg string `json:"message"`
}

// Announce
// @Summary			发送公告
// @Tags			dontstarve_extra_control
// @Accept			json
// @Produce			json
// @Param			request body AnnounceRequest true "公告内容"
// @Success			200 {object} comety.Response
// @Failure			500 {object} any
// @Router			/api/v1/dst/console/announce [post]
func (d *Api) Announce(c *gin.Context) {
	var req AnnounceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, comety.Response{Message: err.Error()})
		return
	}
	if err := d.plugin.Announce(c, req.Msg); err != nil {
		c.JSON(http.StatusInternalServerError, comety.Response{Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, comety.Response{})
}

// Regenerate
// @Summary			重置世界
// @Tags			dontstarve_extra_control
// @Accept			json
// @Produce			json
// @Success			200 {object} comety.Response
// @Failure			500 {object} any
// @Router			/api/v1/dst/console/regenerate [post]
func (d *Api) Regenerate(c *gin.Context) {
	if err := d.plugin.Regenerate(c); err != nil {
		c.JSON(http.StatusInternalServerError, comety.Response{Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, comety.Response{})
}

type RollbackRequest struct {
	Days int `json:"days"`
}

// Rollback
// @Summary			回滚世界
// @Tags			dontstarve_extra_control
// @Accept			json
// @Produce			json
// @Param			request body RollbackRequest true "回滚天数"
// @Success			200 {object} comety.Response
// @Failure			500 {object} any
// @Router			/api/v1/dst/console/rollback [post]
func (d *Api) Rollback(c *gin.Context) {
	var req RollbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, comety.Response{Message: err.Error()})
		return
	}
	if err := d.plugin.Rollback(c, req.Days); err != nil {
		c.JSON(http.StatusInternalServerError, comety.Response{Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, comety.Response{})
}

// ListAdmin
// @Summary			获取管理员列表
// @Tags			dontstarve_admin
// @Accept			json
// @Produce			json
// @Success			200 {object} comety.Response{data=[]store.Admin}
// @Failure			500 {object} any
// @Router			/api/v1/dst/admins [get]
func (d *Api) ListAdmin(c *gin.Context) {
	res, err := d.db.ListAdmins(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, comety.Response{Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, comety.Response{Data: res})
}

// CreateAdmin
// @Summary			创建管理员
// @Tags			dontstarve_admin
// @Accept			json
// @Produce			json
// @Param			request body store.Admin true "body"
// @Success			200 {object} comety.Response{data=store.Admin}
// @Failure			500 {object} any
// @Router			/api/v1/dst/admins [post]
func (d *Api) CreateAdmin(c *gin.Context) {
	var req store.Admin
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, comety.Response{Message: err.Error()})
		return
	}
	res, err := d.db.CreateAdmin(c, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, comety.Response{Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, comety.Response{Data: res})
}

// DeleteAdmin
// @Summary			删除管理员
// @Tags			dontstarve_admin
// @Accept			json
// @Produce			json
// @Param			id path string true "id"
// @Success			200 {object} comety.Response
// @Failure			500 {object} any
// @Router			/api/v1/dst/admins/{id} [delete]
func (d *Api) DeleteAdmin(c *gin.Context) {
	id := c.Param("id")
	err := d.db.DeleteAdmin(c, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, comety.Response{Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, comety.Response{})
}

// EnableArchive
// @Summary			启用存档
// @Tags			dontstarve_archive
// @Accept			json
// @Produce			json
// @Param			id path string true "id"
// @Success			200 {object} comety.Response
// @Failure			500 {object} any
// @Router			/api/v1/dst/archives/{id}/enable [post]
func (d *Api) EnableArchive(c *gin.Context) {
	id := c.Param("id")
	conf := d.plugin.GetConfig(c)
	conf.EnabledArchiveId = id
	d.plugin.UpdateConfig(c, conf)
	c.JSON(http.StatusOK, comety.Response{})
}

// DownloadEnabledArchive
// @Summary			下载当前启用的存档
// @Tags			dontstarve_archive
// @Accept			json
// @Produce			octet-stream
// @Success			200 {file} file
// @Failure			500 {object} any
// @Router			/api/v1/dst/archives/enabled/download [get]
func (d *Api) DownloadEnabledArchive(c *gin.Context) {
	conf := d.plugin.GetConfig(c)
	if conf.EnabledArchiveId == "" {
		c.JSON(http.StatusBadRequest, comety.Response{Message: "no enabled archive"})
		return
	}

	c.Header("Content-Disposition", "attachment; filename="+conf.EnabledArchiveId+".zip")
	c.Header("Content-Type", "application/zip")

	err := d.plugin.DownloadArchive(c, conf.EnabledArchiveId, c.Writer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, comety.Response{Message: err.Error()})
		return
	}
}

// StreamServiceLog
// @Summary			服务日志流
// @Description		以 SSE 持续推送各分片服务端进程的 stdout/stderr；连接建立时回灌最近 300 行
// @Tags			dontstarve_service
// @Produce			text/event-stream
// @Success			200 {string} string "SSE 事件流：meta / line / reset / ping"
// @Failure			401 {object} any
// @Router			/api/v1/dst/service/log/stream [get]
func (d *Api) StreamServiceLog(c *gin.Context) {
	// 心跳与日志行都由 StreamLog 的同一个 goroutine 写出，避免 gin Writer 并发写
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	// 关闭 nginx 等反向代理的缓冲，否则日志会被攒起来一次性吐出
	c.Header("X-Accel-Buffering", "no")

	err := d.plugin.StreamLog(c.Request.Context(), func(ev dontstarve.LogEvent) error {
		if ev.Type == dontstarve.LogEventTypePing {
			_, err := io.WriteString(c.Writer, ": ping\n\n")
			if err != nil {
				return err
			}
		} else {
			c.SSEvent(string(ev.Type), ev)
		}
		c.Writer.Flush()
		return nil
	})
	if err != nil && !errors.Is(err, context.Canceled) {
		log.DefaultLogger().Debug("service log stream closed", "err", err)
	}
}
