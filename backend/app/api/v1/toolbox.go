package v1

import (
	"net/http"
	"strconv"

	"xpanel/app/api/v1/helper"
	"xpanel/app/dto"
	"xpanel/app/service"
	"xpanel/utils/iplocation"

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

// ====== Fail2ban ======

func (a *ToolboxAPI) GetFail2banStatus(c *gin.Context) {
	data, err := service.NewIFail2banService().GetStatus()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}

func (a *ToolboxAPI) InstallFail2ban(c *gin.Context) {
	if err := service.NewIFail2banService().Install(); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *ToolboxAPI) UninstallFail2ban(c *gin.Context) {
	if err := service.NewIFail2banService().Uninstall(); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *ToolboxAPI) OperateFail2ban(c *gin.Context) {
	var req dto.ServiceOperate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIFail2banService().Operate(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *ToolboxAPI) ListFail2banJails(c *gin.Context) {
	data, err := service.NewIFail2banService().ListJails()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}

func (a *ToolboxAPI) UpdateFail2banJail(c *gin.Context) {
	var req dto.Fail2banJailUpdate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIFail2banService().UpdateJail(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *ToolboxAPI) SetFail2banSSH(c *gin.Context) {
	var req dto.Fail2banSSHConfig
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIFail2banService().SetSSHJail(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *ToolboxAPI) ListFail2banBanned(c *gin.Context) {
	data, err := service.NewIFail2banService().ListBanned()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}

func (a *ToolboxAPI) UnbanFail2banIP(c *gin.Context) {
	var req dto.Fail2banUnbanReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIFail2banService().Unban(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *ToolboxAPI) GetFail2banLogs(c *gin.Context) {
	lines := 200
	if v := c.Query("lines"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			lines = n
		}
	}
	data, err := service.NewIFail2banService().GetLogs(lines)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}

// ====== IP Location ======

func (a *ToolboxAPI) LookupIP(c *gin.Context) {
	ip := c.Query("ip")
	if ip == "" {
		helper.ErrorWithDetail(c, http.StatusBadRequest, "ip is required")
		return
	}
	info := iplocation.GetService().Lookup(ip)
	helper.SuccessWithData(c, info)
}

func (a *ToolboxAPI) LookupIPBatch(c *gin.Context) {
	var req dto.IPBatchLookupReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	result := iplocation.GetService().LookupBatch(req.IPs)
	helper.SuccessWithData(c, result)
}

func (a *ToolboxAPI) GetIPDBInfo(c *gin.Context) {
	info := iplocation.GetService().GetDBInfo()
	helper.SuccessWithData(c, info)
}

func (a *ToolboxAPI) DownloadIPDB(c *gin.Context) {
	if err := iplocation.GetService().DownloadDB(); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

// ====== Systemd Service Manager ======

func (a *ToolboxAPI) ListSystemdServices(c *gin.Context) {
	showAll := c.Query("all") == "true"
	data, err := service.NewISystemdService().List(showAll)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}

func (a *ToolboxAPI) GetSystemdServiceDetail(c *gin.Context) {
	name := c.Query("name")
	if name == "" {
		helper.ErrorWithDetail(c, http.StatusBadRequest, "name is required")
		return
	}
	data, err := service.NewISystemdService().GetDetail(name)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}

func (a *ToolboxAPI) CreateSystemdService(c *gin.Context) {
	var req dto.SystemdServiceCreate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewISystemdService().Create(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *ToolboxAPI) UpdateSystemdService(c *gin.Context) {
	var req dto.SystemdServiceUpdate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewISystemdService().Update(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *ToolboxAPI) DeleteSystemdService(c *gin.Context) {
	var req dto.SystemdServiceDelete
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewISystemdService().Delete(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *ToolboxAPI) OperateSystemdService(c *gin.Context) {
	var req dto.SystemdServiceOperate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewISystemdService().Operate(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *ToolboxAPI) GetSystemdServiceLogs(c *gin.Context) {
	name := c.Query("name")
	if name == "" {
		helper.ErrorWithDetail(c, http.StatusBadRequest, "name is required")
		return
	}
	lines := 100
	if v := c.Query("lines"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			lines = n
		}
	}
	data, err := service.NewISystemdService().GetLogs(dto.SystemdServiceLogReq{Name: name, Lines: lines})
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}
