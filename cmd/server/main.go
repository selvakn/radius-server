package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/crypto/bcrypt"
	"layeh.com/radius"

	"github.com/selvakn/radius-server/internal/auth"
	"github.com/selvakn/radius-server/internal/config"
	"github.com/selvakn/radius-server/internal/db"
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
	defer database.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	radiusAddr := fmt.Sprintf(":%d", cfg.Radius.Port)
	radiusSrv := &radius.PacketServer{
		Addr:         radiusAddr,
		Handler:      auth.New(database, cfg.Radius.SharedSecret),
		SecretSource: radius.StaticSecretSource([]byte(cfg.Radius.SharedSecret)),
	}

	slog.Info("starting RADIUS server", "addr", radiusAddr)
	go func() {
		if err := radiusSrv.ListenAndServe(); err != nil {
			slog.Error("radius server error", "err", err)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")
	radiusSrv.Shutdown(context.Background())
}

func runHashPassword() {
	fmt.Fprintf(os.Stderr, "Enter password: ")
	var plain string
	fmt.Scanln(&plain)
	if plain == "" {
		fmt.Fprintln(os.Stderr, "error: empty password")
		os.Exit(1)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), 12)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(hash))
}
