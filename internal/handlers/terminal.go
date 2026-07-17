package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/cauanvital/payment-gateway-simulator/internal/models"
	"github.com/cauanvital/payment-gateway-simulator/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type TerminalService interface {
	Create(ctx context.Context, merchantID uuid.UUID, serial string) (*models.Terminal, error)
	Get(ctx context.Context, id uuid.UUID) (*models.Terminal, error)
	List(ctx context.Context, merchantID uuid.UUID) ([]models.Terminal, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status models.TerminalStatus) (*models.Terminal, error)
}

type TerminalHandler struct {
	service TerminalService
	logger  *slog.Logger
}

func NewTerminalHandler(service TerminalService, logger *slog.Logger) *TerminalHandler {
	return &TerminalHandler{service: service, logger: logger}
}

func (h *TerminalHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createTerminalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	terminal, err := h.service.Create(r.Context(), req.MerchantID, req.Serial)
	if err != nil {
		h.handleError(w, err)
		return
	}

	respondJSON(w, http.StatusCreated, newTerminalResponse(*terminal))
}

func (h *TerminalHandler) Get(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")

	id, err := uuid.Parse(rawID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid terminal id")
		return
	}

	terminal, err := h.service.Get(r.Context(), id)
	if err != nil {
		h.handleError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, newTerminalResponse(*terminal))
}

func (h *TerminalHandler) List(w http.ResponseWriter, r *http.Request) {
	rawMerchantID := chi.URLParam(r, "merchant_id")

	merchantID, err := uuid.Parse(rawMerchantID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid merchant id")
		return
	}

	terminals, err := h.service.List(r.Context(), merchantID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	terminalsResponse := make([]terminalResponse, len(terminals))
	for i, m := range terminals {
		terminalsResponse[i] = newTerminalResponse(m)
	}

	respondJSON(w, http.StatusOK, terminalsResponse)
}

func (h *TerminalHandler) Block(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")

	id, err := uuid.Parse(rawID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid terminal id")
		return
	}

	terminal, err := h.service.UpdateStatus(r.Context(), id, models.TerminalBlocked)
	if err != nil {
		h.handleError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, newTerminalResponse(*terminal))
}

func (h *TerminalHandler) Activate(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")

	id, err := uuid.Parse(rawID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid terminal id")
		return
	}

	terminal, err := h.service.UpdateStatus(r.Context(), id, models.TerminalActive)
	if err != nil {
		h.handleError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, newTerminalResponse(*terminal))
}

func (h *TerminalHandler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrTerminalNotFound):
		respondError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrTerminalSerialBlank):
		respondError(w, http.StatusBadRequest, err.Error())
	default:
		h.logger.Error("unexpected error", "error", err)
		respondError(w, http.StatusInternalServerError, "internal server error")
	}
}

type createTerminalRequest struct {
	Serial     string    `json:"serial"`
	MerchantID uuid.UUID `json:"merchant_id"`
}

type terminalResponse struct {
	ID         uuid.UUID             `json:"id"`
	Serial     string                `json:"serial"`
	MerchantID uuid.UUID             `json:"merchant_id"`
	Status     models.TerminalStatus `json:"status"`
	CreatedAt  time.Time             `json:"created_at"`
}

func newTerminalResponse(m models.Terminal) terminalResponse {
	return terminalResponse{
		ID:         m.ID,
		Serial:     m.Serial,
		MerchantID: m.MerchantID,
		Status:     m.Status,
		CreatedAt:  m.CreatedAt,
	}
}

type updateTerminalStatusRequest struct {
	Status models.TerminalStatus `json:"terminal_status"`
}
