package cli

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/getflaggy/flaggy/internal/api"
	"github.com/getflaggy/flaggy/internal/config"
	"github.com/getflaggy/flaggy/internal/sse"
	"github.com/getflaggy/flaggy/internal/store"
	"github.com/getflaggy/flaggy/migrations"
)

func init() {
	rootCmd.AddCommand(serveCmd)
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the Flaggy server",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()

		slog.Info("starting flaggy", "version", Version, "port", cfg.Port, "db", cfg.DBPath)

		db, err := store.NewSQLiteStore(cfg.DBPath, migrations.FS)
		if err != nil {
			slog.Error("failed to open database", "error", err)
			os.Exit(1)
		}
		defer db.Close()

		broadcaster := sse.NewBroadcaster()
		defer broadcaster.Close()

		if cfg.MasterKey == "" {
			slog.Warn("FLAGGY_MASTER_KEY not set — auth disabled (dev mode)")
		}

		router := api.NewRouter(db, broadcaster, cfg.MasterKey, cfg.CORSEnabled)

		srv := &http.Server{
			Addr:        cfg.Port,
			Handler:     router,
			ReadTimeout: 10 * time.Second,
			IdleTimeout: 120 * time.Second,
			// No WriteTimeout — SSE streams are long-lived connections
		}

		done := make(chan os.Signal, 1)
		signal.Notify(done, os.Interrupt, syscall.SIGTERM)

		go func() {
			slog.Info("server listening", "addr", cfg.Port)
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				slog.Error("server error", "error", err)
				os.Exit(1)
			}
		}()

		<-done
		slog.Info("shutting down...")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			slog.Error("shutdown error", "error", err)
		}

		slog.Info("server stopped")
		return nil
	},
}
