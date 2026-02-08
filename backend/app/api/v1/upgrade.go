package v1

import (
	"xpanel/app/api/v1/helper"
	"xpanel/app/dto"
	"xpanel/app/service"

	"github.com/gin-gonic/gin"
)

type UpgradeAPI struct{}

// GetVersion 获取当前版本信息
func (a *UpgradeAPI) GetVersion(c *gin.Context) {
	svc := service.NewIUpgradeService()
	helper.SuccessWithData(c, svc.GetCurrentVersion())
}

// CheckUpdate 检查可用更新
func (a *UpgradeAPI) CheckUpdate(c *gin.Context) {
	var req dto.UpgradeCheckReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	svc := service.NewIUpgradeService()
	data, err := svc.CheckUpdate(req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}

// DoUpgrade 执行升级
func (a *UpgradeAPI) DoUpgrade(c *gin.Context) {
	var req dto.UpgradeReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	svc := service.NewIUpgradeService()
	if err := svc.DoUpgrade(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

// GetUpgradeLog 获取升级日志
func (a *UpgradeAPI) GetUpgradeLog(c *gin.Context) {
	svc := service.NewIUpgradeService()
	data, err := svc.GetUpgradeLog()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}
