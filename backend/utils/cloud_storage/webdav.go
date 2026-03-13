package cloud_storage

import (
	"io"
	"os"
	"path"

	"github.com/studio-b12/gowebdav"
)

type WebDAVClient struct {
	client   *gowebdav.Client
	basePath string
}

func NewWebDAVClient(endpoint, username, password, basePath string) (*WebDAVClient, error) {
	client := gowebdav.NewClient(endpoint, username, password)
	if err := client.Connect(); err != nil {
		return nil, err
	}
	return &WebDAVClient{client: client, basePath: basePath}, nil
}

func (c *WebDAVClient) Upload(src, target string) error {
	remotePath := path.Join(c.basePath, target)
	c.client.MkdirAll(path.Dir(remotePath), 0755)

	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()
	return c.client.WriteStream(remotePath, file, 0644)
}

func (c *WebDAVClient) Download(src, target string) error {
	remotePath := path.Join(c.basePath, src)
	reader, err := c.client.ReadStream(remotePath)
	if err != nil {
		return err
	}
	defer reader.Close()

	file, err := os.Create(target)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(file, reader)
	return err
}

func (c *WebDAVClient) Delete(filePath string) error {
	return c.client.Remove(path.Join(c.basePath, filePath))
}

func (c *WebDAVClient) ListObjects(prefix string) ([]string, error) {
	files, err := c.client.ReadDir(path.Join(c.basePath, prefix))
	if err != nil {
		return nil, err
	}
	var result []string
	for _, f := range files {
		result = append(result, f.Name())
	}
	return result, nil
}
