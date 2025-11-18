package main

import (
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"github.com/pacahar/pr-reviewer-assignment/internal/config"
	"github.com/pacahar/pr-reviewer-assignment/internal/constants"
	handlers "github.com/pacahar/pr-reviewer-assignment/internal/http"
	"github.com/pacahar/pr-reviewer-assignment/internal/storage/postgres"
)

func main() {
	config := config.MustLoad()

	log := setupLogger(config.Environment)

	storage, err := postgres.NewPostgresStorage(config.Database.DSN())
	if err != nil {
		log.Error("failed to initialize storage", slog.String("error", err.Error()))
		return
	}

	h := handlers.NewHandler(storage, log)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	srv := &http.Server{
		Addr:    ":" + strconv.Itoa(config.HTTPServer.Port),
		Handler: mux,
	}

	log.Info("server listening", slog.String("addr", srv.Addr))

	if err := srv.ListenAndServe(); err != nil {
		log.Error("server failed", slog.String("error", err.Error()))
	}
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case constants.EnvLocal:
		log = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case constants.EnvDev:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case constants.EnvProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	default:
		return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}

	return log
}
