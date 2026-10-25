package watcher

import (
	"path/filepath"
	"testing"
	"time"
)

func TestAdaptiveDebounce_SingleEventUsesFloor(t *testing.T) {
	dir := t.TempDir()
	w, err := NewWatcher(dir, nil, 200)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	w.SetAdaptiveDebounce(true, 2000)

	w.debounceEvent(FileEvent{Type: EventModify, Path: "a.go"})
	if got := w.EffectiveDebounceMs(); got != 200 {
		t.Fatalf("effective=%d want floor 200", got)
	}
}

func TestAdaptiveDebounce_GrowsWithBurst(t *testing.T) {
	dir := t.TempDir()
	w, err := NewWatcher(dir, nil, 100)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	w.SetAdaptiveDebounce(true, 1000)

	// Stay inside the floor window so flush does not clear the burst.
	w.debounceEvent(FileEvent{Type: EventModify, Path: "a.go"})
	time.Sleep(50 * time.Millisecond)
	w.debounceEvent(FileEvent{Type: EventModify, Path: "b.go"})
	got := w.EffectiveDebounceMs()
	if got <= 100 {
		t.Fatalf("expected effective > floor after burst, got %d (adaptive=%v)", got, w.AdaptiveDebounce())
	}
	if got > 1000 {
		t.Fatalf("expected effective capped at max, got %d", got)
	}
	_ = filepath.Base(dir)
}

func TestAdaptiveDebounce_DisabledStaysFixed(t *testing.T) {
	dir := t.TempDir()
	w, err := NewWatcher(dir, nil, 300)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	w.debounceEvent(FileEvent{Type: EventModify, Path: "a.go"})
	time.Sleep(50 * time.Millisecond)
	w.debounceEvent(FileEvent{Type: EventModify, Path: "b.go"})
	if got := w.EffectiveDebounceMs(); got != 300 {
		t.Fatalf("fixed mode effective=%d want 300", got)
	}
}
