package helper

//nolint:revive
import (
	"errors"
	"fmt"
	"net"
	"oil/config"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/rs/zerolog/log"
)

func getDBName(config *config.Config, baseName string) string {
	if config.DB.Postgres.Prefix != "" {
		return config.DB.Postgres.Prefix + baseName
	}

	return baseName
}

func getConnection(config *config.Config) (*migrate.Migrate, error) {
	connectionString := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s&x-migrations-table=%s",
		config.DB.Postgres.Write.Username,
		config.DB.Postgres.Write.Password,
		net.JoinHostPort(config.DB.Postgres.Write.Host, config.DB.Postgres.Write.Port),
		getDBName(config, config.DB.Postgres.Write.Name),
		config.DB.Postgres.Write.SSLMode,
		config.DB.Postgres.MigrationTable,
	)

	mig, err := migrate.New(
		"file://migrations/postgres",
		connectionString,
	)

	if err != nil {
		return nil, fmt.Errorf("error creating migrate instance: %w", err)
	}

	return mig, nil
}

func Runner(config *config.Config, action string) error {
	mig, err := getConnection(config)
	if err != nil {
		return fmt.Errorf("error creating migrate instance: %w", err)
	}

	defer mig.Close()

	switch action {
	case "up":
		if err := mig.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return fmt.Errorf("error running migrations: %w", err)
		}

		log.Info().Msg("Database migrations completed successfully")

		return nil
	case "down":
		if err := mig.Steps(-1); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return fmt.Errorf("error rolling back migrations: %w", err)
		}

		log.Info().Msg("Database migrations rolled back successfully")

		return nil
	case "step-up":
		if err := mig.Steps(1); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return fmt.Errorf("error running migrations: %w", err)
		}

		log.Info().Msg("Database migrations completed successfully")

		return nil
	case "drop":
		if err := mig.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return fmt.Errorf("error rolling back migrations: %w", err)
		}

		log.Info().Msg("Database migrations rolled back successfully")

		return nil
	}

	return nil
}

func Up(config *config.Config) error {
	return Runner(config, "up")
}

func StepUp(config *config.Config) error {
	return Runner(config, "step-up")
}

func Down(config *config.Config) error {
	return Runner(config, "down")
}

func Drop(config *config.Config) error {
	return Runner(config, "drop")
}
