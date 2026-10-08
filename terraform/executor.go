package terraform

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// maxLineSize は 1 行として読み取る出力の上限。terraform の plan 出力には長い行が含まれることがある。
const maxLineSize = 1024 * 1024

func RunCommand(prefix string, modulePath string, args ...string) error {
	fmt.Printf("[%s] Running: terraform %v\n", prefix, args)
	return runWithPrefix(os.Stdout, prefix, modulePath, "terraform", args...)
}

// runWithPrefix はコマンドを実行し、stdout と stderr を 1 行ずつ [prefix] 付きで out に逐次書き出す。
func runWithPrefix(out io.Writer, prefix string, dir string, name string, args ...string) error {
	pr, pw := io.Pipe()

	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = pw
	cmd.Stderr = pw

	done := make(chan struct{})
	go func() {
		defer close(done)
		scanner := bufio.NewScanner(pr)
		scanner.Buffer(make([]byte, 0, 64*1024), maxLineSize)
		for scanner.Scan() {
			line := scanner.Text()
			if line != "" {
				_, _ = fmt.Fprintf(out, "[%s] %s\n", prefix, line)
			}
		}
		if err := scanner.Err(); err != nil {
			_, _ = fmt.Fprintf(out, "[%s] (output truncated: %v)\n", prefix, err)
			// 読み取りを止めるとコマンドが書き込みでブロックするので、残りは読み捨てる
			_, _ = io.Copy(io.Discard, pr)
		}
	}()

	err := cmd.Run()
	_ = pw.Close()
	<-done

	return err
}
