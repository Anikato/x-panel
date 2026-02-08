package model

// LoginLog 登录日志
type LoginLog struct {
	BaseModel
	IP      string `json:"ip" gorm:"type:varchar(64)"`
	Agent   string `json:"agent" gorm:"type:varchar(512)"`
	Status  string `json:"status" gorm:"type:varchar(64)"`
	Message string `json:"message" gorm:"type:varchar(512)"`
}

// OperationLog 操作日志
type OperationLog struct {
	BaseModel
	Group    string `json:"group" gorm:"type:varchar(64)"`
	Source   string `json:"source" gorm:"type:varchar(64)"`
	Action   string `json:"action" gorm:"type:varchar(64)"`
	IP       string `json:"ip" gorm:"type:varchar(64)"`
	Path     string `json:"path" gorm:"type:varchar(256)"`
	Method   string `json:"method" gorm:"type:varchar(16)"`
	Body     string `json:"body" gorm:"type:text"`
	Status   string `json:"status" gorm:"type:varchar(64)"`
	Message  string `json:"message" gorm:"type:text"`
}
