package service

import (
	"context"
	"errors"
	"testing"

	"github.com/cauanvital/payment-gateway-simulator/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type merchantRepoFake struct {
	createName string
	getErr     error
	listErr    error
}

func (f *merchantRepoFake) Create(_ context.Context, name string) (*models.Merchant, error) {
	f.createName = name
	return &models.Merchant{Name: name}, nil
}
func (f *merchantRepoFake) Get(context.Context, uuid.UUID) (*models.Merchant, error) {
	return nil, f.getErr
}
func (f *merchantRepoFake) List(context.Context) ([]models.Merchant, error) { return nil, f.listErr }

func TestMerchantServiceCreateValidatesAndTrimsName(t *testing.T) {
	t.Parallel()

	repo := &merchantRepoFake{}
	service := NewMerchantService(repo)

	merchant, err := service.Create(context.Background(), "  Loja Exemplo  ")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if merchant.Name != "Loja Exemplo" || repo.createName != "Loja Exemplo" {
		t.Fatalf("merchant = %#v, repository name = %q", merchant, repo.createName)
	}

	_, err = service.Create(context.Background(), " \t ")
	if !errors.Is(err, ErrMerchantNameBlank) {
		t.Fatalf("blank Create() error = %v, want %v", err, ErrMerchantNameBlank)
	}
}

func TestMerchantServiceMapsNotFound(t *testing.T) {
	t.Parallel()

	service := NewMerchantService(&merchantRepoFake{getErr: pgx.ErrNoRows, listErr: pgx.ErrNoRows})
	if _, err := service.Get(context.Background(), uuid.New()); !errors.Is(err, ErrMerchantNotFound) {
		t.Fatalf("Get() error = %v, want %v", err, ErrMerchantNotFound)
	}
	if _, err := service.List(context.Background()); !errors.Is(err, ErrMerchantNotFound) {
		t.Fatalf("List() error = %v, want %v", err, ErrMerchantNotFound)
	}
}

type terminalRepoFake struct {
	createSerial string
	getErr       error
	listErr      error
	updateErr    error
	updatedTo    models.TerminalStatus
}

func (f *terminalRepoFake) Create(_ context.Context, merchantID uuid.UUID, serial string) (*models.Terminal, error) {
	f.createSerial = serial
	return &models.Terminal{MerchantID: merchantID, Serial: serial}, nil
}
func (f *terminalRepoFake) GetByID(context.Context, uuid.UUID) (*models.Terminal, error) {
	return nil, f.getErr
}
func (f *terminalRepoFake) GetBySerial(context.Context, string) (*models.Terminal, error) {
	return nil, f.getErr
}
func (f *terminalRepoFake) List(context.Context, uuid.UUID) ([]models.Terminal, error) {
	return nil, f.listErr
}
func (f *terminalRepoFake) UpdateStatus(_ context.Context, id uuid.UUID, status models.TerminalStatus) (*models.Terminal, error) {
	f.updatedTo = status
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	return &models.Terminal{ID: id, Status: status}, nil
}

func TestTerminalServiceCreateAndUpdate(t *testing.T) {
	t.Parallel()

	repo := &terminalRepoFake{}
	service := NewTerminalService(repo)
	merchantID := uuid.New()

	terminal, err := service.Create(context.Background(), merchantID, "  POS-001 ")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if terminal.Serial != "POS-001" || repo.createSerial != "POS-001" {
		t.Fatalf("serials = %q and %q, want POS-001", terminal.Serial, repo.createSerial)
	}
	if _, err := service.Create(context.Background(), merchantID, " "); !errors.Is(err, ErrTerminalSerialBlank) {
		t.Fatalf("blank Create() error = %v, want %v", err, ErrTerminalSerialBlank)
	}

	if _, err := service.UpdateStatus(context.Background(), uuid.New(), models.TerminalBlocked); err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}
	if repo.updatedTo != models.TerminalBlocked {
		t.Fatalf("updated status = %s, want BLOCKED", repo.updatedTo)
	}
}

func TestTerminalServiceMapsNotFound(t *testing.T) {
	t.Parallel()

	service := NewTerminalService(&terminalRepoFake{getErr: pgx.ErrNoRows, listErr: pgx.ErrNoRows, updateErr: pgx.ErrNoRows})
	if _, err := service.Get(context.Background(), uuid.New()); !errors.Is(err, ErrTerminalNotFound) {
		t.Fatalf("Get() error = %v", err)
	}
	if _, err := service.List(context.Background(), uuid.New()); !errors.Is(err, ErrTerminalNotFound) {
		t.Fatalf("List() error = %v", err)
	}
	if _, err := service.UpdateStatus(context.Background(), uuid.New(), models.TerminalActive); !errors.Is(err, ErrTerminalNotFound) {
		t.Fatalf("UpdateStatus() error = %v", err)
	}
}
