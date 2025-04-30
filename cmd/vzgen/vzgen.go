package vzgen

import (
	"fmt"
	"os"
	"os/exec"
)

//go:generate go run main.go

func main() {
	fmt.Println("Running nswrap to generate Virtualization.framework bindings...")

	// Look for nswrap in PATH or specify full path
	cmd := exec.Command("nswrap", "nswrap.yaml")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running nswrap: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Successfully generated Virtualization.framework bindings")
}
