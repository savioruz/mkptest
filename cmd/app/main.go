package main

import (
	"oil/config"
	"oil/di"
	"oil/shared/logger"

	"github.com/rs/zerolog/log"

	migration "oil/helper"
)

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-KEY
func main() {
	cfg := config.Get()

	logger.InitLogger()

	logger.SetLogLevel(cfg)

	if cfg.DB.Postgres.AutoMigrate {
		// Run migrations
		err := migration.Up(cfg)
		if err != nil {
			log.Error().Err(err).Msg("failed to run migrations")
		}
	}

	events := di.InitializeEvents()
	events.Start()

	workers := di.InitializeWorkers()
	workers.Start()

	http := di.InitializeService()
	http.Serve()
}
