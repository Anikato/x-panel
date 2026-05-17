package service

import (
	"encoding/json"
	"strings"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/repo"
)

const notificationPreferenceKey = "NotificationPreferences"

type INotificationService interface {
	Create(req dto.NotificationCreate) error
	SearchWithPage(req dto.NotificationSearch) (int64, []dto.NotificationInfo, error)
	Recent(limit int) ([]dto.NotificationInfo, error)
	Summary() (*dto.NotificationSummary, error)
	GetPreference() (*dto.NotificationPreference, error)
	UpdatePreference(req dto.NotificationPreference) error
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
	event := strings.TrimSpace(req.Event)
	pref, _ := s.GetPreference()
	rule := notificationRuleFor(pref, event)
	if !rule.Center {
		return nil
	}
	notification := &model.Notification{
		Type:      normalizeNotificationType(req.Type),
		Event:     event,
		Title:     strings.TrimSpace(req.Title),
		Content:   strings.TrimSpace(req.Content),
		Source:    strings.TrimSpace(req.Source),
		TargetURL: strings.TrimSpace(req.TargetURL),
		ShowBadge: rule.Badge,
		Popup:     rule.Popup,
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
		repo.WithNotificationEvent(req.Event),
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

func (s *NotificationService) Recent(limit int) ([]dto.NotificationInfo, error) {
	notifications, err := s.notificationRepo.Recent(limit)
	if err != nil {
		return nil, err
	}
	items := make([]dto.NotificationInfo, 0, len(notifications))
	for i := range notifications {
		items = append(items, toNotificationInfo(&notifications[i]))
	}
	return items, nil
}

func (s *NotificationService) Summary() (*dto.NotificationSummary, error) {
	unread, err := s.notificationRepo.UnreadCount()
	if err != nil {
		return nil, err
	}
	return &dto.NotificationSummary{Unread: unread}, nil
}

func (s *NotificationService) GetPreference() (*dto.NotificationPreference, error) {
	raw, err := settingRepo.GetValueByKey(notificationPreferenceKey)
	if err != nil || strings.TrimSpace(raw) == "" {
		pref := defaultNotificationPreference()
		return &pref, nil
	}
	pref := defaultNotificationPreference()
	if err := json.Unmarshal([]byte(raw), &pref); err != nil {
		return nil, err
	}
	normalizeNotificationPreference(&pref)
	return &pref, nil
}

func (s *NotificationService) UpdatePreference(req dto.NotificationPreference) error {
	normalizeNotificationPreference(&req)
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	return settingRepo.CreateOrUpdate(notificationPreferenceKey, string(data))
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

func defaultNotificationPreference() dto.NotificationPreference {
	return dto.NotificationPreference{
		Defaults: dto.NotificationPreferenceRule{Center: true, Badge: true, Popup: false},
		Events: map[string]dto.NotificationPreferenceRule{
			"file.upload.completed": {Center: true, Badge: false, Popup: false},
			"file.task.failed":      {Center: true, Badge: true, Popup: true},
			"database.task.failed":  {Center: true, Badge: true, Popup: true},
			"cronjob.failed":        {Center: true, Badge: true, Popup: true},
			"operation.failed":      {Center: true, Badge: true, Popup: true},
			"system.log.error":      {Center: true, Badge: true, Popup: false},
		},
	}
}

func normalizeNotificationPreference(pref *dto.NotificationPreference) {
	if pref.Defaults == (dto.NotificationPreferenceRule{}) {
		pref.Defaults = dto.NotificationPreferenceRule{Center: true, Badge: true, Popup: false}
	}
	if pref.Events == nil {
		pref.Events = map[string]dto.NotificationPreferenceRule{}
	}
	defaults := defaultNotificationPreference()
	for event, rule := range defaults.Events {
		if _, ok := pref.Events[event]; !ok {
			pref.Events[event] = rule
		}
	}
}

func notificationRuleFor(pref *dto.NotificationPreference, event string) dto.NotificationPreferenceRule {
	if pref == nil {
		defaults := defaultNotificationPreference()
		pref = &defaults
	}
	normalizeNotificationPreference(pref)
	if event != "" {
		if rule, ok := pref.Events[event]; ok {
			return rule
		}
	}
	return pref.Defaults
}

func toNotificationInfo(notification *model.Notification) dto.NotificationInfo {
	return dto.NotificationInfo{
		ID:        notification.ID,
		Type:      notification.Type,
		Event:     notification.Event,
		Title:     notification.Title,
		Content:   notification.Content,
		Source:    notification.Source,
		TargetURL: notification.TargetURL,
		ShowBadge: notification.ShowBadge,
		Popup:     notification.Popup,
		ReadAt:    notification.ReadAt,
		CreatedAt: notification.CreatedAt,
		UpdatedAt: notification.UpdatedAt,
	}
}
