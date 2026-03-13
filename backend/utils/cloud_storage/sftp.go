package cloud_storage

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type SFTPClient struct {
	address  string
	username string
	password string
	basePath string
}

func NewSFTPClient(address, username, password, basePath string) (*SFTPClient, error) {
	return &SFTPClient{address: address, username: username, password: password, basePath: basePath}, nil
}

func (c *SFTPClient) connect() (*sftp.Client, *ssh.Client, error) {
	config := &ssh.ClientConfig{
		User:            c.username,
		Auth:            []ssh.AuthMethod{ssh.Password(c.password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	host := c.address
	if _, _, err := net.SplitHostPort(host); err != nil {
		host = host + ":22"
	}
	conn, err := ssh.Dial("tcp", host, config)
	if err != nil {
		return nil, nil, fmt.Errorf("ssh dial failed: %w", err)
	}
	client, err := sftp.NewClient(conn)
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("sftp new client failed: %w", err)
	}
	return client, conn, nil
}

func (c *SFTPClient) Upload(src, target string) error {
	client, conn, err := c.connect()
	if err != nil {
		return err
	}
	defer conn.Close()
	defer client.Close()

	remotePath := filepath.Join(c.basePath, target)
	client.MkdirAll(filepath.Dir(remotePath))

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := client.Create(remotePath)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

func (c *SFTPClient) Download(src, target string) error {
	client, conn, err := c.connect()
	if err != nil {
		return err
	}
	defer conn.Close()
	defer client.Close()

	remotePath := filepath.Join(c.basePath, src)
	srcFile, err := client.Open(remotePath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

func (c *SFTPClient) Delete(path string) error {
	client, conn, err := c.connect()
	if err != nil {
		return err
	}
	defer conn.Close()
	defer client.Close()
	return client.Remove(filepath.Join(c.basePath, path))
}

func (c *SFTPClient) ListObjects(prefix string) ([]string, error) {
	client, conn, err := c.connect()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	defer client.Close()

	entries, err := client.ReadDir(filepath.Join(c.basePath, prefix))
	if err != nil {
		return nil, err
	}
	var result []string
	for _, e := range entries {
		result = append(result, e.Name())
	}
	return result, nil
}
