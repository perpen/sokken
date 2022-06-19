package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/natefinch/lumberjack.v2"
)

type loggingConfig struct{
	initialised bool
	debug *bool
	pretty *bool
	file *string
	maxSizeMb *int
	maxAgeDays *int
	maxRotated *int
}

func newLoggingConfig() loggingConfig {
	lc := loggingConfig{}
	lc.initialised = true
	lc.debug = flag.Bool("log-debug", false, "sets log level to DEBUG rather than INFO")
	lc.pretty = flag.Bool("log-pretty", false, "logs to console, in colourful non-json format - overrides log-file option")
	lc.file = flag.String("log-file", "", "sets path of log file, if absent log to stderr")
	lc.maxSizeMb = flag.Int("log-max-size-mb", 10, "max file size before rotation")
	lc.maxAgeDays = flag.Int("log-max-age-days", 1, "max file age before rotation")
	lc.maxRotated = flag.Int("log-max-rotated", 7, "number of rotated files to keep")
	return lc
}

func (lc *loggingConfig) makeLogger() *zerolog.Logger {
	if !lc.initialised {
		fmt.Fprintf(os.Stderr, "loggingConfig.makeLogger() called before flag.Parse()\n")
		os.Exit(1)
	}

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *lc.debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	var writer io.Writer
	if *lc.pretty {
		writer = zerolog.ConsoleWriter{Out: os.Stderr}
	} else if *lc.file != "" {
		fmt.Printf("logging to %v\n", *lc.file)
		writer = &lumberjack.Logger{
			Filename:   *lc.file,
			MaxSize:    *lc.maxSizeMb,
			MaxAge:     *lc.maxAgeDays,
			MaxBackups: *lc.maxRotated,
			LocalTime:  false,
			Compress:   false,
		}
	}

	logger := log.Logger
	if writer != nil {
		logger = zerolog.New(writer).With().Timestamp().Logger()
	}
	return &logger
}
