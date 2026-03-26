package v1

import (
	"net/http"

	"xpanel/app/api/v1/helper"
	"xpanel/app/dto"
	"xpanel/app/service"

	"github.com/gin-gonic/gin"
)

type NginxAPI struct{}

var nginxService = service.NewINginxService()
var nginxInstallService = service.NewINginxInstallService()

// GetNginxStatus 获取 Nginx 运行状态
func (api *NginxAPI) GetNginxStatus(c *gin.Context) {
	status, err := nginxService.GetStatus()
	if err != nil {
		helper.ErrorWithDetail(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.SuccessWithData(c, status)
}

// OperateNginx 执行 Nginx 操作（start/stop/reload/reopen/quit）
func (api *NginxAPI) OperateNginx(c *gin.Context) {
	var req dto.NginxOperateReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}
	if err := nginxService.Operate(req); err != nil {
		helper.ErrorWithDetail(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.SuccessWithData(c, nil)
}

// TestNginxConfig 测试 Nginx 配置
func (api *NginxAPI) TestNginxConfig(c *gin.Context) {
	result, err := nginxService.TestConfig()
	if err != nil {
		helper.ErrorWithDetail(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.SuccessWithData(c, result)
}

// InstallNginx 安装 Nginx（apt 或预编译）
func (api *NginxAPI) InstallNginx(c *gin.Context) {
	var req dto.NginxInstallReq
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := nginxInstallService.Install(req); err != nil {
		helper.ErrorWithDetail(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.SuccessWithData(c, nil)
}

// GetInstallProgress 获取 Nginx 安装进度
func (api *NginxAPI) GetInstallProgress(c *gin.Context) {
	progress := nginxInstallService.GetProgress()
	helper.SuccessWithData(c, progress)
}

// UninstallNginx 卸载 Nginx
func (api *NginxAPI) UninstallNginx(c *gin.Context) {
	var req dto.NginxUninstallReq
	_ = c.ShouldBindJSON(&req)
	if err := nginxInstallService.Uninstall(req); err != nil {
		helper.ErrorWithDetail(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.SuccessWithData(c, nil)
}

// SetNginxAutoStart 设置 Nginx 开机自启
func (api *NginxAPI) SetNginxAutoStart(c *gin.Context) {
	var req dto.NginxAutoStartReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}
	if err := nginxService.SetAutoStart(req.Enable); err != nil {
		helper.ErrorWithDetail(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.SuccessWithData(c, nil)
}

// ListNginxVersions 获取可用的 Nginx 预编译版本列表
func (api *NginxAPI) ListNginxVersions(c *gin.Context) {
	versions, err := nginxInstallService.ListVersions()
	if err != nil {
		helper.ErrorWithDetail(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.SuccessWithData(c, versions)
}

// CheckNginxUpdate 检查 Nginx 是否有可用更新
func (api *NginxAPI) CheckNginxUpdate(c *gin.Context) {
	info, err := nginxInstallService.CheckUpdate()
	if err != nil {
		helper.ErrorWithDetail(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.SuccessWithData(c, info)
}

// UpgradeNginx 升级 Nginx
func (api *NginxAPI) UpgradeNginx(c *gin.Context) {
	var req dto.NginxUpgradeReq
	_ = c.ShouldBindJSON(&req)
	if err := nginxInstallService.Upgrade(req); err != nil {
		helper.ErrorWithDetail(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.SuccessWithData(c, nil)
}
