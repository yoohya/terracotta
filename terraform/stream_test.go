package terraform

import (
	"bytes"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"
)

// syncBuffer は並行に書き込まれても安全に読めるバッファ。
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func requireSh(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not found in PATH")
	}
}

func TestRunWithPrefixPrefixesStdoutAndStderr(t *testing.T) {
	requireSh(t)
	var out syncBuffer

	err := runWithPrefix(&out, "mod", t.TempDir(), "sh", "-c", `echo one; echo; echo two 1>&2`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "[mod] one\n[mod] two\n"
	if out.String() != want {
		t.Errorf("expected %q, got %q", want, out.String())
	}
}

func TestRunWithPrefixStreamsBeforeExit(t *testing.T) {
	requireSh(t)
	var out syncBuffer

	errCh := make(chan error, 1)
	go func() {
		errCh <- runWithPrefix(&out, "mod", t.TempDir(), "sh", "-c", `echo started; sleep 2; echo finished`)
	}()

	// コマンドの終了を待たずに最初の行が表示されること
	deadline := time.Now().Add(1500 * time.Millisecond)
	for !strings.Contains(out.String(), "[mod] started") {
		if time.Now().After(deadline) {
			t.Fatalf("first line was not streamed before command exit, got %q", out.String())
		}
		time.Sleep(20 * time.Millisecond)
	}

	if err := <-errCh; err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "[mod] finished") {
		t.Errorf("expected final line, got %q", out.String())
	}
}

func TestRunWithPrefixReturnsExitError(t *testing.T) {
	requireSh(t)
	var out syncBuffer

	err := runWithPrefix(&out, "mod", t.TempDir(), "sh", "-c", `echo failing; exit 3`)
	if err == nil {
		t.Fatal("expected error for non-zero exit, got none")
	}
	if !strings.Contains(out.String(), "[mod] failing") {
		t.Errorf("expected output before failure, got %q", out.String())
	}
}

func TestRunWithPrefixHandlesLongLines(t *testing.T) {
	requireSh(t)
	var out syncBuffer

	// 上限を超える行があっても、コマンドが詰まらずに終了すること
	err := runWithPrefix(&out, "mod", t.TempDir(), "sh", "-c", `head -c 2000000 /dev/zero | tr '\0' 'x'; echo; echo after`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "output truncated") {
		t.Errorf("expected truncation notice, got %d bytes", len(out.String()))
	}
}
