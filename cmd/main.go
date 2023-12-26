package main

import (
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"

	"github.com/FlipTheBoard/s3-backup-manager/config"
	"github.com/FlipTheBoard/s3-backup-manager/executor"
)

func main() {
	cfg, err := config.ParseConfig()
	if err != nil {
		zlog.Fatal().Err(err).Send()
	}

	ctx := context.Background()
	log := zlog.
		Output(zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) { w.TimeFormat = time.RFC3339Nano })).
		Level(cfg.LoggingLevel)

	ctx = log.WithContext(ctx)

	if err = config.Log(ctx, cfg); err != nil {
		log.Fatal().Err(err).Send()
	}

	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	region := "ru-1"
	endpoint := "https://s3.ru-1.storage.selcloud.ru"
	s := session.Must(
		session.NewSession(
			&aws.Config{
				Credentials: credentials.NewEnvCredentials(),
				Endpoint:    &endpoint,
				Region:      &region,
			},
		),
	)
	uploader := s3manager.NewUploader(s)

	e := executor.NewExecutor(ctx, uploader, cfg)
	if err = e.Run(ctx); err != nil {
		log.Fatal().Err(err).Send()
	}

	select {
	case <-ctx.Done():
		log.Info().Msg("service shutting down")
	}

}
