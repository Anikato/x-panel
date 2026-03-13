package dto

import "time"

type CronjobCreate struct {
	Name           string `json:"name" binding:"required"`
	Type           string `json:"type" binding:"required"`
	Spec           string `json:"spec" binding:"required"`
	Script         string `json:"script"`
	URL            string `json:"url"`
	Website        string `json:"website"`
	DBType         string `json:"dbType"`
	DBName         string `json:"dbName"`
	SourceDir      string `json:"sourceDir"`
	TargetAccountID uint  `json:"targetAccountID"`
	RetainCopies   uint   `json:"retainCopies"`
	ExclusionRules string `json:"exclusionRules"`
}

type CronjobUpdate struct {
	ID             uint   `json:"id" binding:"required"`
	Name           string `json:"name" binding:"required"`
	Type           string `json:"type" binding:"required"`
	Spec           string `json:"spec" binding:"required"`
	Script         string `json:"script"`
	URL            string `json:"url"`
	Website        string `json:"website"`
	DBType         string `json:"dbType"`
	DBName         string `json:"dbName"`
	SourceDir      string `json:"sourceDir"`
	TargetAccountID uint  `json:"targetAccountID"`
	RetainCopies   uint   `json:"retainCopies"`
	ExclusionRules string `json:"exclusionRules"`
}

type CronjobSearch struct {
	PageInfo
	Type   string `json:"type"`
	Status string `json:"status"`
	Info   string `json:"info"`
}

type CronjobInfo struct {
	ID             uint      `json:"id"`
	CreatedAt      time.Time `json:"createdAt"`
	Name           string    `json:"name"`
	Type           string    `json:"type"`
	Spec           string    `json:"spec"`
	Status         string    `json:"status"`
	EntryID        int       `json:"entryID"`
	Script         string    `json:"script"`
	URL            string    `json:"url"`
	Website        string    `json:"website"`
	DBType         string    `json:"dbType"`
	DBName         string    `json:"dbName"`
	SourceDir      string    `json:"sourceDir"`
	TargetAccountID uint     `json:"targetAccountID"`
	RetainCopies   uint      `json:"retainCopies"`
	ExclusionRules string    `json:"exclusionRules"`
}

type CronjobRecordSearch struct {
	PageInfo
	CronjobID uint   `json:"cronjobID" binding:"required"`
	Status    string `json:"status"`
}

type CronjobRecordInfo struct {
	ID        uint      `json:"id"`
	CronjobID uint      `json:"cronjobID"`
	StartTime time.Time `json:"startTime"`
	Duration  float64   `json:"duration"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	File      string    `json:"file"`
}
