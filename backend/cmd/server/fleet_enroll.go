package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"xpanel/app/model"
	"xpanel/global"

	"gorm.io/gorm"
)

const fleetEnrollmentTokenEnv = "XPANEL_FLEET_ENROLLMENT_TOKEN"

func parseFleetEnrollmentToken(args []string, getenv func(string) string) (string, error) {
	fs := flag.NewFlagSet("fleet-enroll", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	flagToken := fs.String("token", "", "Fleet Center 一次性 Enrollment Token")
	if err := fs.Parse(args); err != nil {
		return "", err
	}
	if fs.NArg() != 0 {
		return "", errors.New("fleet-enroll 不接受位置参数")
	}

	token := strings.TrimSpace(*flagToken)
	if token == "" {
		token = strings.TrimSpace(getenv(fleetEnrollmentTokenEnv))
	}
	if len(token) < 32 {
		return "", errors.New("Enrollment Token 缺失或长度不足")
	}
	return token, nil
}

func runFleetEnroll(args []string) {
	token, err := parseFleetEnrollmentToken(args, os.Getenv)
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		fmt.Fprintf(os.Stderr, "用法: %s=... xpanel fleet-enroll\n", fleetEnrollmentTokenEnv)
		os.Exit(1)
	}

	initializeOfflineDatabase()

	if err := saveFleetEnrollmentToken(global.DB, token); err != nil {
		fmt.Fprintln(os.Stderr, "错误: Enrollment Token 写入失败")
		os.Exit(1)
	}

	fmt.Println("✓ Fleet Enrollment Token 已安全写入，首次注册成功后会自动清除")
}

func saveFleetEnrollmentToken(db *gorm.DB, token string) error {
	if global.CREDENTIALS == nil {
		return errors.New("credential protector is unavailable")
	}
	protected, err := global.CREDENTIALS.Protect("settings.FleetEnrollmentToken", token)
	if err != nil {
		return fmt.Errorf("protect Fleet Enrollment Token: %w", err)
	}
	var setting model.Setting
	return db.
		Where(&model.Setting{Key: "FleetEnrollmentToken"}).
		Assign(model.Setting{Value: protected}).
		FirstOrCreate(&setting).Error
}
