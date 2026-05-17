package v1

import (
	"net/http"

	"xpanel/app/api/v1/helper"
	"xpanel/app/dto"
	"xpanel/app/service"

	"github.com/gin-gonic/gin"
)

type NotificationAPI struct{}

var notificationService = service.NewINotificationService()

func (a *NotificationAPI) SearchNotifications(c *gin.Context) {
	var req dto.NotificationSearch
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	total, items, err := notificationService.SearchWithPage(req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithPage(c, total, items)
}

func (a *NotificationAPI) GetNotificationSummary(c *gin.Context) {
	summary, err := notificationService.Summary()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, summary)
}

func (a *NotificationAPI) GetRecentNotifications(c *gin.Context) {
	items, err := notificationService.Recent(10)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, items)
}

func (a *NotificationAPI) GetNotificationPreference(c *gin.Context) {
	pref, err := notificationService.GetPreference()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, pref)
}

func (a *NotificationAPI) UpdateNotificationPreference(c *gin.Context) {
	var req dto.NotificationPreference
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := notificationService.UpdatePreference(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *NotificationAPI) MarkNotificationsRead(c *gin.Context) {
	var req struct {
		IDs []uint `json:"ids" binding:"required"`
	}
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := notificationService.MarkRead(req.IDs); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *NotificationAPI) MarkAllNotificationsRead(c *gin.Context) {
	if err := notificationService.MarkAllRead(); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *NotificationAPI) DeleteReadNotifications(c *gin.Context) {
	if err := notificationService.DeleteRead(); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *NotificationAPI) DeleteAllNotifications(c *gin.Context) {
	if err := notificationService.DeleteAll(); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *NotificationAPI) DeleteNotification(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := notificationService.Delete(req.ID); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}
