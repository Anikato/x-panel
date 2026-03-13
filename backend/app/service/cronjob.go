package service

import (
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/buserr"
	"xpanel/constant"
	"xpanel/global"

	"github.com/robfig/cron/v3"
)

type ICronjobService interface {
	Create(req dto.CronjobCreate) error
	Update(req dto.CronjobUpdate) error
	Delete(id uint) error
	Get(id uint) (*dto.CronjobInfo, error)
	SearchWithPage(req dto.CronjobSearch) (int64, []dto.CronjobInfo, error)
	UpdateStatus(id uint, status string) error
	HandleOnce(id uint) error
	SearchRecords(req dto.CronjobRecordSearch) (int64, []dto.CronjobRecordInfo, error)
	StartAllJobs()
}

func NewICronjobService() ICronjobService {
	return &CronjobService{
		cronjobRepo: repo.NewICronjobRepo(),
	}
}

type CronjobService struct {
	cronjobRepo repo.ICronjobRepo
}

func (s *CronjobService) Create(req dto.CronjobCreate) error {
	job := &model.Cronjob{
		Name:           req.Name,
		Type:           req.Type,
		Spec:           req.Spec,
		Status:         constant.StatusEnable,
		Script:         req.Script,
		URL:            req.URL,
		Website:        req.Website,
		DBType:         req.DBType,
		DBName:         req.DBName,
		SourceDir:      req.SourceDir,
		TargetAccountID: req.TargetAccountID,
		RetainCopies:   req.RetainCopies,
		ExclusionRules: req.ExclusionRules,
	}
	if err := s.cronjobRepo.Create(job); err != nil {
		return err
	}
	if err := s.addCronJob(job); err != nil {
		return err
	}
	return nil
}

func (s *CronjobService) Update(req dto.CronjobUpdate) error {
	job, err := s.cronjobRepo.Get(req.ID)
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	s.removeCronJob(job)
	fields := map[string]interface{}{
		"name":              req.Name,
		"type":              req.Type,
		"spec":              req.Spec,
		"script":            req.Script,
		"url":               req.URL,
		"website":           req.Website,
		"db_type":           req.DBType,
		"db_name":           req.DBName,
		"source_dir":        req.SourceDir,
		"target_account_id": req.TargetAccountID,
		"retain_copies":     req.RetainCopies,
		"exclusion_rules":   req.ExclusionRules,
	}
	if err := s.cronjobRepo.Update(req.ID, fields); err != nil {
		return err
	}
	if job.Status == constant.StatusEnable {
		updated, _ := s.cronjobRepo.Get(req.ID)
		if updated != nil {
			_ = s.addCronJob(updated)
		}
	}
	return nil
}

func (s *CronjobService) Delete(id uint) error {
	job, err := s.cronjobRepo.Get(id)
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	s.removeCronJob(job)
	_ = s.cronjobRepo.DeleteRecordByCronjobID(id)
	return s.cronjobRepo.Delete(id)
}

func (s *CronjobService) Get(id uint) (*dto.CronjobInfo, error) {
	job, err := s.cronjobRepo.Get(id)
	if err != nil {
		return nil, buserr.New(constant.ErrRecordNotFound)
	}
	return toCronjobInfo(job), nil
}

func (s *CronjobService) SearchWithPage(req dto.CronjobSearch) (int64, []dto.CronjobInfo, error) {
	opts := []repo.DBOption{
		repo.WithCronjobType(req.Type),
		repo.WithCronjobStatus(req.Status),
	}
	if req.Info != "" {
		opts = append(opts, repo.WithLikeName(req.Info))
	}
	total, jobs, err := s.cronjobRepo.Page(req.Page, req.PageSize, opts...)
	if err != nil {
		return 0, nil, err
	}
	var items []dto.CronjobInfo
	for _, j := range jobs {
		items = append(items, *toCronjobInfo(&j))
	}
	return total, items, nil
}

func (s *CronjobService) UpdateStatus(id uint, status string) error {
	job, err := s.cronjobRepo.Get(id)
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	if status == constant.StatusEnable {
		_ = s.addCronJob(job)
	} else {
		s.removeCronJob(job)
	}
	return s.cronjobRepo.Update(id, map[string]interface{}{"status": status})
}

