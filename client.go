package main

import (
	"context"
	"net"

	"github.com/rs/zerolog/log"
	"nhooyr.io/websocket"
)

func runClients(clients []sokkenClient) error {
	errc := make(chan error, 1) // used to detect issue with any client
	for _, client := range clients {
		log.Info().Msgf("tunnelling %v to %v",
			client.localAddr, client.remoteAddr)
		go client.runClient(errc)
	}
	err := <-errc
	return err
}

type sokkenClient struct {
	localAddr  string
	remoteAddr string
}

// Listen on a local port, pipe connections to a remote websocket.
// If fatal error, pushes it to channel.
func (c sokkenClient) runClient(errc chan error) {
	ln, err := net.Listen("tcp", c.localAddr)
	if err != nil {
		errc <- err
		return
	}
	for {
		local, err := ln.Accept()
		if err != nil {
			log.Info().Msgf("oops: %v", err)
			continue
		}
		go c.handleConn(local, c.remoteAddr)
	}
}

func (c sokkenClient) handleConn(local net.Conn, remoteAddr string) {
	localAddr := local.RemoteAddr().String()

	logger := log.With().Str("local", localAddr).Str("remote", remoteAddr).Logger()
	logger.Info().Msgf("tunnel request")

	ctx := context.Background()
	remoteWs, _, err := websocket.Dial(ctx, remoteAddr, &websocket.DialOptions{
		Subprotocols: []string{sokkenSubprotocol},
	})
	if err != nil {
		log.Error().Msgf("Dial error: %v", err)
		local.Close()
		return
	}

	remote := websocket.NetConn(ctx, remoteWs, websocket.MessageBinary)
	plumb(remote, local, logger)

	if false {
		// Not necessary as plumb() closed the NetConn, which caused the websocket
		// to close.
		logger.Info().Msg("client: closing websocket")
		err = remoteWs.Close(websocket.StatusNormalClosure, "")
		if err != nil {
			logger.Error().Msgf("%v", err)
		}
	}
}
