package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/perpen/sokken/logging"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const sokkenSubprotocol = "sokken"

func main() {
	log.Logger = *logging.ParseFlags()
	err := run()
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}
}

func run() error {
	usage := func() {
		basename := path.Base(os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), ""+
			"Usage: %v server [OPTION]... LISTEN_ADDR REMOTE_ADDR [REMOTE_ADDR_2...]\n"+
			"       %v client [OPTION]... LISTEN_ADDR REMOTE_ADDR [LISTEN_ADDR_2 REMOTE_ADDR_2 ...]\n"+
			"Options:\n",
			basename, basename)
		flag.PrintDefaults()
		os.Exit(2)
	}

	args := flag.Args()
	if len(args) < 3 {
		usage()
	}

	handleSignals()

	if args[0] == "server" {
		addr := args[1]
		portArgs := args[2:]
		localAddrs := make([]string, len(portArgs))
		for i := range localAddrs {
			localAddrs[i] = portArgs[i]
		}
		return runServer(addr, localAddrs)
	} else if args[0] == "client" {
		args = args[1:]
		if len(args)%2 != 0 {
			usage()
		}
		clients := make([]sokkenClient, len(args)/2)
		for i := 0; i < len(args); i += 2 {
			clt := sokkenClient{
				args[i],
				args[i+1],
			}
			clients[i/2] = clt
		}
		return runClients(clients)
	} else {
		usage()
		return nil
	}
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

// Pipes the connections
func plumb(remote, local net.Conn, logger zerolog.Logger) {
	logger.Info().Msg("plumbing start")
	start := time.Now()

	push := func(tgt, src net.Conn) {
		if _, err := io.Copy(tgt, src); err != nil {
			logger.Info().Msgf("connection closed, io.Copy says: %v", err)
		}
		src.Close()
		tgt.Close()
	}
	go push(remote, local)
	push(local, remote)

	elapsed := time.Now().Sub(start)
	logger.Info().
		Int64("plumbing-duration-ms", elapsed.Milliseconds()).
		Msgf("plumbing end, lasted %v", elapsed)
}
