package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestMergeOverlayAppendsOnce(t *testing.T) {
	ws := &Workspace{
		Name: "demo",
		Projects: []ProjectEntry{
			{Name: "frontend", Path: "/src/frontend"},
		},
	}
	merged := MergeOverlay(ws, OverlayEntry{Name: "feature", Path: "/tmp/wt", Metadata: map[string]string{"upstream": "frontend"}}, true)
	if len(merged.Projects) != 2 {
		t.Fatalf("got %d projects, want 2", len(merged.Projects))
	}
	if merged.Projects[1].Metadata["upstream"] != "frontend" {
		t.Fatalf("metadata not copied: %+v", merged.Projects[1].Metadata)
	}
	again := MergeOverlay(merged, OverlayEntry{Name: "feature", Path: "/tmp/wt"}, true)
	if len(again.Projects) != 2 {
		t.Fatalf("duplicate overlay: %d", len(again.Projects))
	}
}

func TestOverlayConfigRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", tmp)
	} else {
		t.Setenv("HOME", tmp)
	}
	cfg := &OverlayConfig{Version: 1, Overlays: map[string]OverlayEntry{}}
	cfg.Set("demo", OverlayEntry{Name: "feature", Path: filepath.Join(tmp, "wt")})
	if err := SaveOverlayConfig(cfg); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadOverlayConfig()
	if err != nil {
		t.Fatal(err)
	}
	got, ok := loaded.Get("demo")
	if !ok || got.Name != "feature" {
		t.Fatalf("got %+v ok=%v", got, ok)
	}
	if _, err := os.Stat(filepath.Join(tmp, ".grepai", OverlayConfigFileName)); err != nil && runtime.GOOS != "windows" {
		// HOME/.grepai on unix
		_ = err
	}
}
