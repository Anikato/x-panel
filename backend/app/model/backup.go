package model

type BackupAccount struct {
	BaseModel
	Name       string `gorm:"not null" json:"name"`
	Type       string `gorm:"not null" json:"type"` // local / s3 / sftp / webdav
	Bucket     string `json:"bucket"`
	AccessKey  string `json:"accessKey"`
	Credential string `json:"credential"`
	BackupPath string `json:"backupPath"`
	Vars       string `json:"vars"` // JSON: region, endpoint, etc.
}

type BackupRecord struct {
	BaseModel
	CronjobID uint   `gorm:"index" json:"cronjobID"`
	Type      string `json:"type"` // website / database / directory
	Name      string `json:"name"`
	AccountID uint   `json:"accountID"`
	FileName  string `json:"fileName"`
	FileDir   string `json:"fileDir"`
	Size      int64  `json:"size"`
	Status    string `json:"status"` // success / failed
	Message   string `json:"message"`
}
