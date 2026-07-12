package service

import (
	"errors"
	"fmt"
	"os"

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

func findCertificateConsumerTargets(certIDs []uint) (certificateConsumerTargets, error) {
	if len(certIDs) == 0 {
		return certificateConsumerTargets{}, nil
	}
	var targets certificateConsumerTargets
	var count int64
	if err := global.DB.Model(&model.Website{}).
		Where("certificate_id IN ? AND ssl_enable = ? AND status = ?", certIDs, true, "running").
		Count(&count).Error; err != nil {
		return targets, err
	}
	targets.Nginx = count > 0
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
