package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	"xpanel/app/model"
	"xpanel/global"
)

type certificateConsumerTargets struct {
	Nginx   bool
	HAProxy bool
	GOST    bool
}

type certificateConsumerRefreshActions struct {
	ReloadNginx   func() error
	VerifyNginx   func() error
	ReloadHAProxy func() error
	ReloadGOST    func() error
}

func refreshUpdatedCertificateConsumers(certIDs []uint) error {
	targets, err := findCertificateConsumerTargets(certIDs)
	if err != nil {
		return err
	}
	return refreshCertificateConsumers(targets, certificateConsumerRefreshActions{
		ReloadNginx: reloadNginxGlobal,
		VerifyNginx: func() error {
			return verifyUpdatedNginxCertificateConsumers(certIDs)
		},
		ReloadHAProxy: func() error {
			if !isHAProxyInstalled() {
				return nil
			}
			return NewIHAProxyService().ApplyChange("证书同步更新", "certificate-sync")
		},
		ReloadGOST: func() error {
			if _, err := os.Stat(gostBinaryPath); os.IsNotExist(err) {
				return nil
			}
			return NewIGostService().SyncAll()
		},
	})
}

func verifyUpdatedNginxCertificateConsumers(certIDs []uint) error {
	if len(certIDs) == 0 {
		return nil
	}
	var sites []model.Website
	if err := global.DB.Where("certificate_id IN ? AND ssl_enable = ? AND status = ?", certIDs, true, "running").
		Find(&sites).Error; err != nil {
		return err
	}
	if len(sites) == 0 {
		return nil
	}
	websiteService := NewIWebsiteService().(*WebsiteService)
	ops := websiteService.effectiveCertificateHealthOps()
	semaphore := make(chan struct{}, 5)
	errorsBySite := make(chan error, len(sites))
	var waitGroup sync.WaitGroup
	for _, site := range sites {
		site := site
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			if err := websiteService.verifyWebsiteLocalCertificate(context.Background(), site, ops); err != nil {
				errorsBySite <- fmt.Errorf("网站 %s: %w", site.PrimaryDomain, err)
			}
		}()
	}
	waitGroup.Wait()
	close(errorsBySite)
	var errs []error
	for err := range errorsBySite {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func findCertificateConsumerTargets(certIDs []uint) (certificateConsumerTargets, error) {
	if len(certIDs) == 0 {
		return certificateConsumerTargets{}, nil
	}
	var targets certificateConsumerTargets
	if global.CONF.Nginx.IsInstalled() {
		_, targets.Nginx = (&NginxService{}).readPID()
	}
	var count int64
	if err := global.DB.Model(&model.HAProxyLB{}).
		Where("certificate_id IN ? AND enable_ssl = ? AND enabled = ?", certIDs, true, true).
		Count(&count).Error; err != nil {
		return targets, err
	}
	targets.HAProxy = count > 0
	if err := global.DB.Model(&model.GostService{}).
		Where("certificate_id IN ? AND enabled = ? AND custom_cert_path = '' AND custom_key_path = ''", certIDs, true).
		Count(&count).Error; err != nil {
		return targets, err
	}
	targets.GOST = count > 0
	return targets, nil
}

func refreshCertificateConsumers(targets certificateConsumerTargets, actions certificateConsumerRefreshActions) error {
	var errs []error
	if targets.Nginx && actions.ReloadNginx != nil {
		if err := actions.ReloadNginx(); err != nil {
			errs = append(errs, fmt.Errorf("Nginx reload: %w", err))
		} else if actions.VerifyNginx != nil {
			if err := actions.VerifyNginx(); err != nil {
				errs = append(errs, fmt.Errorf("Nginx 证书验证: %w", err))
			}
		}
	}
	if targets.HAProxy && actions.ReloadHAProxy != nil {
		if err := actions.ReloadHAProxy(); err != nil {
			errs = append(errs, fmt.Errorf("HAProxy reload: %w", err))
		}
	}
	if targets.GOST && actions.ReloadGOST != nil {
		if err := actions.ReloadGOST(); err != nil {
			errs = append(errs, fmt.Errorf("GOST sync: %w", err))
		}
	}
	return errors.Join(errs...)
}

func runCertificateSyncPostActions(certIDs []uint, postCommand string, refresh func([]uint) error, runCommand func(string) error) error {
	if len(certIDs) == 0 {
		return nil
	}
	var errs []error
	if refresh != nil {
		if err := refresh(certIDs); err != nil {
			errs = append(errs, err)
		}
	}
	if postCommand != "" && runCommand != nil {
		if err := runCommand(postCommand); err != nil {
			errs = append(errs, fmt.Errorf("同步后命令: %w", err))
		}
	}
	return errors.Join(errs...)
}
