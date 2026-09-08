// Package entrypoint wires the API and the bot together for the service
// binaries. It exists so cmd/main.go (the unified deployment) and cmd/api (the
// API-capable binary) share identical startup/shutdown behavior: the API comes
// up first, then the bot is attached so dashboard endpoints that need a live
// Discord session (server setup) work even when the API binary is the only
// process running.
package entrypoint

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/kibetnathan/minjibot/internal/api"
	"github.com/kibetnathan/minjibot/internal/bot"
)

// Run starts the API and, when attachBot is true, boots the bot too and wires
// its session + command handler into the API so server-setup endpoints work.
// Bot failures are logged and non-fatal: the API stays up (setup endpoints will
// report 503) for healthchecks. Run blocks until SIGINT/SIGTERM and then shuts
// everything down. It returns an error only if the API itself cannot start.
func Run(attachBot bool) error {
	apiApp, err := api.NewApp()
	if err != nil {
		return err
	}

	// Start the API in a goroutine so the healthcheck responds as soon as
	// possible, regardless of bot state.
	go func() {
		if err := apiApp.Start(); err != nil {
			apiApp.Echo.Logger.Error("API error", "error", err)
		}
	}()

	// Initialize the bot. Failures here (e.g. missing/invalid DISCORD_TOKEN or
	// a database that isn't ready yet) are logged rather than fatal, so the API
	// stays up for healthchecks and can report the service as degraded.
	var botApp *bot.App
	if attachBot {
		botApp, err = bot.NewApp()
		if err != nil {
			apiApp.Echo.Logger.Error("Failed to initialize bot (server setup disabled)", "error", err.Error())
		} else {
			// Give the API access to the bot's session + command handler so the
			// server-setup endpoints can provision guilds.
			apiApp.AttachBot(botApp.Session, botApp.CommandHandler())
			go func() {
				if err := botApp.Start(); err != nil {
					botApp.Logger.Error("Bot error", "error", err.Error())
				}
			}()
		}
	}

	// Channel for graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	apiApp.Echo.Logger.Info("API is running (bot may be degraded). Press CTRL-C to exit.")

	// Block until interrupt signal
	<-stop
	apiApp.Echo.Logger.Info("Shutting down gracefully...")

	// Shutdown API
	if err := apiApp.Shutdown(context.Background()); err != nil {
		apiApp.Echo.Logger.Error("Error shutting down API", "error", err)
	}

	// Shutdown bot (if it was initialized)
	if attachBot && botApp != nil {
		if err := botApp.Session.Close(); err != nil {
			botApp.Logger.Error("Error closing Discord session", "error", err.Error())
		}
		botApp.Pool.Close()
	}

	apiApp.Pool.Close()
	apiApp.Echo.Logger.Info("Shutdown complete")
	return nil
}
