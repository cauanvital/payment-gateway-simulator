package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/cauanvital/payment-gateway-simulator/internal/models"
	"github.com/cauanvital/payment-gateway-simulator/internal/payment"
	"github.com/cauanvital/payment-gateway-simulator/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type PaymentService interface {
	Create(
		ctx context.Context,
		terminalSerial string,
		amount int64,
		currency string,
		method models.PaymentMethod,
		card string,
	) (*models.Transaction, error)
	Get(ctx context.Context, transactionID uuid.UUID) (*models.Transaction, []models.TransactionEvent, error)
	Capture(ctx context.Context, id uuid.UUID) (*models.Transaction, error)
	Refund(ctx context.Context, id uuid.UUID) (*models.Transaction, error)
}

type TransactionHandler struct {
	service PaymentService
	logger  *slog.Logger
}

func NewTransactionHandler(service PaymentService, logger *slog.Logger) *TransactionHandler {
	return &TransactionHandler{service: service, logger: logger}
}

func (h *TransactionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	transaction, err := h.service.Create(
		r.Context(),
		req.TerminalSerial,
		req.Amount,
		req.Currency,
		req.PaymentMethod,
		req.Card,
	)
	if err != nil {
		h.handleError(w, err)
		return
	}

	respondJSON(w, http.StatusCreated, newTransactionResponse(*transaction))
}

func (h *TransactionHandler) Get(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")

	id, err := uuid.Parse(rawID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid transaction id")
		return
	}

	transaction, events, err := h.service.Get(r.Context(), id)
	if err != nil {
		h.handleError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"transaction": transaction,
		"events":      events,
	})
}

func (h *TransactionHandler) Capture(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")

	id, err := uuid.Parse(rawID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid transaction id")
		return
	}

	transaction, err := h.service.Capture(r.Context(), id)
	if err != nil {
		h.handleError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, transaction)
}

func (h *TransactionHandler) Refund(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")

	id, err := uuid.Parse(rawID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid transaction id")
		return
	}

	transaction, err := h.service.Refund(r.Context(), id)
	if err != nil {
		h.handleError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, transaction)
}

func (h *TransactionHandler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrTerminalNotFound):
		respondError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrTransactionNotFound):
		respondError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrTerminalBlocked):
		respondError(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, payment.ErrInvalidTransition):
		respondError(w, http.StatusUnauthorized, err.Error())
	default:
		h.logger.Error("unexpected error", "error", err)
		respondError(w, http.StatusInternalServerError, "internal server error")
	}
}

type createTransactionRequest struct {
	TerminalSerial string               `json:"terminal_serial"`
	Amount         int64                `json:"amount"`
	Currency       string               `json:"currency"`
	PaymentMethod  models.PaymentMethod `json:"payment_method"`
	Card           string               `json:"card"`
}

type transactionResponse struct {
	ID            uuid.UUID                `json:"id"`
	MerchantID    uuid.UUID                `json:"merchant_id"`
	TerminalID    uuid.UUID                `json:"terminal_id"`
	Amount        int64                    `json:"amount"`
	Currency      string                   `json:"currency"`
	PaymentMethod models.PaymentMethod     `json:"payment_method"`
	Status        models.TransactionStatus `json:"status"`
	CreatedAt     time.Time                `json:"created_at"`
	UpdatedAt     time.Time                `json:"updated_at"`
}

func newTransactionResponse(m models.Transaction) transactionResponse {
	return transactionResponse{
		ID:            m.ID,
		MerchantID:    m.MerchantID,
		TerminalID:    m.TerminalID,
		Amount:        m.Amount,
		Currency:      m.Currency,
		PaymentMethod: m.PaymentMethod,
		Status:        m.Status,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}
}

type transactionEventResponse struct {
	ID        uuid.UUID       `json:"id"`
	Event     string          `json:"event"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
}

func newTransactionEventResponse(m models.TransactionEvent) transactionEventResponse {
	return transactionEventResponse{
		ID:        m.ID,
		Event:     m.Event,
		Payload:   m.Payload,
		CreatedAt: m.CreatedAt,
	}
}
