package dto

import "time"

type NotificationCreate struct {
	Type      string `json:"type" binding:"required,oneof=info success warning error"`
	Title     string `json:"title" binding:"required"`
	Content   string `json:"content"`
	Source    string `json:"source"`
	TargetURL string `json:"targetUrl"`
}

type NotificationSearch struct {
	PageInfo
	Status string `json:"status"` // unread / read / all
	Type   string `json:"type"`
	Source string `json:"source"`
	Info   string `json:"info"`
}

type NotificationInfo struct {
	ID        uint       `json:"id"`
	Type      string     `json:"type"`
	Title     string     `json:"title"`
	Content   string     `json:"content"`
	Source    string     `json:"source"`
	TargetURL string     `json:"targetUrl"`
	ReadAt    *time.Time `json:"readAt,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

type NotificationSummary struct {
	Unread int64 `json:"unread"`
}
