package v1

import (
	"net/http"

	"xpanel/app/api/v1/helper"
	"xpanel/app/dto"
	"xpanel/app/service"

	"github.com/gin-gonic/gin"
)

type DiskAPI struct{}

func (a *DiskAPI) GetDiskInfo(c *gin.Context) {
	info, err := service.NewIDiskService().GetDiskInfo()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, info)
}

func (a *DiskAPI) BrowseShares(c *gin.Context) {
	var req dto.BrowseSharesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	shares, err := service.NewIDiskService().BrowseShares(req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, shares)
}

func (a *DiskAPI) MountRemote(c *gin.Context) {
	var req dto.RemoteMountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := service.NewIDiskService().MountRemote(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, nil)
}

func (a *DiskAPI) UnmountRemote(c *gin.Context) {
	var req dto.RemoteUnmountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := service.NewIDiskService().UnmountRemote(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, nil)
}

func (a *DiskAPI) ListRemoteMounts(c *gin.Context) {
	list, err := service.NewIDiskService().ListRemoteMounts()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, list)
}

func (a *DiskAPI) ListBlockDevices(c *gin.Context) {
	list, err := service.NewIDiskService().ListBlockDevices()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, list)
}

func (a *DiskAPI) MountLocal(c *gin.Context) {
	var req dto.LocalMountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := service.NewIDiskService().MountLocal(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, nil)
}

func (a *DiskAPI) UnmountLocal(c *gin.Context) {
	var req dto.LocalUnmountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := service.NewIDiskService().UnmountLocal(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, nil)
}
