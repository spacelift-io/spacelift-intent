package server

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"net/http"
	"spacelift-intent-mcp/internal"
	"time"

	"github.com/go-chi/chi/v5"
)

type Server struct {
	httpServer        *http.Server
	router            *chi.Mux
	connectionHandler internal.ConnectionHandler
}

func NewServer(ctx context.Context, addr string, connectionHandler internal.ConnectionHandler) (*Server, error) {
	server := &Server{
		router:            chi.NewRouter(),
		connectionHandler: connectionHandler,
	}

	server.router.Get("/ws", NewWebsocketHandler(ctx, connectionHandler).Handle)
	server.router.Get("/health", server.healthHandler)
	server.router.Get("/executor", server.executorsHandler)

	server.httpServer = &http.Server{
		Addr:              addr,
		Handler:           server.router,
		ReadHeaderTimeout: 30 * time.Second,
	}

	return server, nil
}

func (s *Server) WithTLSConfig(tlsConfig *tls.Config) *Server {
	s.httpServer.TLSConfig = tlsConfig
	return s
}

func (s *Server) ListenAndServe() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) ListenAndServeTLS(certFile, keyFile string) error {
	return s.httpServer.ListenAndServeTLS(certFile, keyFile)
}

func (s *Server) Shutdown(ctx context.Context) error {
	err := s.connectionHandler.CloseConnections(ctx)
	return errors.Join(err, s.httpServer.Shutdown(ctx))
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: implement health check
	w.Write([]byte("OK"))
}

func (s *Server) executorsHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: Add proper JSON response structure instead of exposing internal map
	// TODO: Add authentication/authorization for this endpoint
	// TODO: Add pagination and filtering for large connection lists

	executor, err := s.connectionHandler.GetAvailableExecutor()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Write([]byte(executor))
}
