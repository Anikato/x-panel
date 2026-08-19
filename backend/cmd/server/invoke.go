package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"xpanel/app/service"
	"xpanel/global"
	initViper "xpanel/init/viper"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type invokeRequest struct {
	Capability string          `json:"capability"`
	Payload    json.RawMessage `json:"payload"`
}

type invokeResponse struct {
	OK          bool   `json:"ok"`
	Capability  string `json:"capability"`
	GeneratedAt string `json:"generated_at,omitempty"`
	Data        any    `json:"data,omitempty"`
	Error       string `json:"error,omitempty"`
}

func decodeInvokeRequest(r io.Reader) (invokeRequest, error) {
	var req invokeRequest
	if err := json.NewDecoder(r).Decode(&req); err != nil {
		return invokeRequest{}, err
	}
	return req, nil
}

func dispatchInvoke(req invokeRequest, db *gorm.DB) invokeResponse {
	switch req.Capability {
	case "sites.snapshot":
		if len(req.Payload) > 0 && !bytes.Equal(req.Payload, []byte("{}")) {
			return invokeResponse{Capability: req.Capability, Error: "invalid_request"}
		}
		if db == nil {
			return invokeResponse{Capability: req.Capability, Error: "database_unreadable"}
		}
		snapshot, err := service.BuildSitesSnapshot(db)
		if err != nil {
			return invokeResponse{Capability: req.Capability, Error: "internal"}
		}
		return invokeResponse{
			OK:          true,
			Capability:  req.Capability,
			GeneratedAt: time.Now().UTC().Format(time.RFC3339),
			Data:        snapshot,
		}
	default:
		return invokeResponse{Capability: req.Capability, Error: "unknown_capability"}
	}
}

func encodeInvokeResponse(w io.Writer, resp invokeResponse) error {
	return json.NewEncoder(w).Encode(resp)
}

func openInvokeDatabase(databasePath string) (*gorm.DB, error) {
	databaseURL := (&url.URL{Scheme: "file", Path: databasePath}).String()
	db, err := gorm.Open(sqlite.Open(databaseURL+"?mode=ro"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, classifyInvokeDBError(err)
	}
	if err := db.Exec("PRAGMA query_only = ON").Error; err != nil {
		return nil, classifyInvokeDBError(err)
	}
	if err := db.Exec("PRAGMA busy_timeout = 3000").Error; err != nil {
		return nil, classifyInvokeDBError(err)
	}
	return db, nil
}

func classifyInvokeDBError(err error) error {
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "busy") || strings.Contains(msg, "locked") {
		return fmt.Errorf("database_busy: %w", err)
	}
	return fmt.Errorf("database_unreadable: %w", err)
}

func invokeDBErrorCode(err error) string {
	if err == nil {
		return "internal"
	}
	if strings.HasPrefix(err.Error(), "database_busy") {
		return "database_busy"
	}
	return "database_unreadable"
}

func runInvokeWith(args []string, stdin io.Reader, stdout io.Writer, db *gorm.DB) error {
	if len(args) != 0 {
		return fmt.Errorf("usage: xpanel invoke")
	}
	req, err := decodeInvokeRequest(stdin)
	if err != nil {
		_ = encodeInvokeResponse(stdout, invokeResponse{Error: "invalid_request"})
		return err
	}
	resp := dispatchInvoke(req, db)
	if err := encodeInvokeResponse(stdout, resp); err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("%s", resp.Error)
	}
	return nil
}

func runInvoke(args []string) {
	if len(args) != 0 {
		fmt.Fprintln(os.Stderr, "usage: xpanel invoke")
		os.Exit(1)
	}
	initViper.Init()
	db, err := openInvokeDatabase(global.CONF.System.DbPath)
	if err != nil {
		_ = encodeInvokeResponse(os.Stdout, invokeResponse{Error: invokeDBErrorCode(err)})
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if sqlDB, dbErr := db.DB(); dbErr == nil {
		defer sqlDB.Close()
	}
	if err := runInvokeWith(args, os.Stdin, os.Stdout, db); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
