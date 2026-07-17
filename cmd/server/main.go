// Command server é o ponto de entrada do gateway de pagamentos simulado.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cauanvital/payment-gateway-simulator/internal/api"
	"github.com/cauanvital/payment-gateway-simulator/internal/config"
	"github.com/cauanvital/payment-gateway-simulator/internal/database"
	"github.com/cauanvital/payment-gateway-simulator/internal/database/sqlc"
	"github.com/cauanvital/payment-gateway-simulator/internal/handlers"
	"github.com/cauanvital/payment-gateway-simulator/internal/repository"
	"github.com/cauanvital/payment-gateway-simulator/internal/service"
)

func main() {
	if err := run(); err != nil {
		slog.Error("falha ao iniciar aplicação", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := config.NewLogger(cfg.Log)
	logger.Info("iniciando aplicação", "env", cfg.Env)

	connectCtx, cancelConnect := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelConnect()

	pool, err := database.NewPool(connectCtx, cfg.DB)
	if err != nil {
		return fmt.Errorf("conectando ao banco: %w", err)
	}
	defer pool.Close()

	queries := sqlc.New(pool)

	merchantRepo := repository.NewMerchantRepository(queries)
	merchantService := service.NewMerchantService(merchantRepo)
	merchantHandler := handlers.NewMerchantHandler(merchantService, logger)

	terminalRepo := repository.NewTerminalRepository(queries)
	terminalService := service.NewTerminalService(terminalRepo)
	terminalHandler := handlers.NewTerminalHandler(terminalService, logger)

	handler := api.Router(logger, api.Handlers{
		Merchant: merchantHandler,
		Terminal: terminalHandler,
	})
	server := api.NewServer(cfg.Server, handler, logger)

	// Captura SIGINT/SIGTERM para shutdown gracioso.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Start()
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		stop() // para de interceptar sinais; um segundo sinal encerra à força
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	}
}
