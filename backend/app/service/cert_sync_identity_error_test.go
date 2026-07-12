package service

import (
	"errors"
	"testing"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/repo"
)

func TestFindLocalCertDoesNotTreatDatabaseErrorAsMissing(t *testing.T) {
	certRepo := &certificateRepoGetError{err: errors.New("database unavailable")}
	svc := &CertSourceService{certRepo: certRepo}
	_, found, err := svc.findLocalCert(dto.CertServerItem{
		LineageUID: "550e8400-e29b-41d4-a716-446655440000",
	}, 1)
	if err == nil {
		t.Fatal("database lookup failure must not be treated as an absent certificate")
	}
	if found {
		t.Fatal("database lookup failure must not report a local certificate")
	}
	if certRepo.getListCalled {
		t.Fatal("legacy adoption must not run after a database lookup failure")
	}
}

type certificateRepoGetError struct {
	err           error
	getListCalled bool
}

func (r *certificateRepoGetError) Page(int, int, ...repo.DBOption) (int64, []model.Certificate, error) {
	return 0, nil, r.err
}

func (r *certificateRepoGetError) GetList(...repo.DBOption) ([]model.Certificate, error) {
	r.getListCalled = true
	return nil, r.err
}

func (r *certificateRepoGetError) Get(...repo.DBOption) (model.Certificate, error) {
	return model.Certificate{}, r.err
}

func (r *certificateRepoGetError) Create(*model.Certificate) error { return r.err }

func (r *certificateRepoGetError) Update(uint, map[string]interface{}) error { return r.err }

func (r *certificateRepoGetError) Save(*model.Certificate) error { return r.err }

func (r *certificateRepoGetError) Delete(...repo.DBOption) error { return r.err }
