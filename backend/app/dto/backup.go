package dto

import "time"

type BackupAccountCreate struct {
	Name       string `json:"name" binding:"required"`
	Type       string `json:"type" binding:"required"`
	Bucket     string `json:"bucket"`
	AccessKey  string `json:"accessKey"`
	Credential string `json:"credential"`
	BackupPath string `json:"backupPath"`
	Vars       string `json:"vars"`
}

type BackupAccountUpdate struct {
	ID         uint   `json:"id" binding:"required"`
	Name       string `json:"name" binding:"required"`
	Bucket     string `json:"bucket"`
	AccessKey  string `json:"accessKey"`
	Credential string `json:"credential"`
	BackupPath string `json:"backupPath"`
	Vars       string `json:"vars"`
}

type BackupAccountTest struct {
	ID         uint   `json:"id"`
	Type       string `json:"type" binding:"required"`
	Bucket     string `json:"bucket"`
	AccessKey  string `json:"accessKey"`
	Credential string `json:"credential"`
	BackupPath string `json:"backupPath"`
	Vars       string `json:"vars"`
}

type BackupAccountInfo struct {
	ID         uint      `json:"id"`
	CreatedAt  time.Time `json:"createdAt"`
	Name       string    `json:"name"`
	Type       string    `json:"type"`
	Bucket     string    `json:"bucket"`
	BackupPath string    `json:"backupPath"`
	Vars       string    `json:"vars"`
}

type BackupRecordSearch struct {
	PageInfo
	Type      string `json:"type"`
	AccountID uint   `json:"accountID"`
	Name      string `json:"name"`
	Status    string `json:"status"`
}

type BackupRecordInfo struct {
	ID        uint      `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	Type      string    `json:"type"`
	Name      string    `json:"name"`
	AccountID uint      `json:"accountID"`
	CronjobID uint      `json:"cronjobID"`
	FileName  string    `json:"fileName"`
	FileDir   string    `json:"fileDir"`
	Size      int64     `json:"size"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
}

type BackupCreate struct {
	Type      string `json:"type" binding:"required"` // website / database / directory
	Name      string `json:"name" binding:"required"`
	AccountID uint   `json:"accountID" binding:"required"`
	DBType    string `json:"dbType"`
	SourceDir string `json:"sourceDir"`
}

type BackupStorageReq struct {
	AccountID uint   `json:"accountID" binding:"required"`
	Prefix    string `json:"prefix"`
	Path      string `json:"path"`
	Content   string `json:"content"`
}

type BackupStorageObject struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type BackupStorageReadResp struct {
	Content string `json:"content"`
}
