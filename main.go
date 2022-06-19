package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const sokkenSubprotocol = "sokken"

var maxActiveConns int

var activeConns = 0

func main() {
	maxActiveConnsParam := flag.Int("max-connections", 100,
		"when reached further connections are rejected")
	logConfig := newLoggingConfig()
	flag.Parse()
	log.Logger = *logConfig.makeLogger()
	maxActiveConns = *maxActiveConnsParam

	log.Fatal().Err(run()).Msg("")
}

func run() error {
	usage := func() {
		basename := path.Base(os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), ""+
			"Usage: %v server [OPTION]... API_ADDR REMOTE_ADDR [REMOTE_ADDR_2...]\n"+
			"       %v client [OPTION]... API_ADDR LOCAL_ADDR REMOTE_ADDR [LOCAL_ADDR_2 REMOTE_ADDR_2 ...]\n"+
			"Options:\n",
			basename, basename)
		flag.PrintDefaults()
		os.Exit(2)
	}

	args := flag.Args()
	if len(args) == 1 {
		usage()
	}

	handleSignals()

	if args[0] == "server" {
		args = args[1:]
		if len(args) < 2 {
			usage()
		}
		apiAddr := args[0]
		targetAddrs := args[1:]
		return runServer(apiAddr, targetAddrs)
	} else if args[0] == "client" {
		args = args[1:]
		if len(args) < 3 {
			usage()
		}
		apiAddr := args[0]
		args = args[1:]
		if len(args)%2 != 0 {
			usage()
		}
		return runClient(apiAddr, args)
	}

	usage()
	return nil
}

// experiment
func handleSignals() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for {
			sig := <-sigs
			log.Fatal().Msgf("got signal '%v', exiting", sig)
			os.Exit(1)
		}
	}()
}

// Pipes the connections, logs the duration
func plumb(a, b net.Conn, logger zerolog.Logger) {
	activeConns += 1
	logger.Info().Int("active-connections", int(activeConns)).Msg("plumbing start")
	start := time.Now()

	push := func(tgt, src net.Conn) {
		if _, err := io.Copy(tgt, src); err != nil {
			logger.Info().Msgf("connection closed, io.Copy says: %v", err)
		}
		src.Close()
		tgt.Close()
	}
	go push(a, b)
	push(b, a)

	elapsed := time.Now().Sub(start)
	activeConns += -1
	logger.Info().
		Int("active-connections", int(activeConns)).
		Int64("plumbing-duration-ms", elapsed.Milliseconds()).
		Msgf("plumbing end, lasted %v", elapsed)
}

// Used by server and client
func healthEndpoint(w http.ResponseWriter, r *http.Request) {
	type healthInfo struct {
		Connections     uint    `json:"connections"`
		MaxConnections  uint    `json:"max-connections"`
		PercentCapacity float64 `json:"connections-used-percent"`
	}
	info := healthInfo{
		Connections:     uint(activeConns),
		MaxConnections:  uint(maxActiveConns),
		PercentCapacity: float64(activeConns) / float64(maxActiveConns),
	}
	infoJson, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		http.Error(w, "unable to marshal health info", 500)
		return
	}
	fmt.Fprintf(w, "%v\n", string(infoJson))
}
