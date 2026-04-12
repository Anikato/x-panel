package dto

import "time"

// MonitorSearch 历史监控数据查询请求
type MonitorSearch struct {
	Param     string    `json:"param" validate:"required,oneof=all cpu memory load io network"`
	IO        string    `json:"io"`
	Network   string    `json:"network"`
	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime"`
}

// MonitorData 历史监控数据响应
type MonitorData struct {
	Param string        `json:"param"`
	Date  []time.Time   `json:"date"`
	Value []interface{} `json:"value"`
}

// MonitorSetting 监控设置
type MonitorSetting struct {
	MonitorStatus    string `json:"monitorStatus"`
	MonitorStoreDays string `json:"monitorStoreDays"`
	MonitorInterval  string `json:"monitorInterval"`
	DefaultNetwork   string `json:"defaultNetwork"`
	DefaultIO        string `json:"defaultIO"`
}

// MonitorSettingUpdate 监控设置更新
type MonitorSettingUpdate struct {
	Key   string `json:"key" validate:"required,oneof=MonitorStatus MonitorStoreDays MonitorInterval DefaultNetwork DefaultIO"`
	Value string `json:"value" validate:"required"`
}
