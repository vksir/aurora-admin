package dstapi

import (
	"aurora-admin/internal/entity/comety"
	"aurora-admin/internal/entity/dstety"
	"aurora-admin/internal/plugin/dontstarve"
	"aurora-admin/internal/store"
	"net/http"

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
		dst.GET("/mods/:id", d.GetMod)
		dst.PUT("/mods/:id", d.UpdateMod)
		dst.DELETE("/mods/:id", d.DeleteMod)

		// Admins
		dst.GET("/admins", d.ListAdmin)
		dst.POST("/admins", d.CreateAdmin)
		dst.DELETE("/admins/:id", d.DeleteAdmin)

		// Console
		dst.POST("/console/players", d.GetPlayers)
		dst.POST("/console/announce", d.Announce)
		dst.POST("/console/regenerate", d.Regenerate)
		dst.POST("/console/rollback", d.Rollback)
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
	res, err := d.db.GetArchive(c, id)
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
// @Success			200 {object} comety.Response{data=ent.DontStarveArchive}
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
// @Success			200 {object} comety.Response{data=ent.DontStarveArchive}
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

// UploadCreateEnableArchive
// @Summary			上传、创建并启用存档
// @Tags			dontstarve_archive_enable
// @Accept			multipart/form-data
// @Produce			json
// @Param			file formData file true "file"
// @Success			200 {object} comety.Response{data=ent.DontStarveArchive}
// @Failure			500 {object} any
// @Router			/api/v1/dst/archives/{id}/enable [post]
func (d *Api) UploadCreateEnableArchive(c *gin.Context) {
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

	conf := d.plugin.GetConfig(c)
	conf.EnabledArchiveId = res.ID
	d.plugin.UpdateConfig(c, conf)

	err = d.plugin.Service().Restart(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, comety.Response{Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, comety.Response{Data: res})
}

// UploadUpdateEnabledArchive
// @Summary			上传并更新当前启用的存档
// @Tags			dontstarve_archive_enable
// @Accept			multipart/form-data
// @Produce			json
// @Param			file formData file true "file"
// @Success			200 {object} comety.Response{data=ent.DontStarveArchive}
// @Failure			500 {object} any
// @Router			/api/v1/dst/archives/enabled/upload [put]
func (d *Api) UploadUpdateEnabledArchive(c *gin.Context) {
	conf := d.plugin.GetConfig(c)
	if conf.EnabledArchiveId == "" {
		c.JSON(http.StatusBadRequest, comety.Response{Message: "no enabled archive"})
		return
	}

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

	res, err := d.plugin.UploadUpdateArchive(c, file, conf.EnabledArchiveId, fileHeader.Filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, comety.Response{Message: err.Error()})
		return
	}

	err = d.plugin.Service().Restart(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, comety.Response{Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, comety.Response{Data: res})
}

// DownloadCurrentArchive
// @Summary			下载当前启用的存档
// @Tags			dontstarve_archive_enable
// @Accept			json
// @Produce			octet-stream
// @Success			200 {file} file
// @Failure			500 {object} any
// @Router			/api/v1/dst/archives/enabled/download [get]
func (d *Api) DownloadCurrentArchive(c *gin.Context) {
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
// @Success			200 {object} comety.Response{data=ent.DontStarveMod}
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

	err := d.plugin.Service().Restart(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, comety.Response{Message: err.Error()})
		return
	}
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
