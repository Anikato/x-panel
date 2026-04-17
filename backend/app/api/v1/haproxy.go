package v1

import (
	"xpanel/app/api/v1/helper"
	"xpanel/app/dto"
	"xpanel/app/service"

	"github.com/gin-gonic/gin"
)

type HAProxyAPI struct{}

func operatorFrom(c *gin.Context) string {
	if v, ok := c.Get("userName"); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return "system"
}

// --- 安装 / 状态 / 升级 ---

func (a *HAProxyAPI) GetHAProxyStatus(c *gin.Context) {
	status, err := service.NewIHAProxyInstallService().GetStatus()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, status)
}

func (a *HAProxyAPI) InstallHAProxy(c *gin.Context) {
	var req dto.HAProxyInstallReq
	_ = c.ShouldBindJSON(&req)
	if err := service.NewIHAProxyInstallService().Install(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *HAProxyAPI) GetHAProxyInstallProgress(c *gin.Context) {
	helper.SuccessWithData(c, service.NewIHAProxyInstallService().GetProgress())
}

func (a *HAProxyAPI) UninstallHAProxy(c *gin.Context) {
	if err := service.NewIHAProxyInstallService().Uninstall(); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *HAProxyAPI) OperateHAProxy(c *gin.Context) {
	var req dto.HAProxyOperateReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIHAProxyInstallService().Operate(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *HAProxyAPI) CheckHAProxyUpdate(c *gin.Context) {
	resp, err := service.NewIHAProxyInstallService().CheckUpdate()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, resp)
}

func (a *HAProxyAPI) UpgradeHAProxy(c *gin.Context) {
	var req dto.HAProxyUpgradeReq
	_ = c.ShouldBindJSON(&req)
	if err := service.NewIHAProxyInstallService().Upgrade(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

// --- LB ---

func (a *HAProxyAPI) SearchHAProxyLB(c *gin.Context) {
	var req dto.HAProxyLBSearch
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	total, items, err := service.NewIHAProxyService().SearchLB(req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithPage(c, total, items)
}

func (a *HAProxyAPI) ListHAProxyLB(c *gin.Context) {
	items, err := service.NewIHAProxyService().ListLB()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, items)
}

func (a *HAProxyAPI) CreateHAProxyLB(c *gin.Context) {
	var req dto.HAProxyLBCreate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIHAProxyService().CreateLB(req, operatorFrom(c)); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *HAProxyAPI) UpdateHAProxyLB(c *gin.Context) {
	var req dto.HAProxyLBUpdate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIHAProxyService().UpdateLB(req, operatorFrom(c)); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *HAProxyAPI) DeleteHAProxyLB(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIHAProxyService().DeleteLB(req.ID, operatorFrom(c)); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *HAProxyAPI) ToggleHAProxyLB(c *gin.Context) {
	var req dto.HAProxyLBToggle
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIHAProxyService().ToggleLB(req, operatorFrom(c)); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

// --- Backend ---

func (a *HAProxyAPI) SearchHAProxyBackend(c *gin.Context) {
	var req dto.HAProxyBackendSearch
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	total, items, err := service.NewIHAProxyService().SearchBackend(req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithPage(c, total, items)
}

func (a *HAProxyAPI) ListHAProxyBackend(c *gin.Context) {
	items, err := service.NewIHAProxyService().ListBackend()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, items)
}

func (a *HAProxyAPI) GetHAProxyBackend(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	item, err := service.NewIHAProxyService().GetBackend(req.ID)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, item)
}

func (a *HAProxyAPI) CreateHAProxyBackend(c *gin.Context) {
	var req dto.HAProxyBackendCreate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIHAProxyService().CreateBackend(req, operatorFrom(c)); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *HAProxyAPI) UpdateHAProxyBackend(c *gin.Context) {
	var req dto.HAProxyBackendUpdate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIHAProxyService().UpdateBackend(req, operatorFrom(c)); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *HAProxyAPI) DeleteHAProxyBackend(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIHAProxyService().DeleteBackend(req.ID, operatorFrom(c)); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

// --- Server ---

func (a *HAProxyAPI) CreateHAProxyServer(c *gin.Context) {
	var req dto.HAProxyServerCreate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIHAProxyService().CreateServer(req, operatorFrom(c)); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *HAProxyAPI) UpdateHAProxyServer(c *gin.Context) {
	var req dto.HAProxyServerUpdate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIHAProxyService().UpdateServer(req, operatorFrom(c)); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *HAProxyAPI) DeleteHAProxyServer(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIHAProxyService().DeleteServer(req.ID, operatorFrom(c)); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *HAProxyAPI) ToggleHAProxyServerLive(c *gin.Context) {
	var req dto.HAProxyServerToggleLive
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIHAProxyRuntimeService().ToggleServerLive(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *HAProxyAPI) SetHAProxyServerWeightLive(c *gin.Context) {
	var req dto.HAProxyServerWeightLive
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIHAProxyRuntimeService().SetServerWeightLive(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

// --- ACL ---

func (a *HAProxyAPI) ListHAProxyACL(c *gin.Context) {
	var req dto.HAProxyACLSearch
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	items, err := service.NewIHAProxyService().ListACL(req.LBID)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, items)
}

func (a *HAProxyAPI) CreateHAProxyACL(c *gin.Context) {
	var req dto.HAProxyACLCreate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIHAProxyService().CreateACL(req, operatorFrom(c)); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *HAProxyAPI) UpdateHAProxyACL(c *gin.Context) {
	var req dto.HAProxyACLUpdate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIHAProxyService().UpdateACL(req, operatorFrom(c)); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *HAProxyAPI) DeleteHAProxyACL(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIHAProxyService().DeleteACL(req.ID, operatorFrom(c)); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

// --- Stats / Runtime ---

func (a *HAProxyAPI) GetHAProxyStats(c *gin.Context) {
	data, err := service.NewIHAProxyRuntimeService().GetStats()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}

func (a *HAProxyAPI) GetHAProxyRuntimeInfo(c *gin.Context) {
	data, err := service.NewIHAProxyRuntimeService().GetInfo()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}

func (a *HAProxyAPI) ClearHAProxyCounters(c *gin.Context) {
	if err := service.NewIHAProxyRuntimeService().ClearCounters(); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

// --- Raw Config / 版本 ---

func (a *HAProxyAPI) GetHAProxyRawConfig(c *gin.Context) {
	content, err := service.NewIHAProxyService().GetRawConfig()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, dto.HAProxyRawConfig{Content: content})
}

func (a *HAProxyAPI) SaveHAProxyRawConfig(c *gin.Context) {
	var req dto.HAProxyRawConfig
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIHAProxyService().SaveRawConfig(req.Content, operatorFrom(c)); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *HAProxyAPI) TestHAProxyConfig(c *gin.Context) {
	var req dto.HAProxyConfigTestReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	resp, err := service.NewIHAProxyService().TestConfig(req.Content)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, resp)
}

func (a *HAProxyAPI) PreviewHAProxyConfig(c *gin.Context) {
	content, err := service.NewIHAProxyService().PreviewConfig()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, dto.HAProxyRawConfig{Content: content})
}

func (a *HAProxyAPI) RebuildHAProxyConfig(c *gin.Context) {
	if err := service.NewIHAProxyService().RebuildConfig(operatorFrom(c)); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *HAProxyAPI) ListHAProxyConfigVersions(c *gin.Context) {
	items, err := service.NewIHAProxyService().ListConfigVersions(50)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, items)
}

func (a *HAProxyAPI) GetHAProxyConfigVersion(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	content, err := service.NewIHAProxyService().GetConfigVersion(req.ID)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, dto.HAProxyRawConfig{Content: content})
}

func (a *HAProxyAPI) RollbackHAProxyConfig(c *gin.Context) {
	var req dto.HAProxyConfigRollbackReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIHAProxyService().RollbackToVersion(req.ID, operatorFrom(c)); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}
