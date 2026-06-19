package postgres

//nolint:revive
import (
	"fmt"
	"net"
	"oil/config"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog/log"
)

const (
	postgresMaxIdleConnection = 10
	postgresMaxOpenConnection = 10
)

type Connection struct {
	Read  *sqlx.DB
	Write *sqlx.DB
}

func New(config *config.Config) *Connection {
	return &Connection{
		Read:  CreatePostgresReadConn(*config),
		Write: CreatePostgresWriteConn(*config),
	}
}

// getDBName returns the database name with prefix if configured
func getDBName(config config.Config, baseName string) string {
	if config.DB.Postgres.Prefix != "" {
		return config.DB.Postgres.Prefix + baseName
	}

	return baseName
}

// CreatePostgresWriteConn creates a database connection for write access.
func CreatePostgresWriteConn(config config.Config) *sqlx.DB {
	return CreatePostgresConnection(
		"write",
		config.DB.Postgres.Write.Username,
		config.DB.Postgres.Write.Password,
		config.DB.Postgres.Write.Host,
		config.DB.Postgres.Write.Port,
		getDBName(config, config.DB.Postgres.Write.Name),
		config.DB.Postgres.Write.SSLMode,
		config.DB.Postgres.MaxRetry,
		config.DB.Postgres.RetryWaitTime,
	)
}

// CreatePostgresReadConn creates a database connection for read access.
func CreatePostgresReadConn(config config.Config) *sqlx.DB {
	return CreatePostgresConnection(
		"read",
		config.DB.Postgres.Read.Username,
		config.DB.Postgres.Read.Password,
		config.DB.Postgres.Read.Host,
		config.DB.Postgres.Read.Port,
		getDBName(config, config.DB.Postgres.Read.Name),
		config.DB.Postgres.Read.SSLMode,
		config.DB.Postgres.MaxRetry,
		config.DB.Postgres.RetryWaitTime,
	)
}

// CreatePostgresConnection creates a database connection.
func CreatePostgresConnection(name, username, password, host, port, dbName, sslMode string, maxRetry, waitTime int) *sqlx.DB {
	descriptor := fmt.Sprintf(
		"postgres://%s:%s@%s/%s?sslmode=%s",
		username,
		password,
		net.JoinHostPort(host, port),
		dbName,
		sslMode,
	)

	for retry := range maxRetry {
		sqlDB, err := sqlx.Connect("postgres", descriptor)
		if err == nil {
			log.
				Info().
				Str("name", name).
				Str("host", host).
				Str("port", port).
				Str("dbName", dbName).
				Msg("Connected to database")
			sqlDB.SetMaxIdleConns(postgresMaxIdleConnection)
			sqlDB.SetMaxOpenConns(postgresMaxOpenConnection)

			return sqlDB
		}

		log.
			Error().
			Err(err).
			Str("name", name).
			Str("host", host).
			Str("port", port).
			Str("dbName", dbName).
			Int("attempt", retry+1).
			Msg("Failed connecting to database, retrying")

		time.Sleep(time.Duration(waitTime) * time.Second)
	}

	return nil
}
