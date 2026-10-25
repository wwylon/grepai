package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const watchFreezeFileName = "watch.freeze"

func watchFreezePath(projectRoot string) string {
	return filepath.Join(projectRoot, ".grepai", watchFreezeFileName)
}

func writeWatchFreezeFile(projectRoot string) error {
	dir := filepath.Join(projectRoot, ".grepai")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create .grepai: %w", err)
	}
	return os.WriteFile(watchFreezePath(projectRoot), []byte("frozen\n"), 0600)
}

func clearWatchFreezeFile(projectRoot string) error {
	err := os.Remove(watchFreezePath(projectRoot))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func isWatchFrozen(projectRoot string) bool {
	_, err := os.Stat(watchFreezePath(projectRoot))
	return err == nil
}

// frozenDirty coalesces paths while embeds are frozen.
type frozenDirty struct {
	mu      sync.Mutex
	paths   map[string]struct{}
	capN    int
	overflow bool
}

func newFrozenDirty(capN int) *frozenDirty {
	if capN < 1 {
		capN = 4096
	}
	return &frozenDirty{paths: make(map[string]struct{}), capN: capN}
}

func (f *frozenDirty) add(path string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.paths) >= f.capN {
		f.overflow = true
		return
	}
	f.paths[path] = struct{}{}
}

func (f *frozenDirty) takeAll() (paths []string, overflow bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	paths = make([]string, 0, len(f.paths))
	for p := range f.paths {
		paths = append(paths, p)
	}
	overflow = f.overflow
	f.paths = make(map[string]struct{})
	f.overflow = false
	return paths, overflow
}

func (f *frozenDirty) len() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.paths)
}
