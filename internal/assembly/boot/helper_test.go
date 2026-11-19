package boot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"reasonix/internal/base/testenv"
)

// waitFor polls ok until it reports true, failing the test at the package's
// per-test budget. Use it around states only the running assembly can settle.
func waitFor(t *testing.T, what string, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(testenv.Budget(t))
	for !ok() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// writeFileRaw writes body to dir/name, trimming a leading newline so test
// literals can start on the line after the backtick. Parent directories are
// created so callers can write nested paths (e.g. .reasonix/skills/x.md).
func writeFileRaw(dir, name, body string) error {
	full := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	return os.WriteFile(full, []byte(strings.TrimPrefix(body, "\n")), 0o644)
}
