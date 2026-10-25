package cli

import (
	"testing"
)

func TestWatchOnceRequiresWorkspace(t *testing.T) {
	origOnce, origWS, origBg, origStatus, origStop := watchOnce, watchWorkspace, watchBackground, watchStatus, watchStop
	t.Cleanup(func() {
		watchOnce, watchWorkspace, watchBackground, watchStatus, watchStop = origOnce, origWS, origBg, origStatus, origStop
	})

	watchOnce = true
	watchWorkspace = ""
	watchBackground, watchStatus, watchStop = false, false, false
	err := runWatch(nil, nil)
	if err == nil || err.Error() != "--once requires --workspace" {
		t.Fatalf("got %v, want --once requires --workspace", err)
	}
}

func TestWatchOnceExclusiveWithBackground(t *testing.T) {
	origOnce, origWS, origBg, origStatus, origStop := watchOnce, watchWorkspace, watchBackground, watchStatus, watchStop
	t.Cleanup(func() {
		watchOnce, watchWorkspace, watchBackground, watchStatus, watchStop = origOnce, origWS, origBg, origStatus, origStop
	})

	watchOnce = true
	watchWorkspace = "demo"
	watchBackground = true
	watchStatus, watchStop = false, false
	err := runWatch(nil, nil)
	if err == nil {
		t.Fatal("expected mutual exclusion error")
	}
}
