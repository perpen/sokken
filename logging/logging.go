package logging

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/natefinch/lumberjack.v2"
)

func ParseFlags() *zerolog.Logger {
	debug := flag.Bool("log-debug", false, "sets log level to DEBUG rather than INFO")
	pretty := flag.Bool("log-pretty", false, "logs to console, in colourful non-json format - overrides log-file option")
	file := flag.String("log-file", "", "sets path of log file, if absent log to stderr")
	maxSizeMb := flag.Int("log-max-size-mb", 10, "max file size before rotation")
	maxAgeDays := flag.Int("log-max-age-days", 1, "max file age before rotation")
	maxRotated := flag.Int("log-max-rotated", 7, "number of rotated files to keep")
	flag.Parse()

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	var writer io.Writer
	if *pretty {
		writer = zerolog.ConsoleWriter{Out: os.Stderr}
	} else if *file != "" {
		fmt.Printf("logging to %v\n", *file)
		writer = &lumberjack.Logger{
			Filename:   *file,
			MaxSize:    *maxSizeMb,
			MaxAge:     *maxAgeDays,
			MaxBackups: *maxRotated,
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
