package model

// Setting 面板设置，Key-Value 存储
type Setting struct {
	BaseModel
	Key   string `json:"key" gorm:"type:varchar(256);not null;uniqueIndex"`
	Value string `json:"value" gorm:"type:text"`
	About string `json:"about" gorm:"type:varchar(256)"`
}
