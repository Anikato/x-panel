package server

import (
	"crypto/sha256"
	"crypto/tls"
	"fmt"
	"os"
	"sync"
)

type tlsCertificateFileState struct {
	certSum [32]byte
	keySum  [32]byte
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
	certSum, err := fileSum(r.certPath)
	if err != nil {
		return tlsCertificateFileState{}, err
	}
	keySum, err := fileSum(r.keyPath)
	if err != nil {
		return tlsCertificateFileState{}, err
	}
	return tlsCertificateFileState{certSum: certSum, keySum: keySum}, nil
}

func fileSum(path string) ([32]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return [32]byte{}, err
	}
	return sha256.Sum256(data), nil
}
