package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/FabioSol/fuego/core"
	"github.com/FabioSol/fuego/internal/config"
	"github.com/FabioSol/fuego/internal/pipeline"
	"github.com/FabioSol/fuego/internal/serve"
	"github.com/spf13/cobra"
)

func newServeCmd(parsers map[string]core.Parser, hooks *core.Hooks, packs []core.Pack, configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the development server with live reload",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(*configPath, packs)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			return runServe(cfg, parsers, hooks, packs)
		},
	}
}

func runServe(cfg *config.Config, parsers map[string]core.Parser, hooks *core.Hooks, packs []core.Pack) error {
	// Initial build
	fmt.Println("fuego: building site...")
	if err := doBuild(cfg, parsers, hooks, packs); err != nil {
		// In serve mode, log the error but don't exit
		fmt.Fprintf(os.Stderr, "fuego: initial build error: %v\n", err)
	}

	// Start dev subprocess (e.g., Vite)
	sub, err := serve.StartSubprocess(cfg.Dev.Command)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fuego: %v\n", err)
	}

	// Build lock to prevent concurrent rebuilds
	var buildMu sync.Mutex

	// Set up file watcher on content and theme dirs
	watchDirs := []string{cfg.Dirs.Content, cfg.Dirs.Theme}
	watcher, err := serve.NewWatcher(watchDirs, 100*time.Millisecond, func() {
		buildMu.Lock()
		defer buildMu.Unlock()

		fmt.Println("fuego: change detected, rebuilding...")
		if err := doBuild(cfg, parsers, hooks, packs); err != nil {
			fmt.Fprintf(os.Stderr, "fuego: rebuild error: %v\n", err)
		} else {
			fmt.Println("fuego: rebuild complete")
		}
	})
	if err != nil {
		return fmt.Errorf("setting up watcher: %w", err)
	}
	watcher.Start()

	// HTTP server
	handler := serve.NewHandler(cfg.Dirs.Output, cfg.Dev.ProxyPort)
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Dev.Port),
		Handler: handler,
	}

	// Graceful shutdown on SIGINT/SIGTERM
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

func doBuild(cfg *config.Config, parsers map[string]core.Parser, hooks *core.Hooks, packs []core.Pack) error {
	ctx := context.Background()
	// The dev server rebuilds on every change, so incremental parsing keeps
	// rebuilds fast on large sites.
	return pipeline.Build(ctx, cfg, parsers, hooks, packs, pipeline.Options{Incremental: true})
}
