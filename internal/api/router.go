// Package api monta o roteador HTTP e conecta middlewares e rotas.
package api

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/cauanvital/payment-gateway-simulator/internal/handlers"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

type Handlers struct {
	Merchant    *handlers.MerchantHandler
	Terminal    *handlers.TerminalHandler
	Transaction *handlers.TransactionHandler
}

// Router constrói o *chi.Mux com os middlewares base e as rotas
// registradas. Handlers de domínio serão adicionados aqui conforme o
// projeto evoluir.
func Router(logger *slog.Logger, h Handlers) http.Handler {
	r := chi.NewRouter()

	// Middlewares base. Idempotência e request-scoped logging entram
	// nas próximas etapas do projeto.
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(30 * time.Second))

	// Health check — não depende de banco, apenas indica que o processo
	// está de pé.
	r.Get("/health", healthHandler)
	r.Get("/healthz", healthHandler)

	r.Route("/merchants", func(r chi.Router) {
		r.Post("/", h.Merchant.Create)
		r.Get("/", h.Merchant.List)
		r.Get("/{id}", h.Merchant.Get)
		r.Get("/{merchant_id}/terminals", h.Terminal.List)
	})
	r.Route("/terminals", func(r chi.Router) {
		r.Post("/", h.Terminal.Create)
		r.Post("/{id}/block", h.Terminal.Block)
		r.Post("/{id}/activate", h.Terminal.Activate)
		r.Get("/{id}", h.Terminal.Get)
	})
	r.Route("/transactions", func(r chi.Router) {
		r.Post("/", h.Transaction.Create)
		r.Get("/{id}", h.Transaction.Get)
		r.Post("/{id}/capture", h.Transaction.Capture)
		r.Post("/{id}/refund", h.Transaction.Refund)
	})

	return r
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
