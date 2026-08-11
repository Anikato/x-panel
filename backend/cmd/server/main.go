package main

import (
	"fmt"
	"os"
	"xpanel/app/version"
	initPermission "xpanel/init/permission"
	"xpanel/server"
)

func run(
	args []string,
	start func(),
	migrate func(),
	setup func([]string),
	bootstrapConfig func([]string),
	credentialsCommand func([]string),
	showVersion func(),
	updateCommand func([]string) error,
) error {
	if len(args) > 0 {
		switch args[0] {
		case "setup":
			setup(args[1:])
			return nil
		case "bootstrap-config":
			bootstrapConfig(args[1:])
			return nil
		case "credentials":
			credentialsCommand(args[1:])
			return nil
		case "migrate":
			migrate()
			return nil
		case "version", "--version":
			showVersion()
			return nil
		case "update":
			return updateCommand(args[1:])
		}
	}
	start()
	return nil
}

func main() {
	initPermission.SetProcessUmask()
	if err := run(
		os.Args[1:],
		server.Start,
		server.Migrate,
		runSetup,
		runBootstrapConfig,
		runCredentials,
		printVersion,
		runUpdate,
	); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func printVersion() {
	info := version.Get()
	fmt.Printf("xpanel %s (commit %s, built %s, %s)\n", info.Version, info.CommitHash, info.BuildTime, info.GoVersion)
}
