package v1

import (
	"xpanel/app/api/v1/helper"
	"xpanel/app/dto"
	"xpanel/app/service"

	"github.com/gin-gonic/gin"
)

type CertSyncAPI struct{}

// ======================= 证书源管理 =======================

func (a *CertSyncAPI) ListCertSources(c *gin.Context) {
	items, err := service.NewICertSourceService().GetList()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, items)
}

func (a *CertSyncAPI) CreateCertSource(c *gin.Context) {
	var req dto.CertSourceCreate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewICertSourceService().Create(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *CertSyncAPI) UpdateCertSource(c *gin.Context) {
	var req dto.CertSourceUpdate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewICertSourceService().Update(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *CertSyncAPI) DeleteCertSource(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewICertSourceService().Delete(req.ID); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *CertSyncAPI) SyncCertSource(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewICertSourceService().Sync(req.ID); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *CertSyncAPI) TestCertSource(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewICertSourceService().TestConnection(req.ID); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

// ======================= 同步日志 =======================

func (a *CertSyncAPI) SearchSyncLogs(c *gin.Context) {
	var req dto.SearchCertSyncLogReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	total, items, err := service.NewICertSyncLogService().SearchWithPage(req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithPage(c, total, items)
}

// ======================= 证书服务端 =======================

func (a *CertSyncAPI) GetCertServerSetting(c *gin.Context) {
	setting, err := service.NewICertServerService().GetSetting()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, setting)
}

func (a *CertSyncAPI) UpdateCertServerSetting(c *gin.Context) {
	var req dto.CertServerSetting
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewICertServerService().UpdateSetting(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *CertSyncAPI) ServeCerts(c *gin.Context) {
	items, err := service.NewICertServerService().ListCerts()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, items)
}
