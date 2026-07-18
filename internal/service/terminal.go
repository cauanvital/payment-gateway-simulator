package service

import (
	"context"
	"errors"
	"strings"

	"github.com/cauanvital/payment-gateway-simulator/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type TerminalRepository interface {
	Create(ctx context.Context, merchantID uuid.UUID, serial string) (*models.Terminal, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.Terminal, error)
	GetBySerial(ctx context.Context, serial string) (*models.Terminal, error)
	List(ctx context.Context, merchantID uuid.UUID) ([]models.Terminal, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status models.TerminalStatus) (*models.Terminal, error)
}

var (
	ErrTerminalNotFound    = errors.New("terminal not found")
	ErrTerminalSerialBlank = errors.New("terminal serial cannot be blank")
)

type TerminalService struct {
	repo TerminalRepository
}

func NewTerminalService(repo TerminalRepository) *TerminalService {
	return &TerminalService{repo: repo}
}

func (s *TerminalService) Create(ctx context.Context, merchantID uuid.UUID, serial string) (*models.Terminal, error) {
	serial = strings.TrimSpace(serial)
	if serial == "" {
		return nil, ErrTerminalSerialBlank
	}
	return s.repo.Create(ctx, merchantID, serial)
}

func (s *TerminalService) Get(ctx context.Context, id uuid.UUID) (*models.Terminal, error) {
	terminal, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTerminalNotFound
		}
		return nil, err
	}
	return terminal, nil
}

func (s *TerminalService) List(ctx context.Context, merchantID uuid.UUID) ([]models.Terminal, error) {
	terminals, err := s.repo.List(ctx, merchantID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTerminalNotFound
		}
		return nil, err
	}
	return terminals, nil
}

func (s *TerminalService) UpdateStatus(ctx context.Context, id uuid.UUID, status models.TerminalStatus) (*models.Terminal, error) {
	terminal, err := s.repo.UpdateStatus(ctx, id, status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTerminalNotFound
		}
		return nil, err
	}
	return terminal, nil
}
