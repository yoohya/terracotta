package terraform

import (
	"fmt"
	"os"
	"os/exec"
)

func RunCommand(modulePath string, args ...string) error {
	cmd := exec.Command("terraform", args...)
	cmd.Dir = modulePath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("Running: terraform %v (in %s)\n", args, modulePath)
	return cmd.Run()
}
