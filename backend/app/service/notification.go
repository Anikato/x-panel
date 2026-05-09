package service

import (
	"strings"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/repo"
)

type INotificationService interface {
	Create(req dto.NotificationCreate) error
	SearchWithPage(req dto.NotificationSearch) (int64, []dto.NotificationInfo, error)
	Summary() (*dto.NotificationSummary, error)
	MarkRead(ids []uint) error
	MarkAllRead() error
	DeleteRead() error
	Delete(id uint) error
}

func NewINotificationService() INotificationService {
	return &NotificationService{
		notificationRepo: repo.NewINotificationRepo(),
	}
}

type NotificationService struct {
	notificationRepo repo.INotificationRepo
}

func (s *NotificationService) Create(req dto.NotificationCreate) error {
	notification := &model.Notification{
		Type:      normalizeNotificationType(req.Type),
		Title:     strings.TrimSpace(req.Title),
		Content:   strings.TrimSpace(req.Content),
		Source:    strings.TrimSpace(req.Source),
		TargetURL: strings.TrimSpace(req.TargetURL),
	}
	if notification.Title == "" {
		notification.Title = "系统通知"
	}
	if err := s.notificationRepo.Create(notification); err != nil {
		return err
	}
	ReportFleetNotification(*notification)
	return nil
}

func (s *NotificationService) SearchWithPage(req dto.NotificationSearch) (int64, []dto.NotificationInfo, error) {
	opts := []repo.DBOption{
		repo.WithNotificationStatus(req.Status),
		repo.WithNotificationType(req.Type),
		repo.WithNotificationSource(req.Source),
		repo.WithNotificationKeyword(req.Info),
	}
	total, notifications, err := s.notificationRepo.Page(req.Page, req.PageSize, opts...)
	if err != nil {
		return 0, nil, err
	}
	items := make([]dto.NotificationInfo, 0, len(notifications))
	for i := range notifications {
		items = append(items, toNotificationInfo(&notifications[i]))
	}
	return total, items, nil
}

func (s *NotificationService) Summary() (*dto.NotificationSummary, error) {
	unread, err := s.notificationRepo.UnreadCount()
	if err != nil {
		return nil, err
	}
	return &dto.NotificationSummary{Unread: unread}, nil
}

func (s *NotificationService) MarkRead(ids []uint) error {
	return s.notificationRepo.MarkRead(ids)
}

func (s *NotificationService) MarkAllRead() error {
	return s.notificationRepo.MarkAllRead()
}

func (s *NotificationService) DeleteRead() error {
	return s.notificationRepo.DeleteRead()
}

func (s *NotificationService) Delete(id uint) error {
	return s.notificationRepo.Delete(id)
}

func CreateNotification(req dto.NotificationCreate) {
	if err := NewINotificationService().Create(req); err != nil {
		// 通知不应影响主流程，调用方无需感知失败。
		return
	}
}

func normalizeNotificationType(t string) string {
	switch t {
	case "success", "warning", "error":
		return t
	default:
		return "info"
	}
}

func toNotificationInfo(notification *model.Notification) dto.NotificationInfo {
	return dto.NotificationInfo{
		ID:        notification.ID,
		Type:      notification.Type,
		Title:     notification.Title,
		Content:   notification.Content,
		Source:    notification.Source,
		TargetURL: notification.TargetURL,
		ReadAt:    notification.ReadAt,
		CreatedAt: notification.CreatedAt,
		UpdatedAt: notification.UpdatedAt,
	}
}
