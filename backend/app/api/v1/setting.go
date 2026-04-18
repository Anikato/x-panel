package v1

import (
	"net/http"
	"os"
	"os/exec"
	"time"

	"xpanel/app/api/v1/helper"
	"xpanel/app/dto"
	"xpanel/app/service"
	"xpanel/global"

	"github.com/gin-gonic/gin"
)

// SettingAPI 面板设置接口
type SettingAPI struct{}

var settingService = service.NewISettingService()

// GetSettingInfo 获取所有面板设置
func (s *SettingAPI) GetSettingInfo(c *gin.Context) {
	info, err := settingService.GetSettingInfo()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, info)
}

// Update 更新面板设置
func (s *SettingAPI) Update(c *gin.Context) {
	var req dto.SettingUpdate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := settingService.Update(req); err != nil {
		helper.HandleError(c, err)
		return
	}

	helper.SuccessWithMsg(c, "MsgUpdateSuccess")
}

// UpdatePort 更新面板监听端口
func (s *SettingAPI) UpdatePort(c *gin.Context) {
	var req dto.PortUpdate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := settingService.UpdatePort(req); err != nil {
		helper.HandleError(c, err)
		return
	}

	helper.SuccessWithMsg(c, "MsgUpdateSuccess")
}

// RebootServer 重启服务器
func (s *SettingAPI) RebootServer(c *gin.Context) {
	global.LOG.Warn("Server reboot requested by user")
	helper.SuccessWithData(c, nil)
	go func() {
		time.Sleep(500 * time.Millisecond)
		exec.Command("reboot").Run()
	}()
}

// ShutdownServer 关闭服务器
func (s *SettingAPI) ShutdownServer(c *gin.Context) {
	global.LOG.Warn("Server shutdown requested by user")
	helper.SuccessWithData(c, nil)
	go func() {
		time.Sleep(500 * time.Millisecond)
		exec.Command("shutdown", "-h", "now").Run()
	}()
}

// TestProxy 测试代理连通性
func (s *SettingAPI) TestProxy(c *gin.Context) {
	var req dto.ProxyTest
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := settingService.TestProxy(req); err != nil {
		helper.ErrorWithDetail(c, http.StatusOK, err.Error())
		return
	}

	helper.SuccessWithMsg(c, "MsgUpdateSuccess")
}

// GetPanelSSL 获取面板 HTTPS 配置
func (s *SettingAPI) GetPanelSSL(c *gin.Context) {
	info, err := settingService.GetPanelSSL()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, info)
}

// UpdatePanelSSL 将面板 HTTPS 切换为证书管理中的证书（写 config，需重启生效）
func (s *SettingAPI) UpdatePanelSSL(c *gin.Context) {
	var req dto.PanelSSLUpdate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := settingService.UpdatePanelSSL(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithMsg(c, "MsgUpdateSuccess")
}

// RestartPanel 重启面板
func (s *SettingAPI) RestartPanel(c *gin.Context) {
	global.LOG.Warn("Panel restart requested by user")
	helper.SuccessWithData(c, nil)
	go func() {
		time.Sleep(500 * time.Millisecond)
		cmd := exec.Command("systemctl", "restart", "xpanel")
		if err := cmd.Start(); err != nil {
			global.LOG.Warnf("systemctl restart failed: %v, sending signal", err)
			proc, _ := os.FindProcess(os.Getpid())
			if proc != nil {
				proc.Signal(os.Interrupt)
			}
		}
	}()
}