func (s *CronjobService) HandleOnce(id uint) error {
	job, err := s.cronjobRepo.Get(id)
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	go s.executeJob(job)
	return nil
}

func (s *CronjobService) SearchRecords(req dto.CronjobRecordSearch) (int64, []dto.CronjobRecordInfo, error) {
	opts := []repo.DBOption{
		repo.WithCronjobID(req.CronjobID),
		repo.WithRecordStatus(req.Status),
	}
	total, records, err := s.cronjobRepo.PageRecord(req.Page, req.PageSize, opts...)
	if err != nil {
		return 0, nil, err
	}
	var items []dto.CronjobRecordInfo
	for _, r := range records {
		items = append(items, dto.CronjobRecordInfo{
			ID:        r.ID,
			CronjobID: r.CronjobID,
			StartTime: r.StartTime,
			Duration:  r.Duration,
			Status:    r.Status,
			Message:   r.Message,
			File:      r.File,
		})
	}
	return total, items, nil
}

func (s *CronjobService) StartAllJobs() {
	jobs, err := s.cronjobRepo.List(repo.WithCronjobStatus(constant.StatusEnable))
	if err != nil {
		global.LOG.Errorf("load cronjobs failed: %v", err)
		return
	}
	for i := range jobs {
		if err := s.addCronJob(&jobs[i]); err != nil {
			global.LOG.Errorf("add cronjob [%s] failed: %v", jobs[i].Name, err)
		}
	}
}

func (s *CronjobService) addCronJob(job *model.Cronjob) error {
	if global.CRON == nil {
		return nil
	}
	entryID, err := global.CRON.AddFunc(job.Spec, func() {
		s.executeJob(job)
	})
	if err != nil {
		return err
	}
	return s.cronjobRepo.Update(job.ID, map[string]interface{}{"entry_id": int(entryID)})
}

func (s *CronjobService) removeCronJob(job *model.Cronjob) {
	if global.CRON != nil && job.EntryID > 0 {
		global.CRON.Remove(cron.EntryID(job.EntryID))
	}
}

func (s *CronjobService) executeJob(job *model.Cronjob) {
	start := time.Now()
	var msg string
	var status string

	switch job.Type {
	case "shell":
		msg, status = s.execShell(job)
	case "curl":
		msg, status = s.execCurl(job)
	default:
		msg = fmt.Sprintf("backup type [%s] will be handled by backup system", job.Type)
		status = constant.StatusSuccess
	}

	duration := time.Since(start).Seconds()
	record := &model.CronjobRecord{
		CronjobID: job.ID,
		StartTime: start,
		Duration:  duration,
		Status:    status,
		Message:   msg,
	}
	if err := s.cronjobRepo.CreateRecord(record); err != nil {
		global.LOG.Errorf("save cronjob record failed: %v", err)
	}
	if job.RetainCopies > 0 {
		_ = s.cronjobRepo.CleanRecords(job.ID, int(job.RetainCopies))
	}
}

func (s *CronjobService) execShell(job *model.Cronjob) (string, string) {
	cmd := exec.Command("bash", "-c", job.Script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output) + "\n" + err.Error(), constant.StatusFailed
	}
	result := strings.TrimSpace(string(output))
	if len(result) > 10000 {
		result = result[:10000] + "\n...(truncated)"
	}
	return result, constant.StatusSuccess
}

func (s *CronjobService) execCurl(job *model.Cronjob) (string, string) {
	resp, err := http.Get(job.URL)
	if err != nil {
		return err.Error(), constant.StatusFailed
	}
	defer resp.Body.Close()
	return fmt.Sprintf("HTTP %d", resp.StatusCode), constant.StatusSuccess
}

func toCronjobInfo(j *model.Cronjob) *dto.CronjobInfo {
	return &dto.CronjobInfo{
		ID:              j.ID,
		CreatedAt:       j.CreatedAt,
		Name:            j.Name,
		Type:            j.Type,
		Spec:            j.Spec,
		Status:          j.Status,
		EntryID:         j.EntryID,
		Script:          j.Script,
		URL:             j.URL,
		Website:         j.Website,
		DBType:          j.DBType,
		DBName:          j.DBName,
		SourceDir:       j.SourceDir,
		TargetAccountID: j.TargetAccountID,
		RetainCopies:    j.RetainCopies,
		ExclusionRules:  j.ExclusionRules,
	}
}
