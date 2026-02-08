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

// InstallNginx 从预编译仓库安装 Nginx
func (api *NginxAPI) InstallNginx(c *gin.Context) {
	var req dto.NginxInstallReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
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
	if err := nginxInstallService.Uninstall(); err != nil {
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
