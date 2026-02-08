package dto

// PageInfo 分页请求参数
type PageInfo struct {
	Page     int `json:"page" binding:"required,min=1"`
	PageSize int `json:"pageSize" binding:"required,min=1,max=100"`
}

// PageResult 分页响应
type PageResult struct {
	Total int64       `json:"total"`
	Items interface{} `json:"items"`
}

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// SearchWithPage 带搜索的分页请求
type SearchWithPage struct {
	PageInfo
	Info string `json:"info"`
}

// OperateByID 按 ID 操作
type OperateByID struct {
	ID uint `json:"id" binding:"required"`
}

// OperateByIDs 按多个 ID 操作
type OperateByIDs struct {
	IDs []uint `json:"ids" binding:"required"`
}
