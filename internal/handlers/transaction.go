package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/cauanvital/payment-gateway-simulator/internal/models"
	"github.com/cauanvital/payment-gateway-simulator/internal/payment"
	"github.com/cauanvital/payment-gateway-simulator/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type TransactionService interface {
	Create(
		ctx context.Context,
		terminalSerial string,
		amount int64,
		currency string,
		method models.PaymentMethod,
		card string,
		idempotency service.IdempotencyRequest,
	) (*service.IdempotentResponse, error)
	Get(ctx context.Context, transactionID uuid.UUID) (*models.Transaction, []models.TransactionEvent, error)
	Capture(ctx context.Context, id uuid.UUID, idempotency service.IdempotencyRequest) (*service.IdempotentResponse, error)
	Refund(ctx context.Context, id uuid.UUID, idempotency service.IdempotencyRequest) (*service.IdempotentResponse, error)
}

type TransactionHandler struct {
	service TransactionService
	logger  *slog.Logger
}

func NewTransactionHandler(service TransactionService, logger *slog.Logger) *TransactionHandler {
	return &TransactionHandler{service: service, logger: logger}
}

func (h *TransactionHandler) Create(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var req createTransactionRequest
	if err := json.Unmarshal(body, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	idempotency, err := newIdempotencyRequest(r, "POST /transactions", body)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	response, err := h.service.Create(
		r.Context(),
		req.TerminalSerial,
		req.Amount,
		req.Currency,
		req.PaymentMethod,
		req.Card,
		idempotency,
	)
	if err != nil {
		h.handleError(w, err)
		return
	}

	respondRawJSON(w, response.StatusCode, response.Body)
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

	responseEvents := make([]transactionEventResponse, len(events))
	for i, m := range events {
		responseEvents[i] = newTransactionEventResponse(m)
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"transaction": models.NewTransactionResponse(*transaction),
		"events":      responseEvents,
	})
}

func (h *TransactionHandler) Capture(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")

	id, err := uuid.Parse(rawID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid transaction id")
		return
	}

	idempotency, err := newIdempotencyRequest(r, "POST /transactions/{id}/capture", nil)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	response, err := h.service.Capture(r.Context(), id, idempotency)
	if err != nil {
		h.handleError(w, err)
		return
	}

	respondRawJSON(w, response.StatusCode, response.Body)
}

func (h *TransactionHandler) Refund(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")

	id, err := uuid.Parse(rawID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid transaction id")
		return
	}

	idempotency, err := newIdempotencyRequest(r, "POST /transactions/{id}/refund", nil)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	response, err := h.service.Refund(r.Context(), id, idempotency)
	if err != nil {
		h.handleError(w, err)
		return
	}

	respondRawJSON(w, response.StatusCode, response.Body)
}

func (h *TransactionHandler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrTerminalNotFound):
		respondError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrTransactionNotFound):
		respondError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrTerminalBlocked):
		respondError(w, http.StatusConflict, err.Error())
	case errors.Is(err, service.ErrIdempotencyKeyReused):
		respondError(w, http.StatusConflict, err.Error())
	case errors.Is(err, payment.ErrInvalidTransition):
		respondError(w, http.StatusConflict, err.Error())
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

type transactionEventResponse struct {
	ID        uuid.UUID       `json:"id"`
	Event     string          `json:"event"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
}

func newIdempotencyRequest(
	r *http.Request,
	endpoint string,
	body []byte,
) (service.IdempotencyRequest, error) {
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" {
		return service.IdempotencyRequest{}, errors.New("Idempotency-Key is required")
	}
	if len(key) > 255 {
		return service.IdempotencyRequest{}, errors.New("Idempotency-Key must contain at most 255 characters")
	}

	hash := sha256.New()
	_, _ = hash.Write([]byte(r.Method))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write([]byte(r.URL.Path))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write(body)

	return service.IdempotencyRequest{
		Key:         key,
		Endpoint:    endpoint,
		RequestHash: hex.EncodeToString(hash.Sum(nil)),
	}, nil
}

func newTransactionEventResponse(m models.TransactionEvent) transactionEventResponse {
	return transactionEventResponse{
		ID:        m.ID,
		Event:     m.Event,
		Payload:   m.Payload,
		CreatedAt: m.CreatedAt,
	}
}
