package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"ridearchive/internal/httpapi"
	"ridearchive/internal/store"
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
func run() error {
	addr := env("ARCHIVE_ADDR", "127.0.0.1:8080")
	token := os.Getenv("ARCHIVE_API_TOKEN")
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("ARCHIVE_ADDR: %w", err)
	}
	ip := net.ParseIP(host)
	loopback := host == "localhost" || (ip != nil && ip.IsLoopback())
	if !loopback && len(token) < 32 {
		return errors.New("non-loopback binding requires ARCHIVE_API_TOKEN of at least 32 characters")
	}
	if token != "" && len(token) < 32 {
		return errors.New("ARCHIVE_API_TOKEN must have at least 32 characters")
	}
	path := env("ARCHIVE_DB", "local_data/core.sqlite3")
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	s, err := store.Open(context.Background(), path)
	if err != nil {
		return err
	}
	defer s.Close()
	owner := env("ARCHIVE_OWNER_ID", "local-owner")
	if err = s.EnsureUser(context.Background(), owner, env("ARCHIVE_OWNER_NAME", "My archive")); err != nil {
		return err
	}
	// Only compiled UI assets are served, never the workspace or source files.
	root := env("ARCHIVE_WEB_DIR", "web/dist")
	files := http.FileServer(http.Dir(root))
	static := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" && r.Method != "HEAD" {
			http.Error(w, "Method not allowed", 405)
			return
		}
		if r.URL.Path != "/" && r.URL.Path != "/add-trip" && !strings.HasPrefix(r.URL.Path, "/assets/") {
			http.NotFound(w, r)
			return
		}
		if r.URL.Path == "/add-trip" {
			r.URL.Path = "/"
		}
		files.ServeHTTP(w, r)
	})
	server := &http.Server{Addr: addr, Handler: httpapi.New(s, owner, token, static), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 * 1024}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	done := make(chan error, 1)
	go func() { slog.Info("Ride Archive listening", "address", addr); done <- server.ListenAndServe() }()
	select {
	case err = <-done:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return server.Shutdown(shutdown)
}
func main() {
	if err := run(); err != nil {
		slog.Error("startup failed", "error", err)
		os.Exit(1)
	}
}
