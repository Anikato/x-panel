package server

import (
	"crypto/tls"
	"fmt"
	"os"
	"sync"
	"time"
)

type tlsCertificateFileState struct {
	certModTime time.Time
	certSize    int64
	keyModTime  time.Time
	keySize     int64
}

type tlsCertificateReloader struct {
	certPath string
	keyPath  string
	mu       sync.Mutex
	current  *tls.Certificate
	state    tlsCertificateFileState
}

func newTLSCertificateReloader(certPath, keyPath string) (*tlsCertificateReloader, error) {
	reloader := &tlsCertificateReloader{certPath: certPath, keyPath: keyPath}
	if err := reloader.reload(); err != nil {
		return nil, err
	}
	return reloader, nil
}

func (r *tlsCertificateReloader) GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	changed, err := r.filesChanged()
	if err == nil && changed {
		// A certificate/key pair is replaced as two files. Keep serving the last
		// verified pair while an in-progress replacement is not yet loadable.
		_ = r.reload()
	}
	if r.current == nil {
		return nil, fmt.Errorf("TLS certificate is unavailable")
	}
	return r.current, nil
}

func (r *tlsCertificateReloader) reload() error {
	state, err := r.readFileState()
	if err != nil {
		return err
	}
	certificate, err := tls.LoadX509KeyPair(r.certPath, r.keyPath)
	if err != nil {
		return err
	}
	r.current = &certificate
	r.state = state
	return nil
}

func (r *tlsCertificateReloader) filesChanged() (bool, error) {
	state, err := r.readFileState()
	if err != nil {
		return false, err
	}
	return state != r.state, nil
}

func (r *tlsCertificateReloader) readFileState() (tlsCertificateFileState, error) {
	certInfo, err := os.Stat(r.certPath)
	if err != nil {
		return tlsCertificateFileState{}, err
	}
	keyInfo, err := os.Stat(r.keyPath)
	if err != nil {
		return tlsCertificateFileState{}, err
	}
	return tlsCertificateFileState{
		certModTime: certInfo.ModTime(), certSize: certInfo.Size(),
		keyModTime: keyInfo.ModTime(), keySize: keyInfo.Size(),
	}, nil
}
