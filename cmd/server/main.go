package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/crypto/bcrypt"
	"layeh.com/radius"

	"github.com/selvakn/radius-server/internal/auth"
	"github.com/selvakn/radius-server/internal/config"
	"github.com/selvakn/radius-server/internal/db"
	"github.com/selvakn/radius-server/internal/web"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "hash-password" {
		runHashPassword()
		return
	}

	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("load config", "err", err)
		os.Exit(1)
	}

	database, err := db.Open(cfg.Database.Path)
	if err != nil {
		slog.Error("open database", "err", err)
		os.Exit(1)
	}
	defer func() { _ = database.Close() }()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	radiusAddr := fmt.Sprintf(":%d", cfg.Radius.Port)
	radiusSrv := &radius.PacketServer{
		Addr:         radiusAddr,
		Handler:      auth.New(database, cfg.Radius.SharedSecret),
		SecretSource: radius.StaticSecretSource([]byte(cfg.Radius.SharedSecret)),
	}

	sessions := web.NewSessionStore()
	webSrv := web.New(database, cfg, sessions)
	webAddr := fmt.Sprintf(":%d", cfg.Web.Port)

	slog.Info("starting RADIUS server", "addr", radiusAddr)
	go func() {
		if err := radiusSrv.ListenAndServe(); err != nil {
			slog.Error("radius server error", "err", err)
		}
	}()

	slog.Info("starting admin web server", "addr", webAddr)
	httpSrv := &http.Server{Addr: webAddr, Handler: webSrv.Router()}
	go func() {
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("web server error", "err", err)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")
	_ = radiusSrv.Shutdown(context.Background())
	_ = httpSrv.Shutdown(context.Background())
}

func runHashPassword() {
	fmt.Fprintf(os.Stderr, "Enter password: ")
	if err := hashPassword(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func hashPassword(r io.Reader, w io.Writer) error {
	var plain string
	if _, err := fmt.Fscanln(r, &plain); err != nil {
		return fmt.Errorf("reading password: %w", err)
	}
	if plain == "" {
		return fmt.Errorf("empty password")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), 12)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(hash))
	return err
}
