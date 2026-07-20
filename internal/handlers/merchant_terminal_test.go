package handlers

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cauanvital/payment-gateway-simulator/internal/models"
	"github.com/cauanvital/payment-gateway-simulator/internal/service"
	"github.com/google/uuid"
)

type merchantServiceFake struct {
	name string
	err  error
}

func (f *merchantServiceFake) Create(_ context.Context, name string) (*models.Merchant, error) {
	f.name = name
	if f.err != nil {
		return nil, f.err
	}
	return &models.Merchant{ID: uuid.New(), Name: name, CreatedAt: time.Now().UTC()}, nil
}
func (f *merchantServiceFake) Get(context.Context, uuid.UUID) (*models.Merchant, error) {
	return nil, f.err
}
func (f *merchantServiceFake) List(context.Context) ([]models.Merchant, error) { return nil, f.err }

func TestMerchantHandlerCreateAndValidation(t *testing.T) {
	t.Parallel()

	fake := &merchantServiceFake{}
	handler := NewMerchantHandler(fake, slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := httptest.NewRequest(http.MethodPost, "/merchants/", strings.NewReader(`{"name":"Loja Exemplo"}`))
	recorder := httptest.NewRecorder()

	handler.Create(recorder, request)

	if recorder.Code != http.StatusCreated || fake.name != "Loja Exemplo" {
		t.Fatalf("response = %d %s; service name = %q", recorder.Code, recorder.Body.String(), fake.name)
	}

	badRequest := httptest.NewRequest(http.MethodPost, "/merchants/", strings.NewReader(`{`))
	badRecorder := httptest.NewRecorder()
	handler.Create(badRecorder, badRequest)
	assertJSONError(t, badRecorder, http.StatusBadRequest, "invalid request body")
}

type terminalServiceFake struct {
	updatedStatus models.TerminalStatus
}

func (f *terminalServiceFake) Create(context.Context, uuid.UUID, string) (*models.Terminal, error) {
	return nil, nil
}
func (f *terminalServiceFake) Get(context.Context, uuid.UUID) (*models.Terminal, error) {
	return nil, nil
}
func (f *terminalServiceFake) List(context.Context, uuid.UUID) ([]models.Terminal, error) {
	return nil, nil
}
func (f *terminalServiceFake) UpdateStatus(_ context.Context, id uuid.UUID, status models.TerminalStatus) (*models.Terminal, error) {
	f.updatedStatus = status
	return &models.Terminal{ID: id, Status: status}, nil
}

func TestTerminalHandlerBlock(t *testing.T) {
	t.Parallel()

	fake := &terminalServiceFake{}
	handler := NewTerminalHandler(fake, slog.New(slog.NewTextHandler(io.Discard, nil)))
	id := uuid.New()
	request := requestWithURLParam(http.MethodPost, "/terminals/"+id.String()+"/block", "id", id.String())
	recorder := httptest.NewRecorder()

	handler.Block(recorder, request)

	if recorder.Code != http.StatusOK || fake.updatedStatus != models.TerminalBlocked {
		t.Fatalf("response = %d %s; status = %s", recorder.Code, recorder.Body.String(), fake.updatedStatus)
	}

	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
}

func TestMerchantHandlerMapsKnownErrors(t *testing.T) {
	t.Parallel()

	handler := NewMerchantHandler(&merchantServiceFake{err: service.ErrMerchantNameBlank}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := httptest.NewRequest(http.MethodPost, "/merchants/", strings.NewReader(`{"name":""}`))
	recorder := httptest.NewRecorder()
	handler.Create(recorder, request)
	assertJSONError(t, recorder, http.StatusBadRequest, service.ErrMerchantNameBlank.Error())
}
