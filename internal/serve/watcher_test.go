package serve

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestWatcher_DetectsChange(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping watcher test in short mode")
	}

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.md"), []byte("initial"), 0644)

	var called atomic.Int32

	w, err := NewWatcher([]string{dir}, 50*time.Millisecond, func() {
		called.Add(1)
	})
	if err != nil {
		t.Fatalf("creating watcher: %v", err)
	}
	defer w.Close()
	w.Start()

	// Modify file
	time.Sleep(100 * time.Millisecond)
	os.WriteFile(filepath.Join(dir, "test.md"), []byte("changed"), 0644)

	// Wait for debounce + processing
	time.Sleep(300 * time.Millisecond)

	if called.Load() == 0 {
		t.Error("expected onChange to be called at least once")
	}
}

func TestWatcher_DebouncesBurst(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping watcher test in short mode")
	}

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.md"), []byte("initial"), 0644)

	var called atomic.Int32

	w, err := NewWatcher([]string{dir}, 200*time.Millisecond, func() {
		called.Add(1)
	})
	if err != nil {
		t.Fatalf("creating watcher: %v", err)
	}
	defer w.Close()
	w.Start()

	// Burst of rapid writes
	time.Sleep(100 * time.Millisecond)
	for i := 0; i < 5; i++ {
		os.WriteFile(filepath.Join(dir, "test.md"), []byte("burst"), 0644)
		time.Sleep(20 * time.Millisecond)
	}

	// Wait for debounce
	time.Sleep(500 * time.Millisecond)

	count := called.Load()
	if count != 1 {
		t.Errorf("expected 1 debounced call, got %d", count)
	}
}
