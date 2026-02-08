package v1

import (
	"net/http"

	"xpanel/app/api/v1/helper"
	"xpanel/app/dto"
	"xpanel/app/service"

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
