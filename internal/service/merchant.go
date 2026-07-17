package service

import (
	"context"
	"errors"
	"strings"

	"github.com/cauanvital/payment-gateway-simulator/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type MerchantRepository interface {
	Create(ctx context.Context, name string) (*models.Merchant, error)
	Get(ctx context.Context, id uuid.UUID) (*models.Merchant, error)
	List(ctx context.Context) ([]models.Merchant, error)
}

var (
	ErrMerchantNotFound  = errors.New("merchant not found")
	ErrMerchantNameBlank = errors.New("merchant name cannot be blank")
)

type MerchantService struct {
	repo MerchantRepository
}

func NewMerchantService(repo MerchantRepository) *MerchantService {
	return &MerchantService{repo: repo}
}

func (s *MerchantService) Create(ctx context.Context, name string) (*models.Merchant, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrMerchantNameBlank
	}
	return s.repo.Create(ctx, name)
}

func (s *MerchantService) Get(ctx context.Context, id uuid.UUID) (*models.Merchant, error) {
	merchant, err := s.repo.Get(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMerchantNotFound
		}
		return nil, err
	}
	return merchant, nil
}

func (s *MerchantService) List(ctx context.Context) ([]models.Merchant, error) {
	merchants, err := s.repo.List(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMerchantNotFound
		}
		return nil, err
	}
	return merchants, nil
}
