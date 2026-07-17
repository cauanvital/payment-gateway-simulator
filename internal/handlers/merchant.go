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

type MerchantService interface {
	Create(ctx context.Context, name string) (*models.Merchant, error)
	Get(ctx context.Context, id uuid.UUID) (*models.Merchant, error)
	List(ctx context.Context) ([]models.Merchant, error)
}

type MerchantHandler struct {
	service MerchantService
	logger  *slog.Logger
}

func NewMerchantHandler(service MerchantService, logger *slog.Logger) *MerchantHandler {
	return &MerchantHandler{service: service, logger: logger}
}

func (h *MerchantHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createMerchantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	merchant, err := h.service.Create(r.Context(), req.Name)
	if err != nil {
		h.handleError(w, err)
		return
	}

	respondJSON(w, http.StatusCreated, newMerchantResponse(*merchant))
}

func (h *MerchantHandler) Get(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")

	id, err := uuid.Parse(rawID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid merchant id")
		return
	}

	merchant, err := h.service.Get(r.Context(), id)
	if err != nil {
		h.handleError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, newMerchantResponse(*merchant))
}

func (h *MerchantHandler) List(w http.ResponseWriter, r *http.Request) {
	merchants, err := h.service.List(r.Context())
	if err != nil {
		h.handleError(w, err)
		return
	}

	merchantsResponse := make([]merchantResponse, len(merchants))
	for i, m := range merchants {
		merchantsResponse[i] = newMerchantResponse(m)
	}

	respondJSON(w, http.StatusOK, merchantsResponse)
}

func (h *MerchantHandler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrMerchantNotFound):
		respondError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrMerchantNameBlank):
		respondError(w, http.StatusBadRequest, err.Error())
	default:
		h.logger.Error("unexpected error", "error", err)
		respondError(w, http.StatusInternalServerError, "internal server error")
	}
}

type createMerchantRequest struct {
	Name string `json:"name"`
}

type merchantResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

func newMerchantResponse(m models.Merchant) merchantResponse {
	return merchantResponse{ID: m.ID, Name: m.Name, CreatedAt: m.CreatedAt}
}
