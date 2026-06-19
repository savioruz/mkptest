package logger

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"oil/config"
	"os"
	"time"
)

func InitLogger() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.TraceLevel)

	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}

	log.Logger = log.Output(output)
	log.Trace().Msg("Zerolog initialized.")
}

func ErrorWithStack(err error) {
	log.Error().Msgf("%+v", errors.WithStack(err))
}

func SetLogLevel(config *config.Config) {
	level, err := zerolog.ParseLevel(config.Server.LogLevel)
	if err != nil {
		level = zerolog.TraceLevel
		log.Trace().Str("loglevel", level.String()).Msg("Environment has no log level set up, using default.")
	} else {
		log.Trace().Str("loglevel", level.String()).Msg("Desired log level detected.")
	}

	zerolog.SetGlobalLevel(level)
}
