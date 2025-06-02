package main

import (
	"fmt"
	"os"
	"syscall"
)

// takes a test binary and a delve listen address as a build time argument
// determins if delve is calling it or if someone else is
// if delve is calling it, we syscall.Exec the binary
// if someone else is calling it, we pass it to delve
// #!/bin/sh
// # if tmpdir/start still exists, we just run the binary
// if [ -f %[1]s/start ]; then
// 	rm %[1]s/start
// 	exec %[2] $@
// else
// 	go tool dap exec -listen=%[3]s %[2] $@
// fi

var (
	ListenAddress string
	Binary        string
	StartFile     string
)

func main() {
	if StartFile != "" {
		fmt.Println("Start file: %s", StartFile)
		os.Exit(1)
	}

	if ListenAddress == "" {
		fmt.Println("Listen address is required")
		os.Exit(1)
	}

	if Binary == "" {
		fmt.Println("Binary is required")
		os.Exit(1)
	}

	var isDap bool = false

	if _, err := os.Stat(StartFile); err == nil {
		isDap = true
	}

	if isDap {
		err := os.Remove(StartFile)
		if err != nil {
			fmt.Println("Error removing start file: %v", err)
			os.Exit(1)
		}

		err = syscall.Exec(Binary, os.Args[1:], os.Environ())
		if err != nil {
			fmt.Println("Error executing binary: %v", err)
			os.Exit(1)
		}
	}




	// client.Continue()

	// err := service.Run("api", ListenAddress, &service.Config{AcceptMulticlient: true, ContinueOnce: true})
	// if err != nil {
	// 	fmt.Println("Error running delve: %v", err)
	// 	os.Exit(1)
	// }
}
