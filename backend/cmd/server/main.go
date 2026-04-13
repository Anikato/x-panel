package main

import (
	"os"
	"xpanel/server"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "setup":
			runSetup(os.Args[2:])
			return
		}
	}
	server.Start()
}
