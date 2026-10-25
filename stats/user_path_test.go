package stats

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestUserStatsPath(t *testing.T) {
	tmp := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", tmp)
	} else {
		t.Setenv("HOME", tmp)
	}
	p, err := UserStatsPath()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(tmp, ".grepai", StatsFileName)
	if p != want {
		t.Fatalf("got %s want %s", p, want)
	}
}
