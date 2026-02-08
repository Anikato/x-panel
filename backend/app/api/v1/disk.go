package v1

import (
	"xpanel/app/api/v1/helper"
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
