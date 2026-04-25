package app

import (
	"context"
	"errors"
	"net/http"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"pack-shipping-calculator/backend/internal/config"
	"pack-shipping-calculator/backend/internal/packs"
)

func TestRunUsesConfiguredDependencies(t *testing.T) {
	originalLoadConfig := loadConfig
	originalOpenRepository := openRepository
	originalMakeServer := makeServer
	defer func() {
		loadConfig = originalLoadConfig
		openRepository = originalOpenRepository
		makeServer = originalMakeServer
	}()

	t.Setenv("PACK_CALCULATOR_CONFIG", "custom.json")

	repository := &fakeRepository{}
	wantConfig := config.Config{HTTPPort: "9090", DatabasePath: "custom.db", PackSizes: []int{10, 20}}
	loadConfig = func(path string) (config.Config, error) {
		if path != "custom.json" {
			t.Fatalf("got config path %q, want custom.json", path)
		}
		return wantConfig, nil
	}
	openRepository = func(_ context.Context, path string) (packs.Repository, error) {
		if path != "custom.db" {
			t.Fatalf("got database path %q, want custom.db", path)
		}
		return repository, nil
	}
	makeServer = func(cfg config.Config, service *packs.Service) server {
		if !reflect.DeepEqual(cfg, wantConfig) {
			t.Fatalf("got config %#v, want %#v", cfg, wantConfig)
		}
		if got, err := service.GetPackSizes(context.Background()); err != nil || !reflect.DeepEqual(got, []int{10, 20}) {
			t.Fatalf("got seeded sizes %#v, err %v", got, err)
		}
		return &fakeServer{listenErr: http.ErrServerClosed}
	}

	if err := Run(context.Background()); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !repository.closed {
		t.Fatal("expected repository to be closed")
	}
}

func TestDefaultOpenRepository(t *testing.T) {
	repository, err := openRepository(context.Background(), filepath.Join(t.TempDir(), "packs.db"))
	if err != nil {
		t.Fatalf("openRepository returned error: %v", err)
	}
	if err := repository.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
}

func TestRunReturnsConfigError(t *testing.T) {
	sentinel := errors.New("config failed")
	repositoryOpened := false

	err := run(
		context.Background(),
		"config.json",
		func(string) (config.Config, error) {
			return config.Config{}, sentinel
		},
		func(context.Context, string) (packs.Repository, error) {
			repositoryOpened = true
			return &fakeRepository{}, nil
		},
		func(config.Config, *packs.Service) server {
			return &fakeServer{}
		},
		time.Second,
	)

	if !errors.Is(err, sentinel) {
		t.Fatalf("got error %v, want %v", err, sentinel)
	}
	if !strings.Contains(err.Error(), "load config") {
		t.Fatalf("got error %q, want load config context", err.Error())
	}
	if repositoryOpened {
		t.Fatal("repository should not be opened")
	}
}

func TestRunReturnsOpenRepositoryError(t *testing.T) {
	sentinel := errors.New("open failed")

	err := run(
		context.Background(),
		"config.json",
		func(string) (config.Config, error) {
			return config.Config{HTTPPort: "8080", DatabasePath: "data.db", PackSizes: []int{250}}, nil
		},
		func(context.Context, string) (packs.Repository, error) {
			return nil, sentinel
		},
		func(config.Config, *packs.Service) server {
			return &fakeServer{}
		},
		time.Second,
	)

	if !errors.Is(err, sentinel) {
		t.Fatalf("got error %v, want %v", err, sentinel)
	}
	if !strings.Contains(err.Error(), "open repository") {
		t.Fatalf("got error %q, want open repository context", err.Error())
	}
}

func TestRunReturnsSeedErrorAndClosesRepository(t *testing.T) {
	sentinel := errors.New("seed failed")
	repository := &fakeRepository{seedErr: sentinel}

	err := run(
		context.Background(),
		"config.json",
		func(string) (config.Config, error) {
			return config.Config{HTTPPort: "8080", DatabasePath: "data.db", PackSizes: []int{250}}, nil
		},
		func(context.Context, string) (packs.Repository, error) {
			return repository, nil
		},
		func(config.Config, *packs.Service) server {
			return &fakeServer{}
		},
		time.Second,
	)

	if !errors.Is(err, packs.ErrRepository) {
		t.Fatalf("got error %v, want repository error", err)
	}
	if !strings.Contains(err.Error(), "seed pack sizes") {
		t.Fatalf("got error %q, want seed context", err.Error())
	}
	if !repository.closed {
		t.Fatal("expected repository to be closed")
	}
}

func TestRunReturnsServeErrorAndClosesRepository(t *testing.T) {
	sentinel := errors.New("listen failed")
	repository := &fakeRepository{}

	err := run(
		context.Background(),
		"config.json",
		func(string) (config.Config, error) {
			return config.Config{HTTPPort: "8080", DatabasePath: "data.db", PackSizes: []int{250}}, nil
		},
		func(context.Context, string) (packs.Repository, error) {
			return repository, nil
		},
		func(config.Config, *packs.Service) server {
			return &fakeServer{listenErr: sentinel}
		},
		time.Second,
	)

	if !errors.Is(err, sentinel) {
		t.Fatalf("got error %v, want %v", err, sentinel)
	}
	if !repository.closed {
		t.Fatal("expected repository to be closed")
	}
}

