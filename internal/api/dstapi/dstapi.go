package dstapi

import (
	"aurora-admin/ent"
	"aurora-admin/ent/dontstarvearchive"
	"aurora-admin/internal/entity/comety"
	"aurora-admin/internal/entity/dstety"
	"aurora-admin/internal/plugin/dontstarve"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vksir/vkiss-lib/pkg/log"
)

type Api struct {
	plugin *dontstarve.Plugin
	db     *ent.Client
}

func NewApi(dst *dontstarve.Plugin, db *ent.Client) *Api {
	return &Api{plugin: dst, db: db}
}

func (d *Api) Load(g *gin.RouterGroup) {
	dst := g.Group("/dontstarve")
	{
		dst.GET("/config", d.GetConfig)
		dst.PUT("/config", d.UpdateConfig)

		dst.GET("/archive", d.ListArchive)
		dst.GET("/archive/:id", d.GetArchive)
		dst.POST("/archive", d.CreateArchive)
		dst.PUT("/archive/:id", d.UpdateArchive)
		dst.DELETE("/archive/:id", d.DeleteArchive)

		dst.POST("/archive/upload", d.UploadCreateArchive)
		dst.PUT("/archive/upload/:id", d.UploadUpdateArchive)
		dst.GET("/archive/download/:id", d.DownloadArchive)
		dst.POST("/archive/enable/upload", d.UploadCreateEnableArchive)
		dst.PUT("/archive/enable/upload", d.UploadUpdateEnabledArchive)
		dst.GET("/archive/enable/download", d.DownloadCurrentArchive)

		dst.GET("/mod", d.ListMod)
		dst.GET("/mod/:id", d.GetMod)
		dst.PUT("/mod/:id", d.UpdateMod)
		dst.DELETE("/mod", d.DeleteMod)

		dst.GET("/admin", d.ListAdmin)
		dst.POST("/admin", d.CreateAdmin)
		dst.DELETE("/admin/:id", d.DeleteAdmin)

		dst.GET("/player", d.GetPlayers)
		dst.POST("/announce", d.Announce)
		dst.POST("/regenerate", d.Regenerate)
		dst.POST("/rollback", d.Rollback)
	}
}

// GetConfig
// @Summary			获取配置
// @Tags			dontstarve_config
// @Accept			json
// @Produce			json
// @Success			200 {object} comety.Response{data=dstety.Config}
// @Failure			500 {object} any
// @Router			/api/dontstarve/config [get]
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
// @Router			/api/dontstarve/config [put]
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
// @Success			200 {object} comety.Response{data=ent.DontStarveArchive}
// @Failure			500 {object} any
// @Router			/api/dontstarve/archive/{id} [get]
func (d *Api) GetArchive(c *gin.Context) {
	id := c.Param("id")
	res, err := d.db.DontStarveArchive.Query().Where(dontstarvearchive.ID(id)).WithMods().Only(c)
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
// @Success			200 {object} comety.Response{data=[]ent.DontStarveArchive}
// @Failure			500 {object} any
// @Router			/api/dontstarve/archive [get]
func (d *Api) ListArchive(c *gin.Context) {
	res, err := d.db.DontStarveArchive.Query().WithMods().All(c)
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
// @Router			/api/dontstarve/archive [post]
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
// @Param			request body ent.DontStarveArchive true "body"
// @Success			200 {object} comety.Response{data=ent.DontStarveArchive}
// @Failure			500 {object} any
// @Router			/api/dontstarve/archive/{id} [put]
func (d *Api) UpdateArchive(c *gin.Context) {
	id := c.Param("id")
	var req ent.DontStarveArchive
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
// @Router			/api/dontstarve/archive/{id} [delete]
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
// @Router			/api/dontstarve/archive/upload [post]
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
// @Success			200 {object} comety.Response{data=ent.DontStarveArchive}
// @Failure			500 {object} any
// @Router			/api/dontstarve/archive/upload/{id} [put]
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
// @Router			/api/dontstarve/archive/enable/upload [post]
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
// @Router			/api/dontstarve/archive/enable/upload [put]
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
// @Router			/api/dontstarve/archive/enable/download [get]
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
// @Router			/api/dontstarve/archive/download/{id} [get]
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
// @Success			200 {object} comety.Response{data=[]ent.DontStarveMod}
// @Failure			500 {object} any
// @Router			/api/dontstarve/mod [get]
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
// @Router			/api/dontstarve/mod/{id} [get]
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
// @Param			request body ent.DontStarveMod true "body"
// @Success			200 {object} comety.Response{data=ent.DontStarveMod}
// @Failure			500 {object} any
// @Router			/api/dontstarve/mod/{id} [put]
func (d *Api) UpdateMod(c *gin.Context) {
	id := c.Param("id")
	var req ent.DontStarveMod
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
// @Param			request body []string true "ids"
// @Success			200 {object} comety.Response
// @Failure			500 {object} any
// @Router			/api/dontstarve/mod [delete]
func (d *Api) DeleteMod(c *gin.Context) {
	var ids []string
	if err := c.ShouldBindJSON(&ids); err != nil {
		c.JSON(http.StatusBadRequest, comety.Response{Message: err.Error()})
		return
	}
	err := d.plugin.DeleteMod(c, ids...)
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
// @Router			/api/dontstarve/player [get]
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
// @Router			/api/dontstarve/announce [post]
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
// @Router			/api/dontstarve/regenerate [post]
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
// @Router			/api/dontstarve/rollback [post]
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
// @Success			200 {object} comety.Response{data=[]ent.DontStarveAdmin}
// @Failure			500 {object} any
// @Router			/api/dontstarve/admin [get]
func (d *Api) ListAdmin(c *gin.Context) {
	res, err := d.db.DontStarveAdmin.Query().All(c)
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
// @Param			request body ent.DontStarveAdmin true "body"
// @Success			200 {object} comety.Response{data=ent.DontStarveAdmin}
// @Failure			500 {object} any
// @Router			/api/dontstarve/admin [post]
func (d *Api) CreateAdmin(c *gin.Context) {
	var req ent.DontStarveAdmin
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, comety.Response{Message: err.Error()})
		return
	}
	res, err := d.db.DontStarveAdmin.Create().
		SetKleiID(req.KleiID).
		SetName(req.Name).
		Save(c)
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
// @Router			/api/dontstarve/admin/{id} [delete]
func (d *Api) DeleteAdmin(c *gin.Context) {
	id := c.Param("id")
	err := d.db.DontStarveAdmin.DeleteOneID(id).Exec(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, comety.Response{Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, comety.Response{})
}
