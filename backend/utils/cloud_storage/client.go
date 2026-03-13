package cloud_storage

import (
	"encoding/json"
	"fmt"
)

type CloudStorageClient interface {
	Upload(src, target string) error
	Download(src, target string) error
	Delete(path string) error
	ListObjects(prefix string) ([]string, error)
}

type Vars struct {
	Region   string `json:"region"`
	Endpoint string `json:"endpoint"`
}

func NewClient(accountType, bucket, accessKey, credential, backupPath, varsJSON string) (CloudStorageClient, error) {
	var vars Vars
	if varsJSON != "" {
		_ = json.Unmarshal([]byte(varsJSON), &vars)
	}

	switch accountType {
	case "local":
		return NewLocalClient(backupPath), nil
	case "s3":
		return NewS3Client(vars.Region, vars.Endpoint, bucket, accessKey, credential, backupPath)
	case "sftp":
		return NewSFTPClient(vars.Endpoint, accessKey, credential, backupPath)
	case "webdav":
		return NewWebDAVClient(vars.Endpoint, accessKey, credential, backupPath)
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", accountType)
	}
}
