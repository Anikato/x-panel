package v1

import (
	"xpanel/app/api/v1/helper"
	"xpanel/app/dto"
	"xpanel/app/service"

	"github.com/gin-gonic/gin"
)

// NezhaAgentAPI exposes the minimal private panel API for the bundled Nezha Agent.
// Logs are intentionally not exposed here; use /api/v1/toolbox/services/logs
// with the fixed xpanel-nezha-agent unit name.
type NezhaAgentAPI struct{}

// nezhaAgentService is the handler-facing subset of NezhaAgentService.
// Kept minimal so tests can inject fakes without the full service package surface.
type nezhaAgentService interface {
	Status() (*dto.NezhaAgentStatus, error)
	Configure(req dto.NezhaAgentConfigUpdate) error
	Operate(operation string) error
}

// newNezhaAgentService is a package-level factory for handler tests.
// Production defaults to service.NewNezhaAgentService. Tests must restore via cleanup.
var newNezhaAgentService = func() nezhaAgentService {
	return service.NewNezhaAgentService()
}

// GetNezhaAgentStatus returns component/config/service status without secrets.
func (a *NezhaAgentAPI) GetNezhaAgentStatus(c *gin.Context) {
	st, err := newNezhaAgentService().Status()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, st)
}

// UpdateNezhaAgentConfig applies a partial config update. Success has no data body.
// clientSecret is accepted for Configure but never logged or echoed by this handler.
func (a *NezhaAgentAPI) UpdateNezhaAgentConfig(c *gin.Context) {
	var req dto.NezhaAgentConfigUpdate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := newNezhaAgentService().Configure(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

// OperateNezhaAgent runs start|stop|restart|enable|disable. Success has no data body.
func (a *NezhaAgentAPI) OperateNezhaAgent(c *gin.Context) {
	var req dto.NezhaAgentOperateRequest
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := newNezhaAgentService().Operate(req.Operation); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}
