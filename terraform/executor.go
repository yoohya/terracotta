package terraform

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
)

func RunCommand(modulePath string, args ...string) error {
	cmd := exec.Command("terraform", args...)
	cmd.Dir = modulePath

	fmt.Printf("[%s] Running: terraform %v\n", modulePath, args)

	output, err := cmd.CombinedOutput()
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			fmt.Printf("[%s] %s\n", modulePath, line)
		}
	}

	return err
}
