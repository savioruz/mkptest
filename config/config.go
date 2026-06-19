package config

import (
	"fmt"
	"sync"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog/log"
)

type ConsumerConfig struct {
	ConsumerGroup   string `envconfig:"CONSUMER_GROUP"`
	MaxRetry        int    `envconfig:"MAX_RETRY"`
	BackoffDuration int    `envconfig:"BACKOFF_DURATION"` // seconds between retries
}

type Config struct {
	Server struct {
		Env      string `envconfig:"ENV"`
		LogLevel string `envconfig:"LOG_LEVEL"`
		Port     string `envconfig:"PORT"`
		Host     string `envconfig:"HOST"`
		Shutdown struct {
			CleanupPeriodSeconds int64 `envconfig:"CLEANUP_PERIOD_SECONDS"`
			GracePeriodSeconds   int64 `envconfig:"GRACE_PERIOD_SECONDS"`
		} `envconfig:"SHUTDOWN"`
	} `envconfig:"SERVER"`

	App struct {
		Name     string `envconfig:"APP_NAME"`
		Timezone string `envconfig:"TIMEZONE"`
		CORS     struct {
			AllowCredentials bool     `envconfig:"ALLOW_CREDENTIALS"`
			AllowedHeaders   []string `envconfig:"ALLOWED_HEADERS"`
			AllowedMethods   []string `envconfig:"ALLOWED_METHODS"`
			AllowedOrigins   []string `envconfig:"ALLOWED_ORIGINS"`
			Enable           bool     `envconfig:"ENABLE"`
			MaxAgeSeconds    int      `envconfig:"MAX_AGE_SECONDS"`
		} `envconfig:"CORS"`
		RateLimiter struct {
			Enable        bool `envconfig:"ENABLE"`
			MaxRequests   int  `envconfig:"MAX_REQUESTS"`
			WindowSeconds int  `envconfig:"WINDOW_SECONDS"`
		} `envconfig:"RATE_LIMITER"`
		APIKey string `envconfig:"API_KEY"`
	} `envconfig:"APP"`

	Cache struct {
		Redis struct {
			Primary struct {
				Host     string `envconfig:"HOST"`
				Port     string `envconfig:"PORT"`
				Password string `envconfig:"PASSWORD"`
				DB       int    `envconfig:"DB"`
				TLS      bool   `envconfig:"TLS"`
			} `envconfig:"PRIMARY"`
		} `envconfig:"REDIS"`
		TTL int `envconfig:"TTL"`
	} `envconfig:"CACHE"`

	JWT struct {
		AccessSecret     string `envconfig:"ACCESS_SECRET"`
		RefreshSecret    string `envconfig:"REFRESH_SECRET"`
		AccessExpireMin  int    `envconfig:"ACCESS_EXPIRE_MIN"`
		RefreshExpireMin int    `envconfig:"REFRESH_EXPIRE_MIN"`
	} `envconfig:"JWT"`

	DB struct {
		Postgres struct {
			MaxRetry       int    `envconfig:"MAX_RETRY"`
			RetryWaitTime  int    `envconfig:"RETRY_WAIT_TIME"`
			MigrationTable string `envconfig:"MIGRATION_TABLE"`
			AutoMigrate    bool   `envconfig:"AUTO_MIGRATE"`
			Prefix         string `envconfig:"PREFIX"`
			Read           struct {
				Host     string `envconfig:"HOST"`
				Port     string `envconfig:"PORT"`
				Username string `envconfig:"USER"`
				Password string `envconfig:"PASSWORD"`
				Name     string `envconfig:"NAME"`
				Timezone string `envconfig:"TIMEZONE"`
				SSLMode  string `envconfig:"SSL_MODE"`
			} `envconfig:"READ"`
			Write struct {
				Host     string `envconfig:"HOST"`
				Port     string `envconfig:"PORT"`
				Username string `envconfig:"USER"`
				Password string `envconfig:"PASSWORD"`
				Name     string `envconfig:"NAME"`
				Timezone string `envconfig:"TIMEZONE"`
				SSLMode  string `envconfig:"SSL_MODE"`
			} `envconfig:"WRITE"`
		} `envconfig:"POSTGRES"`
	} `envconfig:"DB"`

	External struct {
		S3 struct {
			APIEndpoint     string `envconfig:"API_ENDPOINT"`
			AccessKeyID     string `envconfig:"ACCESS_KEY_ID"`
			SecretAccessKey string `envconfig:"SECRET_ACCESS_KEY"`
			BucketName      string `envconfig:"BUCKET_NAME"`
			PublicDomain    string `envconfig:"PUBLIC_DOMAIN"`
		} `envconfig:"S3"`
		Otel struct {
			Endpoint string `envconfig:"ENDPOINT"`
		} `envconfig:"OTEL"`
	} `envconfig:"EXTERNAL"`

	Kafka struct {
		Brokers       []string `envconfig:"BROKERS"`
		ConsumerGroup string   `envconfig:"CONSUMER_GROUP"`
		Enable        bool     `envconfig:"ENABLE"`
		SASL          struct {
			Enable   bool   `envconfig:"ENABLE"`
			Username string `envconfig:"USERNAME"`
			Password string `envconfig:"PASSWORD"`
		} `envconfig:"SASL"`
		Topics struct {
			Notification string `envconfig:"NOTIFICATION"`
			Refund       string `envconfig:"REFUND"`
		} `envconfig:"TOPICS"`
		Notification ConsumerConfig `envconfig:"NOTIFICATION"`
		Refund       ConsumerConfig `envconfig:"REFUND"`
	} `envconfig:"KAFKA"`

	Asynq struct {
		Enable      bool `envconfig:"ENABLE"`
		Concurrency int  `envconfig:"CONCURRENCY"`
	} `envconfig:"ASYNQ"`

	Payment struct {
		MockMode        bool   `envconfig:"MOCK_MODE"`
		WebhookSecret   string `envconfig:"WEBHOOK_SECRET"`
		ExpireMinutes   int    `envconfig:"EXPIRE_MINUTES"`
		CallbackBaseURL string `envconfig:"CALLBACK_BASE_URL"`
	} `envconfig:"PAYMENT"`
}

var (
	conf        Config
	once        sync.Once
	initialized bool
)

func Init() error {
	var err error

	once.Do(func() {
		err = godotenv.Load(".env")
		if err != nil {
			log.Warn().Err(err).Msg("Could not load .env file, continuing with existing environment variables")
		} else {
			log.Info().Msg("Successfully loaded variables from .env file into environment")
		}

		err = envconfig.Process("", &conf)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to process environment variables")
		}

		initialized = true

		log.Info().Msg("Service configuration initialized successfully")
	})

	if err != nil {
		return fmt.Errorf("loading .env file: %w", err)
	}

	return nil
}

func Get() *Config {
	if !initialized {
		if err := Init(); err != nil {
			log.Fatal().Err(err).Msg("Failed to initialize configuration")
		}
	}

	return &conf
}
