package v1

import (
	"xpanel/app/api/v1/helper"
	"xpanel/app/dto"
	"xpanel/app/service"

	"github.com/gin-gonic/gin"
)

type WebsiteAPI struct{}

var websiteService = service.NewIWebsiteService()

func (a *WebsiteAPI) SearchWebsite(c *gin.Context) {
	var req dto.WebsiteSearch
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	total, items, err := websiteService.SearchWithPage(req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithPage(c, total, items)
}

func (a *WebsiteAPI) CreateWebsite(c *gin.Context) {
	var req dto.WebsiteCreate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := websiteService.Create(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *WebsiteAPI) UpdateWebsite(c *gin.Context) {
	var req dto.WebsiteUpdate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := websiteService.Update(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *WebsiteAPI) DeleteWebsite(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := websiteService.Delete(req.ID); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *WebsiteAPI) GetWebsiteDetail(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	detail, err := websiteService.GetDetail(req.ID)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, detail)
}

func (a *WebsiteAPI) EnableWebsite(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := websiteService.Enable(req.ID); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *WebsiteAPI) DisableWebsite(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := websiteService.Disable(req.ID); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *WebsiteAPI) GetWebsiteNginxConfig(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	config, err := websiteService.GetNginxConfig(req.ID)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, config)
}

func (a *WebsiteAPI) GetWebsiteLog(c *gin.Context) {
	var req dto.WebsiteLogReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	content, err := websiteService.GetSiteLog(req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, content)
}

// --- Nginx 配置文件管理 ---

func (a *WebsiteAPI) GetNginxMainConf(c *gin.Context) {
	content, err := websiteService.GetMainConf()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, content)
}

func (a *WebsiteAPI) SaveNginxMainConf(c *gin.Context) {
	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := websiteService.SaveMainConf(req.Content); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *WebsiteAPI) ListNginxConfFiles(c *gin.Context) {
	files, err := websiteService.ListConfFiles()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, files)
}

func (a *WebsiteAPI) GetNginxConfFile(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	content, err := websiteService.GetConfFile(req.Name)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, content)
}

func (a *WebsiteAPI) SaveNginxConfFile(c *gin.Context) {
	var req dto.NginxConfUpdate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := websiteService.SaveConfFile(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}
