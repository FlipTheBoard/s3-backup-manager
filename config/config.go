package config

import (
	"context"
	"encoding/json"
	"time"

	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type Config struct {
	LoggingLevel zerolog.Level `mapstructure:"logging_level"`
	Backups      map[string]*Backup
}

type Backup struct {
	Interval time.Duration
	Path     string
	Commands []string
}

func ParseConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
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
