package serve

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher monitors content and theme directories for file changes,
// debouncing rapid edits into a single rebuild trigger.
type Watcher struct {
	watcher  *fsnotify.Watcher
	onChange func()
	debounce time.Duration
	done     chan struct{}
}

// NewWatcher creates a file watcher on the given directories with debounce.
func NewWatcher(dirs []string, debounce time.Duration, onChange func()) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("creating watcher: %w", err)
	}

	for _, dir := range dirs {
		if err := addRecursive(fw, dir); err != nil {
			fw.Close()
			return nil, fmt.Errorf("watching %s: %w", dir, err)
		}
	}

	return &Watcher{
		watcher:  fw,
		onChange: onChange,
		debounce: debounce,
		done:     make(chan struct{}),
	}, nil
}

// Start begins watching for changes in a goroutine.
func (w *Watcher) Start() {
	go w.loop()
}

// Close stops the watcher.
func (w *Watcher) Close() error {
	close(w.done)
	return w.watcher.Close()
}

func (w *Watcher) loop() {
	var timer *time.Timer

	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			// Watch new directories
			if event.Has(fsnotify.Create) {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					addRecursive(w.watcher, event.Name)
				}
			}

			// Debounce: reset timer on each event
			if timer != nil {
				timer.Stop()
			}
			timer = time.AfterFunc(w.debounce, w.onChange)

		case _, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			// Log and continue — don't crash the server on watcher errors

		case <-w.done:
			if timer != nil {
				timer.Stop()
			}
			return
		}
	}
}

// addRecursive adds a directory and all subdirectories to the watcher.
func addRecursive(fw *fsnotify.Watcher, dir string) error {
	return filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip inaccessible paths
		}
		if d.IsDir() {
			return fw.Add(path)
		}
		return nil
	})
}
