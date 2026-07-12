package main

import (
	"fmt"
	"os"
	"xpanel/app/version"
	"xpanel/server"
)

func run(args []string, start func(), migrate func(), setup func([]string), showVersion func()) {
	if len(args) > 0 {
		switch args[0] {
		case "setup":
			setup(args[1:])
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
	run(os.Args[1:], server.Start, server.Migrate, runSetup, printVersion)
}

func printVersion() {
	info := version.Get()
	fmt.Printf("xpanel %s (commit %s, built %s, %s)\n", info.Version, info.CommitHash, info.BuildTime, info.GoVersion)
}
