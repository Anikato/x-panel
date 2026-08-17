package model

type ComposeProject struct {
	BaseModel
	Name   string `gorm:"uniqueIndex;not null" json:"name"`
	Path   string `gorm:"not null" json:"path"`
	Source string `gorm:"not null" json:"source"` // created / attached
}
