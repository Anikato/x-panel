package model

import "time"

type Cronjob struct {
	BaseModel
	Name           string `gorm:"not null" json:"name"`
	Type           string `gorm:"not null" json:"type"` // shell / website / database / directory / curl
	Spec           string `gorm:"not null" json:"spec"`
	Status         string `gorm:"default:Enable" json:"status"`
	EntryID        int    `json:"entryID"`
	Script         string `json:"script"`
	URL            string `json:"url"`
	Website        string `json:"website"`
	DBType         string `json:"dbType"`
	DBName         string `json:"dbName"`
	SourceDir      string `json:"sourceDir"`
	TargetAccountID uint  `json:"targetAccountID"`
	RetainCopies   uint   `gorm:"default:7" json:"retainCopies"`
	ExclusionRules string `json:"exclusionRules"`
}

type CronjobRecord struct {
	BaseModel
	CronjobID uint      `gorm:"index" json:"cronjobID"`
	StartTime time.Time `json:"startTime"`
	Duration  float64   `json:"duration"`
	Status    string    `json:"status"` // success / failed
	Message   string    `json:"message"`
	File      string    `json:"file"`
}
