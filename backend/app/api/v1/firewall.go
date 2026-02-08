package v1

import (
	"xpanel/app/api/v1/helper"
	"xpanel/app/dto"
	"xpanel/app/service"

	"github.com/gin-gonic/gin"
)

type FirewallAPI struct{}

func (a *FirewallAPI) GetBaseInfo(c *gin.Context) {
	info, err := service.NewIFirewallService().GetBaseInfo()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, info)
}

func (a *FirewallAPI) Operate(c *gin.Context) {
	var req dto.FirewallOperateReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIFirewallService().Operate(req.Operation); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *FirewallAPI) ListPortRules(c *gin.Context) {
	var req dto.PortRuleSearch
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	total, items, err := service.NewIFirewallService().ListPortRules(req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithPage(c, total, items)
}

func (a *FirewallAPI) CreatePortRule(c *gin.Context) {
	var req dto.PortRuleCreate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIFirewallService().CreatePortRule(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *FirewallAPI) DeletePortRule(c *gin.Context) {
	var req dto.PortRuleDelete
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIFirewallService().DeletePortRule(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *FirewallAPI) ListIPRules(c *gin.Context) {
	items, err := service.NewIFirewallService().ListIPRules()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, items)
}

func (a *FirewallAPI) CreateIPRule(c *gin.Context) {
	var req dto.IPRuleCreate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIFirewallService().CreateIPRule(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *FirewallAPI) DeleteIPRule(c *gin.Context) {
	var req dto.IPRuleDelete
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIFirewallService().DeleteIPRule(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}
