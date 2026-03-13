package cloud_storage

import (
	"io"
	"os"
	"path/filepath"
)

type LocalClient struct {
	basePath string
}

func NewLocalClient(basePath string) *LocalClient {
	if basePath == "" {
		basePath = "/opt/xpanel/backup"
	}
	return &LocalClient{basePath: basePath}
}

func (c *LocalClient) Upload(src, target string) error {
	dst := filepath.Join(c.basePath, target)
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()
	_, err = io.Copy(dstFile, srcFile)
	return err
}

func (c *LocalClient) Download(src, target string) error {
	return c.Upload(filepath.Join(c.basePath, src), target)
}

func (c *LocalClient) Delete(path string) error {
	return os.Remove(filepath.Join(c.basePath, path))
}

func (c *LocalClient) ListObjects(prefix string) ([]string, error) {
	dir := filepath.Join(c.basePath, prefix)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var result []string
	for _, e := range entries {
		result = append(result, e.Name())
	}
	return result, nil
}
