package v1

import (
	"net/http"
	"strconv"

	"xpanel/app/dto"
	"xpanel/app/service"
	"xpanel/utils/helper"
	"github.com/gin-gonic/gin"
)

type AppAPI struct {
	appService           service.IAppService
	appInstallService    service.IAppInstallService
	appBackupService     service.IAppBackupService
	appImportService     service.IAppImportService
	appImportProgressService service.IAppImportProgressService
}

func NewAppAPI() *AppAPI {
	return &AppAPI{
		appService:               service.NewIAppService(),
		appInstallService:        service.NewIAppInstallService(),
		appBackupService:         service.NewIAppBackupService(),
		appImportService:         service.NewIAppImportService(),
		appImportProgressService: service.NewIAppImportProgressService(),
	}
}

// SyncAppStore 同步应用商店
// @Summary 同步应用商店
// @Tags App
// @Accept json
// @Produce json
// @Param request body dto.AppSyncReq true "sync request"
// @Success 200
// @Router /apps/sync [post]
func (a *AppAPI) SyncAppStore(c *gin.Context) {
	var req dto.AppSyncReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}

	if err := a.appService.SyncAppStore(req.Force); err != nil {
		helper.ErrorWithDetail(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.SuccessWithMsg(c, "Sync started successfully")
}

// PageApps 分页查询应用
// @Summary 分页查询应用
// @Tags App
// @Accept json
// @Produce json
// @Param request body dto.AppSearchReq true "search request"
// @Success 200 {object} dto.PageResult
// @Router /apps/search [post]
func (a *AppAPI) PageApps(c *gin.Context) {
	var req dto.AppSearchReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}

	total, apps, err := a.appService.PageApps(req)
	if err != nil {
		helper.ErrorWithDetail(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.SuccessWithData(c, dto.PageResult{
		Total: total,
		Items: apps,
	})
}

// GetAppByKey 根据 key 获取应用详情
// @Summary 获取应用详情
// @Tags App
// @Accept json
// @Produce json
// @Param key path string true "app key"
// @Success 200 {object} dto.AppDTO
// @Router /apps/{key} [get]
func (a *AppAPI) GetAppByKey(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		helper.ErrorWithDetail(c, http.StatusBadRequest, "key is required")
		return
	}

	app, err := a.appService.GetAppByKey(key)
	if err != nil {
		helper.ErrorWithDetail(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.SuccessWithData(c, app)
}

// GetAppDetail 获取应用版本详情
// @Summary 获取应用版本详情
// @Tags App
// @Accept json
// @Produce json
// @Param appId query uint true "app id"
// @Param version query string true "version"
// @Success 200 {object} dto.AppDetailDTO
// @Router /apps/detail [get]
func (a *AppAPI) GetAppDetail(c *gin.Context) {
	appIDStr := c.Query("appId")
	version := c.Query("version")

	if appIDStr == "" || version == "" {
		helper.ErrorWithDetail(c, http.StatusBadRequest, "appId and version are required")
		return
	}

	appID, err := strconv.ParseUint(appIDStr, 10, 32)
	if err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, "invalid appId")
		return
	}

	detail, err := a.appService.GetAppDetail(uint(appID), version)
	if err != nil {
		helper.ErrorWithDetail(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.SuccessWithData(c, detail)
}

// GetTags 获取所有标签
// @Summary 获取所有标签
// @Tags App
// @Accept json
// @Produce json
// @Success 200 {array} dto.TagDTO
// @Router /apps/tags [get]
func (a *AppAPI) GetTags(c *gin.Context) {
	tags, err := a.appService.GetTags()
	if err != nil {
		helper.ErrorWithDetail(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.SuccessWithData(c, tags)
}

// InstallApp 安装应用
// @Summary 安装应用
// @Tags App
// @Accept json
// @Produce json
// @Param request body dto.AppInstallReq true "install request"
// @Success 200
// @Router /apps/install [post]
func (a *AppAPI) InstallApp(c *gin.Context) {
	var req dto.AppInstallReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}

	if err := a.appInstallService.Install(req); err != nil {
		helper.ErrorWithDetail(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.SuccessWithMsg(c, "Installation started successfully")
}

// PageInstalled 分页查询已安装应用
// @Summary 分页查询已安装应用
// @Tags App
// @Accept json
// @Produce json
// @Param request body dto.AppInstallSearchReq true "search request"
// @Success 200 {object} dto.PageResult
// @Router /apps/installed/search [post]
func (a *AppAPI) PageInstalled(c *gin.Context) {
	var req dto.AppInstallSearchReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}

	total, installs, err := a.appInstallService.PageInstalled(req)
	if err != nil {
		helper.ErrorWithDetail(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.SuccessWithData(c, dto.PageResult{
		Total: total,
		Items: installs,
	})
}

// GetInstalled 获取已安装应用详情
// @Summary 获取已安装应用详情
// @Tags App
// @Accept json
// @Produce json
// @Param id path uint true "install id"
// @Success 200 {object} dto.AppInstallDTO
// @Router /apps/installed/{id} [get]
func (a *AppAPI) GetInstalled(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		helper.ErrorWithDetail(c, http.StatusBadRequest, "id is required")
		return
	}

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, "invalid id")
		return
	}

	install, err := a.appInstallService.GetInstalled(uint(id))
	if err != nil {
		helper.ErrorWithDetail(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.SuccessWithData(c, install)
}

// OperateApp 操作应用（启动/停止/重启）
// @Summary 操作应用
// @Tags App
// @Accept json
// @Produce json
// @Param request body dto.AppOperateReq true "operate request"
// @Success 200
// @Router /apps/operate [post]
func (a *AppAPI) OperateApp(c *gin.Context) {
	var req dto.AppOperateReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}

	var err error
	switch req.Operation {
	case "start":
		err = a.appInstallService.Start(req.InstallID)
	case "stop":
		err = a.appInstallService.Stop(req.InstallID)
	case "restart":
		err = a.appInstallService.Restart(req.InstallID)
	default:
		helper.ErrorWithDetail(c, http.StatusBadRequest, "invalid operation")
		return
	}

	if err != nil {
		helper.ErrorWithDetail(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.SuccessWithMsg(c, "Operation completed successfully")
}

// UninstallApp 卸载应用
// @Summary 卸载应用
// @Tags App
// @Accept json
// @Produce json
// @Param request body dto.AppUninstallReq true "uninstall request"
// @Success 200
// @Router /apps/uninstall [post]
func (a *AppAPI) UninstallApp(c *gin.Context) {
	var req dto.AppUninstallReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}

	if err := a.appInstallService.Uninstall(req); err != nil {
		helper.ErrorWithDetail(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.SuccessWithMsg(c, "Uninstallation completed successfully")
}

// UpdateApp 更新应用
// @Summary 更新应用
// @Tags App
// @Accept json
// @Produce json
// @Param request body dto.AppUpdateReq true "update request"
// @Success 200
// @Router /apps/update [post]
func (a *AppAPI) UpdateApp(c *gin.Context) {
	var req dto.AppUpdateReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}

	if err := a.appInstallService.Update(req); err != nil {
		helper.ErrorWithDetail(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.SuccessWithMsg(c, "Update completed successfully")
}

// GetAppLogs 获取应用日志
// @Summary 获取应用日志
// @Tags App
// @Accept json
// @Produce json
// @Param id path int true "install id"
// @Param lines query int false "log lines" default(100)
// @Success 200 {object} string
// @Router /apps/installed/:id/logs [get]
func (a *AppAPI) GetAppLogs(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, "Invalid ID")
		return
	}

	linesStr := c.DefaultQuery("lines", "100")
	lines, err := strconv.Atoi(linesStr)
	if err != nil || lines <= 0 {
		lines = 100
	}
	if lines > 1000 {
		lines = 1000 // 最多 1000 行
	}

	logs, err := a.appInstallService.GetLogs(uint(id), lines)
	if err != nil {
		helper.ErrorWithDetail(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.SuccessWithData(c, logs)
}

// BackupApp 备份应用
// @Summary 备份应用
// @Tags App
// @Accept json
// @Produce json
// @Param request body dto.AppBackupReq true "backup request"
// @Success 200
// @Router /apps/backup [post]
func (a *AppAPI) BackupApp(c *gin.Context) {
	var req dto.AppBackupReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}

	if err := a.appBackupService.Backup(req); err != nil {
		helper.ErrorWithDetail(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.SuccessWithMsg(c, "Backup completed successfully")
}

// RestoreApp 恢复应用
// @Summary 恢复应用
// @Tags App
// @Accept json
// @Produce json
// @Param request body dto.AppRestoreReq true "restore request"
// @Success 200
// @Router /apps/restore [post]
func (a *AppAPI) RestoreApp(c *gin.Context) {
	var req dto.AppRestoreReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}

	if err := a.appBackupService.Restore(req); err != nil {
		helper.ErrorWithDetail(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.SuccessWithMsg(c, "Restore completed successfully")
}

// PageBackups 分页查询备份记录
// @Summary 分页查询备份记录
// @Tags App
// @Accept json
// @Produce json
// @Param request body dto.AppInstallSearchReq true "search request"
// @Success 200 {object} dto.PageResult
// @Router /apps/backups/search [post]
func (a *AppAPI) PageBackups(c *gin.Context) {
	var req dto.AppInstallSearchReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}

	total, backups, err := a.appBackupService.PageBackups(req)
	if err != nil {
		helper.ErrorWithDetail(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.SuccessWithData(c, dto.PageResult{
		Total: total,
		Items: backups,
	})
}

// DeleteBackup 删除备份
// @Summary 删除备份
// @Tags App
// @Accept json
// @Produce json
// @Param request body dto.OperateByIDs true "delete request"
// @Success 200
// @Router /apps/backups/del [post]
func (a *AppAPI) DeleteBackup(c *gin.Context) {
	var req dto.OperateByIDs
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}

	for _, id := range req.IDs {
		if err := a.appBackupService.DeleteBackup(id); err != nil {
			helper.ErrorWithDetail(c, http.StatusInternalServerError, err.Error())
			return
		}
	}

	helper.SuccessWithMsg(c, "Backups deleted successfully")
}

// ImportApp 从备份导入应用
// @Summary 从备份导入应用
// @Tags App
// @Accept json
// @Produce json
// @Param request body dto.AppImportReq true "import request"
// @Success 200
// @Router /apps/import [post]
func (a *AppAPI) ImportApp(c *gin.Context) {
	var req dto.AppImportReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}

	if err := a.appImportProgressService.StartImportWithProgress(req); err != nil {
		helper.ErrorWithDetail(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.SuccessWithMsg(c, "Import task started successfully")
}

// GetImportProgress 获取导入进度
// @Summary 获取导入进度
// @Tags App
// @Produce json
// @Param name path string true "import task name"
// @Success 200 {object} model.AppImportTask
// @Router /apps/import/progress/{name} [get]
func (a *AppAPI) GetImportProgress(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		helper.ErrorWithDetail(c, http.StatusBadRequest, "Task name is required")
		return
	}

	task, err := a.appImportProgressService.GetImportProgress(name)
	if err != nil {
		helper.ErrorWithDetail(c, http.StatusNotFound, err.Error())
		return
	}

	helper.SuccessWithData(c, task)
}

// GetImportTasks 获取所有导入任务
// @Summary 获取所有导入任务
// @Tags App
// @Produce json
// @Success 200 {array} model.AppImportTask
// @Router /apps/import/tasks [get]
func (a *AppAPI) GetImportTasks(c *gin.Context) {
	tasks, err := a.appImportProgressService.GetImportTasks()
	if err != nil {
		helper.ErrorWithDetail(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.SuccessWithData(c, tasks)
}
