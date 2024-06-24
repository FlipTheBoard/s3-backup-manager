package main

import (
	"context"
	"errors"
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
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

	err = run(ctx, cfg)

	switch {
	case errors.Is(err, context.Canceled):
		log.Info().Msg("gracefully stopped")
	case err != nil:
		log.Fatal().Err(err).Msg("unexpectedly terminated")
	}
}

func run(ctx context.Context, cfg *config.Config) error {
	s, err := session.NewSession(
		&aws.Config{
			Credentials: credentials.NewEnvCredentials(),
			Endpoint:    &cfg.S3.Endpoint,
			Region:      &cfg.S3.Region,
		},
	)
	if err != nil {
		return fmt.Errorf("s3 session: %w", err)
	}

	executor.New(ctx,
		executor.WithConfig(cfg),
		executor.WithUploader(s3manager.NewUploader(s)),
	).Run(ctx)

	<-ctx.Done()

	return nil
}
