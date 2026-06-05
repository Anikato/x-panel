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
	address    string
	username   string
	secret     string
	basePath   string
	authMode   string
	passPhrase string
}

func NewSFTPClient(address, username, secret, basePath, authMode, passPhrase string) (*SFTPClient, error) {
	if authMode == "" {
		authMode = "password"
	}
	return &SFTPClient{address: address, username: username, secret: secret, basePath: basePath, authMode: authMode, passPhrase: passPhrase}, nil
}

func (c *SFTPClient) connect() (*sftp.Client, *ssh.Client, error) {
	auth, err := c.authMethods()
	if err != nil {
		return nil, nil, err
	}
	config := &ssh.ClientConfig{
		User:            c.username,
		Auth:            auth,
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

func (c *SFTPClient) authMethods() ([]ssh.AuthMethod, error) {
	if c.authMode == "key" {
		var signer ssh.Signer
		var err error
		if c.passPhrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(c.secret), []byte(c.passPhrase))
		} else {
			signer, err = ssh.ParsePrivateKey([]byte(c.secret))
		}
		if err != nil {
			return nil, fmt.Errorf("parse private key failed: %w", err)
		}
		return []ssh.AuthMethod{ssh.PublicKeys(signer)}, nil
	}
	return []ssh.AuthMethod{ssh.Password(c.secret)}, nil
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
