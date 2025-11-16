package prreviewerassignment

import (
	"log/slog"
	"os"

	"github.com/pacahar/pr-reviewer-assignment/internal/config"
	"github.com/pacahar/pr-reviewer-assignment/internal/constants"
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
