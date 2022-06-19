package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"nhooyr.io/websocket"
)

const maxActiveConns = 100

var activeConns = 0

type healthInfo struct {
	Connections     uint `json:"connections"`
	MaxConnections  uint `json:"max-connections"`
	PercentCapacity uint `json:"connections-capacity-percent"`
	// Addresses       []string `json:"addresses"`
}

func runServer(servingAddr string, localAddrs []string) error {
	log.Info().Msgf("listening on %v, tunnelling to: %v", servingAddr, localAddrs)
	log.Info().Msgf("max connections: %v", maxActiveConns)

	s := &http.Server{
		Addr:         servingAddr,
		ReadTimeout:  time.Second * 2,
		WriteTimeout: time.Second * 2,
	}

	http.Handle("/tunnel/", sokkenServer{
		log:         log.With().Logger(), // useful?
		targetAddrs: localAddrs,
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		info := healthInfo{
			Connections:     uint(activeConns),
			MaxConnections:  maxActiveConns,
			PercentCapacity: 100 * uint(activeConns) / maxActiveConns,
			// Addresses: localAddrs,
		}
		infoJson, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			http.Error(w, "unable to marshal health info", 500)
			return
		}
		fmt.Fprintf(w, "%v\n", string(infoJson))
	})

	return s.ListenAndServe()
}

type sokkenServer struct {
	log         zerolog.Logger
	targetAddrs []string
}

func (s sokkenServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Make logger
	reqTargetAddr := strings.TrimPrefix(r.URL.Path, "/tunnel/")
	loggerFields := s.log.With().
		Str("target", reqTargetAddr).
		Str("client", r.RemoteAddr)
	xForwardedFor := r.Header["X-Forwarded-For"]
	if len(xForwardedFor) > 0 {
		loggerFields = loggerFields.Str("x-forwarded-for", xForwardedFor[0])
	}
	logger := loggerFields.Logger()

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
	remoteWs, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		Subprotocols: []string{sokkenSubprotocol},
	})
	if err != nil {
		logger.Error().Msgf("websocket.Accept: %v", err)
		http.Error(w, "websocket error", http.StatusInternalServerError) // pointless?
		return
	}
	if remoteWs.Subprotocol() != sokkenSubprotocol {
		remoteWs.Close(websocket.StatusPolicyViolation,
			fmt.Sprintf("client not using %v subprotocol", sokkenSubprotocol))
		return
	}
	if !addrAllowed {
		logger.Warn().Msgf("target address not allowed: %v", reqTargetAddr)
		err = remoteWs.Close(websocket.StatusInternalError, "target address not allowed")
		if err != nil {
			logger.Error().Msgf("%v", err)
		}
		return
	}

	activeConns += 1
	logger.Info().Int64("active-connections", int64(activeConns)).Msg("") //pointless?
	err = dialAndPlumb(r.Context(), remoteWs, reqTargetAddr, logger)
	activeConns += -1

	if err != nil {
		if websocket.CloseStatus(err) != websocket.StatusNormalClosure {
			logger.Error().Msgf("abnormal websocket close status: %v", err)
		} else {
			logger.Error().Msgf("failed to dialAndPlumb: %v", err)
		}
	}
}

func dialAndPlumb(ctx context.Context, remoteWs *websocket.Conn,
	addr string, logger zerolog.Logger) error {

	logger.Info().Msgf("dialing %v", addr)
	local, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	remote := websocket.NetConn(ctx, remoteWs, websocket.MessageBinary)

	plumb(remote, local, logger)

	// No need to close the websocket, it was closed when plumb() closed the NetConn
	// return remote.Close()

	return nil
}
