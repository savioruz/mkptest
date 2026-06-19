package timezone

import (
	"oil/config"
	"time"

	"github.com/rs/zerolog/log"
)

var (
	appLocation *time.Location
)

func init() {
	cfg := config.Get()

	if cfg.App.Timezone == "" {
		log.Warn().Msg("No timezone configured, using UTC as default")

		cfg.App.Timezone = "UTC"
	}

	loc, err := time.LoadLocation(cfg.App.Timezone)
	if err != nil {
		log.Error().
			Err(err).
			Str("timezone", cfg.App.Timezone).
			Msg("Failed to load timezone, falling back to UTC. Please use standard timezone names like 'Asia/Jakarta', 'UTC', 'America/New_York'")

		appLocation = time.UTC

		return
	}

	appLocation = loc
	log.Info().
		Str("timezone", cfg.App.Timezone).
		Str("location", loc.String()).
		Msg("Application timezone initialized")
}

// Now returns the current time in the application timezone
func Now() time.Time {
	if appLocation == nil {
		log.Warn().Msg("Timezone not initialized, using UTC")

		return time.Now().UTC()
	}

	return time.Now().In(appLocation)
}

// ToAppTime converts a time to the application timezone
func ToAppTime(t time.Time) time.Time {
	if appLocation == nil {
		log.Warn().Msg("Timezone not initialized, using UTC")

		return t.UTC()
	}

	return t.In(appLocation)
}

// GetLocation returns the current application timezone location
func GetLocation() *time.Location {
	if appLocation == nil {
		log.Warn().Msg("Timezone not initialized, returning UTC")

		return time.UTC
	}

	return appLocation
}

// Parse parses a time string in the application timezone
func Parse(layout, value string) (time.Time, error) {
	if appLocation == nil {
		log.Warn().Msg("Timezone not initialized, parsing in UTC")

		return time.Parse(layout, value)
	}

	return time.ParseInLocation(layout, value, appLocation)
}

// Format formats a time in the application timezone
func Format(t time.Time, layout string) string {
	return ToAppTime(t).Format(layout)
}
