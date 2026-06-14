package serve

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gofuego/fuego/internal/config"
)

// Run starts the development server: an initial build, a dev subprocess (if
// configured), a file watcher that rebuilds on change, and an HTTP server. The
// build closure performs one site build; Run owns the watch/serve loop and the
// build lock. It blocks until the process is signaled (SIGINT/SIGTERM).
func Run(cfg *config.Config, build func() error) error {
	fmt.Println("fuego: building site...")
	if err := build(); err != nil {
		fmt.Fprintf(os.Stderr, "fuego: initial build error: %v\n", err)
	}

	sub, err := StartSubprocess(cfg.Dev.Command)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fuego: %v\n", err)
	}

	var buildMu sync.Mutex
	watchDirs := []string{cfg.Dirs.Content, cfg.Dirs.Theme}
	watcher, err := NewWatcher(watchDirs, 100*time.Millisecond, func() {
		buildMu.Lock()
		defer buildMu.Unlock()
		fmt.Println("fuego: change detected, rebuilding...")
		if err := build(); err != nil {
			fmt.Fprintf(os.Stderr, "fuego: rebuild error: %v\n", err)
		} else {
			fmt.Println("fuego: rebuild complete")
		}
	})
	if err != nil {
		return fmt.Errorf("setting up watcher: %w", err)
	}
	watcher.Start()

	handler := NewHandler(cfg.Dirs.Output, cfg.Dev.ProxyPort)
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Dev.Port),
		Handler: handler,
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nfuego: shutting down...")
		watcher.Close()
		if sub != nil {
			sub.Stop()
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	fmt.Printf("fuego: serving at http://localhost:%d\n", cfg.Dev.Port)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}
