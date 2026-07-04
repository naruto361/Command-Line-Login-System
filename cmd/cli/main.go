// Package main wires configuration, persistence, and the interactive CLI.
package main

import (
	"fmt"
	"os"

	"github.com/naruto361/command-line-login-system/internal/auth"
	"github.com/naruto361/command-line-login-system/internal/cli"
	"github.com/naruto361/command-line-login-system/internal/config"
	"github.com/naruto361/command-line-login-system/internal/email"
	"github.com/naruto361/command-line-login-system/internal/session"
	"github.com/naruto361/command-line-login-system/internal/store"
)

func main() {
	// Load runtime settings from environment variables (see docker-compose.yml).
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "configuration error: %v\n", err)
		os.Exit(1)
	}

	// Open SQLite and apply embedded schema on first run.
	db, err := store.Open(cfg.DatabasePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "database error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	mailer := email.NewSender(cfg)
	authSvc := auth.NewService(db, cfg, mailer)
	// Sessions live in memory only — suitable for a single interactive CLI process.
	sess := session.NewManager(cfg.SessionTimeout, cfg.SessionWarningBefore)
	app := cli.NewApp(authSvc, sess)

	app.Run()
}
