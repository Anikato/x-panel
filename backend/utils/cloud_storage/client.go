package cloud_storage

import (
	"encoding/json"
	"fmt"
	"time"
)

const (
	metadataTimeout = 60 * time.Second
	transferTimeout = 2 * time.Hour
)

type CloudStorageClient interface {
	Upload(src, target string) error
	Download(src, target string) error
	Delete(path string) error
	ListObjects(prefix string) ([]string, error)
}

type Vars struct {
	Region     string `json:"region"`
	Endpoint   string `json:"endpoint"`
	AuthMode   string `json:"authMode"`
	PassPhrase string `json:"passPhrase"`
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
		return NewSFTPClient(vars.Endpoint, accessKey, credential, backupPath, vars.AuthMode, vars.PassPhrase)
	case "webdav":
		return NewWebDAVClient(vars.Endpoint, accessKey, credential, backupPath)
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", accountType)
	}
}

func retryStorageOp(fn func() error) error {
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Second)
		}
		if err := fn(); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	return lastErr
}
