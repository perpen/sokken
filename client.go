package main

import (
	"context"
	"net"
	"net/http"

	"github.com/rs/zerolog/log"
	"nhooyr.io/websocket"
)

type sokkenTunnel struct {
	localAddr  string
	remoteAddr string
}

// `args` must be of even length
func runClient(apiAddr string, args []string) error {
	log.Info().Msgf("health endpoint listening on %v", apiAddr)
	log.Info().Msgf("max connections: %v", maxActiveConns)

	tunnels := make([]sokkenTunnel, len(args)/2)
	for i := 0; i < len(args); i += 2 {
		tunnel := sokkenTunnel{
			localAddr:  args[i],
			remoteAddr: args[i+1],
		}
		tunnels[i/2] = tunnel
	}
	err := setupPorts(tunnels)
	if err != nil {
		return err
	}

	http.HandleFunc("/health", healthEndpoint)
	return http.ListenAndServe(apiAddr, nil)
}

func setupPorts(tunnels []sokkenTunnel) error {
	for _, tunnel := range tunnels {
		log.Info().Msgf("tunnelling %v to %v",
			tunnel.localAddr, tunnel.remoteAddr)
		ln, err := net.Listen("tcp", tunnel.localAddr)
		if err != nil {
			return err
		}
		go tunnel.listen(ln)
	}
	return nil
}

// Listen on a local port, pipe connections through a websocket.
// If unable to accept connection, pushes error to channel.
func (c sokkenTunnel) listen(ln net.Listener) {
	for {
		local, err := ln.Accept()
		if err != nil {
			log.Error().Err(err).Msg("Accept error")
			continue
		}
		if activeConns >= maxActiveConns {
			log.Error().Msgf("rejecting client since we have %v connections", maxActiveConns)
			local.Close()
			return
		}
		go c.tunnel(local, c.remoteAddr)
	}
}

func (c sokkenTunnel) tunnel(local net.Conn, remoteAddr string) {
	localAddr := local.RemoteAddr().String()
	logger := log.With().Str("local", localAddr).Str("remote", remoteAddr).Logger()
	logger.Info().Msgf("tunnel request")

	ctx := context.Background()
	remoteWs, _, err := websocket.Dial(ctx, remoteAddr, &websocket.DialOptions{
		Subprotocols: []string{sokkenSubprotocol},
	})
	if err != nil {
		log.Error().Err(err).Msg("Dial error")
		local.Close()
		return
	}

	remote := websocket.NetConn(ctx, remoteWs, websocket.MessageBinary)
	plumb(remote, local, logger)
}
