package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"pack-shipping-calculator/backend/internal/config"
	"pack-shipping-calculator/backend/internal/httpapi"
	"pack-shipping-calculator/backend/internal/packs"
	"pack-shipping-calculator/backend/internal/storage/sqlite"
)

const defaultConfigPath = "config/config.json"
const defaultShutdownTimeout = 10 * time.Second
const portEnv = "PORT"

func Run(ctx context.Context) error {
	return run(ctx, configPath(), loadConfig, openRepository, makeServer, defaultShutdownTimeout)
}

type server interface {
	ListenAndServe() error
	Shutdown(context.Context) error
}

var (
	loadConfig     = config.Load
	openRepository = func(ctx context.Context, path string) (packs.Repository, error) {
		return sqlite.Open(ctx, path)
	}
	makeServer = newHTTPServer
)

func run(
	ctx context.Context,
	cfgPath string,
	load func(string) (config.Config, error),
	openRepo func(context.Context, string) (packs.Repository, error),
	newServer func(config.Config, *packs.Service) server,
	shutdownTimeout time.Duration,
) error {
	cfg, err := load(cfgPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	repository, err := openRepo(ctx, cfg.DatabasePath)
	if err != nil {
		return fmt.Errorf("open repository: %w", err)
	}

	service := packs.NewService(repository)
	if err := service.SeedPackSizesIfEmpty(ctx, cfg.PackSizes); err != nil {
		_ = service.Close()
		return fmt.Errorf("seed pack sizes: %w", err)
	}

	serveErr := serve(ctx, newServer(cfg, service), shutdownTimeout)
	closeErr := service.Close()
	if serveErr != nil {
		return serveErr
	}
	if closeErr != nil {
		return fmt.Errorf("close repository: %w", closeErr)
	}
	return nil
}

func serve(ctx context.Context, server server, shutdownTimeout time.Duration) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
}

func configPath() string {
	cfgPath := os.Getenv("PACK_CALCULATOR_CONFIG")
	if cfgPath != "" {
		return cfgPath
	}
	return defaultConfigPath
}

func newHTTPServer(cfg config.Config, service *packs.Service) server {
	return &http.Server{
		Addr:              ":" + httpPort(cfg),
		Handler:           httpapi.NewRouter(service),
		ReadHeaderTimeout: 5 * time.Second,
	}
}

func httpPort(cfg config.Config) string {
	port := os.Getenv(portEnv)
	if port != "" {
		return port
	}
	return cfg.HTTPPort
}
