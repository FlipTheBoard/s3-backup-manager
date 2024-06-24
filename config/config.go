package config

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"time"

	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type Config struct {
	LoggingLevel zerolog.Level `mapstructure:"logging_level"`
	S3           S3Config
	Backups      map[string]*Backup
}

type S3Config struct {
	Region   string
	Endpoint string
}

type Backup struct {
	Interval time.Duration
	Path     string
	Commands []string
}

func ParseConfig() (*Config, error) {
	path, ok := os.LookupEnv("CONFIG_PATH")
	if !ok {
		return nil, errors.New("CONFIG_PATH env not found")
	}

	viper.SetConfigName("config")
	viper.AddConfigPath(path)
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}

	var config Config
	if err = viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, err
}

func Log(ctx context.Context, config *Config) error {
	log := zlog.Ctx(ctx)

	msg, err := json.MarshalIndent(config, "config ", "  ")
	if err != nil {
		return err
	}

	log.Debug().Msg(string(msg))

	return nil
}
