package main

import (
	"fmt"

	"xpanel/app/service"
	"xpanel/server"
)

func runUpdate(args []string) error {
	return executeUpdate(args, server.Migrate, func() error {
		return (&service.UpgradeService{}).UpgradeLatest()
	})
}

func executeUpdate(args []string, migrate func(), updateLatest func() error) error {
	if len(args) != 1 || args[0] != "--latest" {
		return fmt.Errorf("usage: xpanel update --latest")
	}
	migrate()
	return updateLatest()
}
