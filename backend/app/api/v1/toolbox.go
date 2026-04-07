package v1

import (
	"net/http"

	"xpanel/app/api/v1/helper"
	"xpanel/app/dto"
	"xpanel/app/service"

	"github.com/gin-gonic/gin"
)

type ToolboxAPI struct{}

// ====== Samba ======

func (a *ToolboxAPI) GetSambaStatus(c *gin.Context) {
	data, err := service.NewISambaService().GetStatus()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}

func (a *ToolboxAPI) InstallSamba(c *gin.Context) {
	if err := service.NewISambaService().Install(); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *ToolboxAPI) UninstallSamba(c *gin.Context) {
	if err := service.NewISambaService().Uninstall(); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *ToolboxAPI) OperateSamba(c *gin.Context) {
	var req dto.ServiceOperate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewISambaService().Operate(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *ToolboxAPI) ListSambaShares(c *gin.Context) {
	data, err := service.NewISambaService().ListShares()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}

func (a *ToolboxAPI) CreateSambaShare(c *gin.Context) {
	var req dto.SambaShareCreate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewISambaService().CreateShare(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *ToolboxAPI) UpdateSambaShare(c *gin.Context) {
	var req dto.SambaShareUpdate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewISambaService().UpdateShare(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *ToolboxAPI) DeleteSambaShare(c *gin.Context) {
	var req dto.SambaShareDelete
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewISambaService().DeleteShare(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *ToolboxAPI) ListSambaUsers(c *gin.Context) {
	data, err := service.NewISambaService().ListUsers()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}

func (a *ToolboxAPI) CreateSambaUser(c *gin.Context) {
	var req dto.SambaUserCreate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewISambaService().CreateUser(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *ToolboxAPI) DeleteSambaUser(c *gin.Context) {
	var req dto.SambaUserDelete
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewISambaService().DeleteUser(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *ToolboxAPI) UpdateSambaPassword(c *gin.Context) {
	var req dto.SambaPasswordUpdate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewISambaService().UpdatePassword(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *ToolboxAPI) ToggleSambaUser(c *gin.Context) {
	var req dto.SambaUserToggle
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewISambaService().ToggleUser(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *ToolboxAPI) GetSambaGlobalConfig(c *gin.Context) {
	data, err := service.NewISambaService().GetGlobalConfig()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}

func (a *ToolboxAPI) UpdateSambaGlobalConfig(c *gin.Context) {
	var req dto.SambaGlobalConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := service.NewISambaService().UpdateGlobalConfig(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *ToolboxAPI) GetSambaConnections(c *gin.Context) {
	data, err := service.NewISambaService().GetConnections()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}

// ====== NFS ======

func (a *ToolboxAPI) GetNfsStatus(c *gin.Context) {
	data, err := service.NewINfsService().GetStatus()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}

func (a *ToolboxAPI) InstallNfs(c *gin.Context) {
	if err := service.NewINfsService().Install(); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *ToolboxAPI) UninstallNfs(c *gin.Context) {
	if err := service.NewINfsService().Uninstall(); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *ToolboxAPI) OperateNfs(c *gin.Context) {
	var req dto.ServiceOperate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewINfsService().Operate(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *ToolboxAPI) ListNfsExports(c *gin.Context) {
	data, err := service.NewINfsService().ListExports()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}

func (a *ToolboxAPI) CreateNfsExport(c *gin.Context) {
	var req dto.NfsExportCreate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewINfsService().CreateExport(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *ToolboxAPI) UpdateNfsExport(c *gin.Context) {
	var req dto.NfsExportUpdate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewINfsService().UpdateExport(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *ToolboxAPI) DeleteNfsExport(c *gin.Context) {
	var req dto.NfsExportDelete
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewINfsService().DeleteExport(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *ToolboxAPI) GetNfsConnections(c *gin.Context) {
	data, err := service.NewINfsService().GetConnections()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}
