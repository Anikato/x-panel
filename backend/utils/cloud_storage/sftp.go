package cloud_storage

import (
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"xpanel/utils/checksum"
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
		Timeout:         metadataTimeout,
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
	return c.UploadLogged(src, target, nil)
}

func (c *SFTPClient) UploadLogged(src, target string, logf func(string, ...any)) error {
	localHash, err := checksum.FileSHA256(src)
	if err != nil {
		return fmt.Errorf("hash local backup failed: %w", err)
	}
	return c.uploadVerified(src, target, localHash, logf)
}

func (c *SFTPClient) uploadVerified(src, target, localHash string, logf func(string, ...any)) error {
	store, cleanup, err := c.openStore()
	if err != nil {
		return err
	}
	defer cleanup()
	remoteTarget := path.Join(c.basePath, target)
	return uploadWithIntegrity(src, remoteTarget, localHash, store, logf)
}

type sftpRemoteStore struct {
	sftp *sftp.Client
	ssh  *ssh.Client
}

func (c *SFTPClient) openStore() (*sftpRemoteStore, func(), error) {
	client, conn, err := c.connect()
	if err != nil {
		return nil, nil, err
	}
	return &sftpRemoteStore{sftp: client, ssh: conn}, func() {
		_ = client.Close()
		_ = conn.Close()
	}, nil
}

func (s *sftpRemoteStore) put(src, dest string) error {
	if err := s.sftp.MkdirAll(path.Dir(dest)); err != nil {
		return err
	}
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	dstFile, err := s.sftp.Create(dest)
	if err != nil {
		return err
	}
	defer dstFile.Close()
	_, err = io.Copy(dstFile, srcFile)
	return err
}

func (s *sftpRemoteStore) hash(path string) (string, error) {
	session, err := s.ssh.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()
	output, err := session.CombinedOutput(remoteSHA256Command(path))
	if err != nil {
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return parseRemoteSHA256(string(output))
}

func (s *sftpRemoteStore) rename(oldPath, newPath string) error {
	_ = s.sftp.Remove(newPath)
	if err := s.sftp.PosixRename(oldPath, newPath); err == nil {
		return nil
	}
	return s.sftp.Rename(oldPath, newPath)
}

func (s *sftpRemoteStore) remove(path string) error {
	err := s.sftp.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (c *SFTPClient) Download(src, target string) error {
	return retryStorageOp(func() error {
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
	})
}

func (c *SFTPClient) Delete(path string) error {
	return retryStorageOp(func() error {
		client, conn, err := c.connect()
		if err != nil {
			return err
		}
		defer conn.Close()
		defer client.Close()
		return client.Remove(filepath.Join(c.basePath, path))
	})
}

func (c *SFTPClient) ListObjects(prefix string) ([]string, error) {
	var entries []os.FileInfo
	err := retryStorageOp(func() error {
		client, conn, err := c.connect()
		if err != nil {
			return err
		}
		defer conn.Close()
		defer client.Close()

		entries, err = client.ReadDir(filepath.Join(c.basePath, prefix))
		return err
	})
	if err != nil {
		return nil, err
	}
	var result []string
	for _, e := range entries {
		result = append(result, filepath.ToSlash(filepath.Join(prefix, e.Name())))
	}
	return result, nil
}
