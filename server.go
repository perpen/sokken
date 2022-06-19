package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"nhooyr.io/websocket"
)

// Starts an http server serving the health endpoint and requests for tunnels
func runServer(apiAddr string, targetAddrs []string) error {
	log.Info().Msgf("listening on %v, tunnelling to: %v", apiAddr, targetAddrs)
	log.Info().Msgf("max connections: %v", maxActiveConns)

	srv := &http.Server{
		Addr:         apiAddr,
		ReadTimeout:  time.Second * 5,
		WriteTimeout: time.Second * 5,
	}

	http.HandleFunc("/health", healthEndpoint)
	http.Handle("/tunnel/", sokkenServer{
		targetAddrs: targetAddrs,
	})

	return srv.ListenAndServe()
}

type sokkenServer struct {
	targetAddrs []string
}

// Serves a tunnel request
func (s sokkenServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Make logger
	reqTargetAddr := strings.TrimPrefix(r.URL.Path, "/tunnel/")
	loggerFields := log.With().
		Str("target", reqTargetAddr).
		Str("client", r.RemoteAddr)
	xForwardedFor := r.Header["X-Forwarded-For"]
	if len(xForwardedFor) > 0 {
		loggerFields = loggerFields.Str("x-forwarded-for", xForwardedFor[0])
	}
	logger := loggerFields.Logger()

	// Validate
	if activeConns >= maxActiveConns {
		logger.Error().Msgf("rejecting client since we have %v connections", maxActiveConns)
		http.Error(w, "too many proxied connections", http.StatusTooManyRequests)
		return
	}
	addrAllowed := false
	for _, targetAddr := range s.targetAddrs {
		if targetAddr == reqTargetAddr {
			addrAllowed = true
			break
		}
	}

	// Setup websocket
	remoteWs, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		Subprotocols: []string{sokkenSubprotocol},
	})
	if err != nil {
		logger.Error().Err(err).Msg("websocket.Accept error")
		http.Error(w, "websocket error", http.StatusInternalServerError) // pointless?
		return
	}
	if remoteWs.Subprotocol() != sokkenSubprotocol {
		remoteWs.Close(websocket.StatusPolicyViolation,
			fmt.Sprintf("client not using %v subprotocol", sokkenSubprotocol))
		return
	}
	// If the address is not allowed we send an error msg on the websocket.
	// I tried writing it in the http response, but this ws lib does not expose the
	// message to the client if the ws handshake fails.
	if !addrAllowed {
		logger.Warn().Msgf("target address not allowed: %v", reqTargetAddr)
		err = remoteWs.Close(websocket.StatusInternalError, "target address not allowed")
		if err != nil {
			logger.Error().Err(err).Msgf("")
		}
		return
	}

	// Pipe the connections
	err = dialAndPlumb(r.Context(), remoteWs, reqTargetAddr, logger)
	if err != nil {
		if websocket.CloseStatus(err) != websocket.StatusNormalClosure {
			logger.Error().Err(err).Msg("abnormal websocket close status")
		} else {
			logger.Error().Err(err).Msg("failed to dialAndPlumb")
		}
	}
}

func dialAndPlumb(ctx context.Context, remoteWs *websocket.Conn, addr string,
	logger zerolog.Logger) error {

	logger.Info().Msgf("dialing %v", addr)
	local, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	remote := websocket.NetConn(ctx, remoteWs, websocket.MessageBinary)

	plumb(remote, local, logger)
	return nil
}
