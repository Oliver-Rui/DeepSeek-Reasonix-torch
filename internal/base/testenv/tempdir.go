package testenv

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"reasonix/internal/base/workspaceid"
)

// TB is the subset of *testing.T that TempDir needs, declared here so this
// package stays free of a testing import the way TestingM already does.
type TB interface {
	Helper()
	Cleanup(func())
	Fatalf(format string, args ...any)
	Logf(format string, args ...any)
}

const (
	tempDirRemoveAttempts = 100
	tempDirRemoveBackoff  = 20 * time.Millisecond
)

// tempBase resolves where isolated test directories are made, once. The system
// temp folder is it — unless that sits inside a repository: a home that doubles
// as a dotfiles repo keys every temp folder as the one same project, and
// per-project behavior under test collapses with it. Then a writable folder at
// the volume root stands in, one the bare root itself is not on Windows.
var (
	tempBaseOnce sync.Once
	tempBaseDir  string
)

func tempBase() string {
	tempBaseOnce.Do(func() {
		base := os.TempDir()
		if strings.HasPrefix(workspaceid.Key(base), workspaceid.RepoPrefix) {
			root := filepath.VolumeName(filepath.Clean(base))
			if root == "" {
				root = "/"
			}
			// Keep the separator: Join would turn "C:" + name into a
			// drive-relative spelling, which is not an absolute path.
			standIn := root + string(filepath.Separator) + "reasonix-tests"
			if err := os.MkdirAll(standIn, 0o755); err == nil {
				base = standIn
			}
		}
		tempBaseDir = base
	})
	return tempBaseDir
}

// TempDir is a drop-in for t.TempDir whose cleanup retries RemoveAll. A test
// can finish while the OS still holds a file it wrote: Windows keeps a deleted
// file's directory entry until its last handle closes, so removing the
// directory fails as non-empty when every file is already gone. Measured at ~2%
// of session saves, always inside 10ms; one that never frees is logged, not fatal.
func TempDir(t TB) string {
	t.Helper()
	dir, err := os.MkdirTemp(tempBase(), "reasonix-test-*")
	if err != nil {
		t.Fatalf("testenv.TempDir: %v", err)
		return ""
	}
	t.Cleanup(func() {
		var rmErr error
		for range tempDirRemoveAttempts {
			if rmErr = os.RemoveAll(dir); rmErr == nil {
				return
			}
			time.Sleep(tempDirRemoveBackoff)
		}
		t.Logf("testenv.TempDir: cleanup did not converge for %s: %v", dir, rmErr)
	})
	return dir
}
