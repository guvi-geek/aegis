package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// StartServer starts the Gin HTTP server on the specified port
// It runs the server in a goroutine and returns the http.Server instance for graceful shutdown
func StartServer(router *gin.Engine, port string) *http.Server {
	addr := fmt.Sprintf(":%s", port)

	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		log.Info().Str("port", port).Str("address", addr).Msg("Starting Gin HTTP server")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	return srv
}

// ShutdownServer gracefully shuts down the HTTP server
// It waits for the specified timeout for existing connections to close
func ShutdownServer(srv *http.Server, timeout time.Duration) error {
	log.Info().Msg("Shutting down HTTP server...")

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}

	log.Info().Msg("HTTP server shutdown complete")
	return nil
}
