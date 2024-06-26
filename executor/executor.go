package executor

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"

	"github.com/FlipTheBoard/s3-backup-manager/config"
)

var cmdMutex sync.Mutex

type Executor struct {
	config   *config.Config
	uploader *s3manager.Uploader
	log      *zerolog.Logger
}

type Option = func(*Executor)

func WithConfig(cfg *config.Config) Option {
	return func(e *Executor) {
		e.config = cfg
	}
}

func WithUploader(u *s3manager.Uploader) Option {
	return func(e *Executor) {
		e.uploader = u
	}
}

func New(ctx context.Context, opts ...Option) *Executor {
	log := zlog.Ctx(ctx).
		With().
		Str("module", "executor").
		Logger()

	e := &Executor{log: &log}

	for _, opt := range opts {
		opt(e)
	}

	return e
}

func (e *Executor) Run(ctx context.Context) {
	e.log.Info().
		Msg("backup manager started")

	for name, backup := range e.config.Backups {
		go e.startBackupRunner(ctx, name, *backup)
	}
}

func (e *Executor) startBackupRunner(_ context.Context, name string, backup config.Backup) {
	log := e.log.With().
		Str("backup_name", name).
		Dur("duration", backup.Interval).
		Logger()

	log.Info().Msg("starting backup runner...")

	t := time.NewTicker(backup.Interval)
	defer t.Stop()

	for ; true; <-t.C {
		log.Debug().Msg("running backup...")
		cmdMutex.Lock()
		path := formatPath(backup.Path, name)

		for _, command := range backup.Commands {
			cmd := formatCommand(command, path)

			cmdFmt := exec.Command("/bin/bash", "-c", cmd)
			output, err := cmdFmt.CombinedOutput()
			if err != nil {
				log.Err(err).
					Bytes("stdout", output).
					Str("command", command).
					Send()

				continue
			}

			log.Info().
				Str("command", command).
				Msg("success")
		}

		if err := e.uploadToS3(name, path); err != nil {
			log.Err(err).Send()
		}

		cmdFmt := exec.Command("rm", path)
		output, err := cmdFmt.CombinedOutput()
		if err != nil {
			log.Err(err).
				Bytes("stdout", output).
				Str("path", path).
				Send()
		}

		cmdMutex.Unlock()

		log.Info().Msg("backup finished")
	}
}

func (e *Executor) uploadToS3(bucket, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}

	defer f.Close()

	fName := filepath.Base(path)
	_, err = e.uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Body:   f,
		Key:    &fName,
	})

	return err
}

func formatPath(path string, name string) string {
	match := map[string]func() string{
		"{dt}":   func() string { return time.Now().Format("2006-01-02_15:04:05") },
		"{name}": func() string { return name },
	}

	for key, fn := range match {
		path = strings.ReplaceAll(path, key, fn())
	}

	return path
}

func formatCommand(cmd string, path string) string {
	return strings.ReplaceAll(cmd, "{path}", path)
}
