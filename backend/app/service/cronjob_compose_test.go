package service

import (
	"fmt"
	"strings"
	"testing"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/constant"
)

func TestValidateComposeJob(t *testing.T) {
	s := &CronjobService{}
	err := s.validateJobConfig(&model.Cronjob{Type: "compose", Spec: "0 3 * * *"})
	if err == nil {
		t.Fatal("missing compose name should fail")
	}
	err = s.validateJobConfig(&model.Cronjob{
		Type: "compose", Spec: "0 3 * * *", ComposeName: "blog", ComposeOperation: "down",
	})
	if err == nil {
		t.Fatal("illegal operation should fail")
	}
	err = s.validateJobConfig(&model.Cronjob{
		Type: "compose", Spec: "0 3 * * *", ComposeName: "blog", ComposeOperation: "update",
	})
	if err != nil {
		t.Fatalf("valid compose job: %v", err)
	}
	err = s.validateJobConfig(&model.Cronjob{
		Type: "compose", Spec: "0 3 * * *", ComposeName: "blog", ComposeOperation: "pull",
	})
	if err != nil {
		t.Fatalf("valid pull job: %v", err)
	}
}

func TestExecComposeUsesOperate(t *testing.T) {
	var got dto.ComposeOperate
	s := &CronjobService{
		operateCompose: func(req dto.ComposeOperate) error {
			got = req
			return nil
		},
	}
	msg, status := s.execCompose(&model.Cronjob{ComposeName: "blog", ComposeOperation: "pull"})
	if status != constant.StatusSuccess || got.Name != "blog" || got.Operation != "pull" {
		t.Fatalf("msg=%q status=%q got=%#v", msg, status, got)
	}
}

func TestExecComposeMissingProject(t *testing.T) {
	s := &CronjobService{
		operateCompose: func(req dto.ComposeOperate) error {
			return fmt.Errorf("compose project not found: %s", req.Name)
		},
	}
	msg, status := s.execCompose(&model.Cronjob{ComposeName: "gone", ComposeOperation: "update"})
	if status != constant.StatusFailed || !strings.Contains(msg, "not found") {
		t.Fatalf("msg=%q status=%q", msg, status)
	}
}