func TestRunReturnsCloseError(t *testing.T) {
	sentinel := errors.New("close failed")

	err := run(
		context.Background(),
		"config.json",
		func(string) (config.Config, error) {
			return config.Config{HTTPPort: "8080", DatabasePath: "data.db", PackSizes: []int{250}}, nil
		},
		func(context.Context, string) (packs.Repository, error) {
			return &fakeRepository{closeErr: sentinel}, nil
		},
		func(config.Config, *packs.Service) server {
			return &fakeServer{listenErr: http.ErrServerClosed}
		},
		time.Second,
	)

	if !errors.Is(err, sentinel) {
		t.Fatalf("got error %v, want %v", err, sentinel)
	}
	if !strings.Contains(err.Error(), "close repository") {
		t.Fatalf("got error %q, want close repository context", err.Error())
	}
}

func TestServeReturnsListenError(t *testing.T) {
	sentinel := errors.New("listen failed")

	err := serve(context.Background(), &fakeServer{listenErr: sentinel}, time.Second)
	if !errors.Is(err, sentinel) {
		t.Fatalf("got error %v, want %v", err, sentinel)
	}
}

func TestServeTreatsServerClosedAsSuccess(t *testing.T) {
	err := serve(context.Background(), &fakeServer{listenErr: http.ErrServerClosed}, time.Second)
	if err != nil {
		t.Fatalf("serve returned error: %v", err)
	}
}

func TestServeShutsDownOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	server := &fakeServer{block: make(chan struct{})}
	if err := serve(ctx, server, time.Second); err != nil {
		t.Fatalf("serve returned error: %v", err)
	}
	if !server.wasShutdown() {
		t.Fatal("expected Shutdown to be called")
	}
}

func TestConfigPath(t *testing.T) {
	t.Setenv("PACK_CALCULATOR_CONFIG", "")
	if got := configPath(); got != defaultConfigPath {
		t.Fatalf("got config path %q, want %q", got, defaultConfigPath)
	}
}

func TestNewHTTPServer(t *testing.T) {
	t.Setenv("PORT", "")

	cfg := config.Config{HTTPPort: "9090", DatabasePath: "data.db", PackSizes: []int{250}}

	server, ok := newHTTPServer(cfg, packs.NewService(&fakeRepository{})).(*http.Server)
	if !ok {
		t.Fatalf("got %T, want *http.Server", server)
	}
	if server.Addr != ":9090" {
		t.Fatalf("got addr %q, want :9090", server.Addr)
	}
	if server.Handler == nil {
		t.Fatal("expected handler")
	}
	if server.ReadHeaderTimeout != 5*time.Second {
		t.Fatalf("got timeout %s, want 5s", server.ReadHeaderTimeout)
	}
}

func TestNewHTTPServerUsesPortEnvironment(t *testing.T) {
	t.Setenv("PORT", "10000")

	cfg := config.Config{HTTPPort: "9090", DatabasePath: "data.db", PackSizes: []int{250}}

	server, ok := newHTTPServer(cfg, packs.NewService(&fakeRepository{})).(*http.Server)
	if !ok {
		t.Fatalf("got %T, want *http.Server", server)
	}
	if server.Addr != ":10000" {
		t.Fatalf("got addr %q, want :10000", server.Addr)
	}
}

type fakeRepository struct {
	sizes    []int
	seedErr  error
	closeErr error
	closed   bool
}

func (r *fakeRepository) GetPackSizes(context.Context) ([]int, error) {
	return r.sizes, nil
}

func (r *fakeRepository) ReplacePackSizes(_ context.Context, sizes []int) ([]int, error) {
	r.sizes = sizes
	return sizes, nil
}

func (r *fakeRepository) SeedPackSizesIfEmpty(_ context.Context, sizes []int) error {
	if r.seedErr != nil {
		return r.seedErr
	}
	if len(r.sizes) == 0 {
		r.sizes = sizes
	}
	return nil
}

func (r *fakeRepository) Close() error {
	r.closed = true
	return r.closeErr
}

type fakeServer struct {
	listenErr error
	block     chan struct{}

	shutdownOnce sync.Once
	shutdownMu   sync.Mutex
	shutdown     bool
}

func (s *fakeServer) ListenAndServe() error {
	if s.block != nil {
		<-s.block
	}
	return s.listenErr
}

func (s *fakeServer) Shutdown(context.Context) error {
	s.shutdownMu.Lock()
	s.shutdown = true
	s.shutdownMu.Unlock()

	if s.block != nil {
		s.shutdownOnce.Do(func() {
			close(s.block)
		})
	}
	return nil
}

func (s *fakeServer) wasShutdown() bool {
	s.shutdownMu.Lock()
	defer s.shutdownMu.Unlock()
	return s.shutdown
}
