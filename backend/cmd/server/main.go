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
) {
	if len(args) > 0 {
		switch args[0] {
		case "setup":
			setup(args[1:])
			return
		case "bootstrap-config":
			bootstrapConfig(args[1:])
			return
		case "credentials":
			credentialsCommand(args[1:])
			return
		case "migrate":
			migrate()
			return
		case "version", "--version":
			showVersion()
			return
		}
	}
	start()
}

func main() {
	initPermission.SetProcessUmask()
	run(
		os.Args[1:],
		server.Start,
		server.Migrate,
		runSetup,
		runBootstrapConfig,
		runCredentials,
		printVersion,
	)
}

func printVersion() {
	info := version.Get()
	fmt.Printf("xpanel %s (commit %s, built %s, %s)\n", info.Version, info.CommitHash, info.BuildTime, info.GoVersion)
}
