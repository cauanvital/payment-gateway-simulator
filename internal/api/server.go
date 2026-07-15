package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/cauanvital/payment-gateway-simulator/internal/config"
)

// Server encapsula o *http.Server e provê start/shutdown graciosos.
type Server struct {
	httpServer *http.Server
	logger     *slog.Logger
}

// NewServer cria o servidor HTTP a partir da configuração e do handler.
func NewServer(cfg config.ServerConfig, handler http.Handler, logger *slog.Logger) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:         ":" + cfg.Port,
			Handler:      handler,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
		},
		logger: logger,
	}
}

// Start inicia o servidor e bloqueia até o servidor parar. Retorna nil
// em caso de shutdown gracioso.
func (s *Server) Start() error {
	s.logger.Info("servidor HTTP iniciado", "addr", s.httpServer.Addr)
	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Shutdown encerra o servidor de forma graciosa respeitando o contexto.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("encerrando servidor HTTP")
	return s.httpServer.Shutdown(ctx)
}
