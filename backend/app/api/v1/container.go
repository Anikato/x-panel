package v1

import (
	"net/http"

	"xpanel/app/api/v1/helper"
	"xpanel/app/dto"
	"xpanel/app/service"

	"github.com/gin-gonic/gin"
)

type ContainerAPI struct{}

var containerService = service.NewIContainerService()
var composeService = service.NewIComposeService()

func (a *ContainerAPI) DockerStatus(c *gin.Context) {
	helper.SuccessWithData(c, gin.H{"available": containerService.DockerAvailable()})
}

// Container
func (a *ContainerAPI) ListContainers(c *gin.Context) {
	var req dto.ContainerSearch
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	total, items, err := containerService.ListContainers(req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithPage(c, total, items)
}

func (a *ContainerAPI) CreateContainer(c *gin.Context) {
	var req dto.ContainerCreate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := containerService.CreateContainer(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithMsg(c, "MsgCreateSuccess")
}

func (a *ContainerAPI) OperateContainer(c *gin.Context) {
	var req dto.ContainerOperate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := containerService.OperateContainer(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *ContainerAPI) ContainerLogs(c *gin.Context) {
	var req dto.ContainerLog
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	logs, err := containerService.ContainerLogs(req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, logs)
}

func (a *ContainerAPI) RemoveContainer(c *gin.Context) {
	var req struct {
		ContainerID string `json:"containerID" binding:"required"`
	}
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := containerService.RemoveContainer(req.ContainerID); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithMsg(c, "MsgDeleteSuccess")
}

// Image
func (a *ContainerAPI) ListImages(c *gin.Context) {
	items, err := containerService.ListImages()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, items)
}

func (a *ContainerAPI) PullImage(c *gin.Context) {
	var req dto.ImagePull
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := containerService.PullImage(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *ContainerAPI) RemoveImage(c *gin.Context) {
	var req struct {
		ImageID string `json:"imageID" binding:"required"`
	}
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := containerService.RemoveImage(req.ImageID); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithMsg(c, "MsgDeleteSuccess")
}

// Network
func (a *ContainerAPI) ListNetworks(c *gin.Context) {
	items, err := containerService.ListNetworks()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, items)
}

func (a *ContainerAPI) CreateNetwork(c *gin.Context) {
	var req dto.NetworkCreate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := containerService.CreateNetwork(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithMsg(c, "MsgCreateSuccess")
}

func (a *ContainerAPI) RemoveNetwork(c *gin.Context) {
	var req struct {
		NetworkID string `json:"networkID" binding:"required"`
	}
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := containerService.RemoveNetwork(req.NetworkID); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithMsg(c, "MsgDeleteSuccess")
}

// Volume
func (a *ContainerAPI) ListVolumes(c *gin.Context) {
	items, err := containerService.ListVolumes()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, items)
}

func (a *ContainerAPI) CreateVolume(c *gin.Context) {
	var req dto.VolumeCreate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := containerService.CreateVolume(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithMsg(c, "MsgCreateSuccess")
}

func (a *ContainerAPI) RemoveVolume(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := containerService.RemoveVolume(req.Name); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithMsg(c, "MsgDeleteSuccess")
}

// Compose
func (a *ContainerAPI) ListCompose(c *gin.Context) {
	items, err := composeService.ListComposeProjects()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, items)
}

func (a *ContainerAPI) CreateCompose(c *gin.Context) {
	var req dto.ComposeCreate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := composeService.CreateCompose(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithMsg(c, "MsgCreateSuccess")
}

func (a *ContainerAPI) OperateCompose(c *gin.Context) {
	var req dto.ComposeOperate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := composeService.OperateCompose(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}
