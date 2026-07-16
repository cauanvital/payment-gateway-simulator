package service

import "github.com/cauanvital/payment-gateway-simulator/internal/repository"

type MerchantService struct {
	r *repository.MerchantRepository
}

func NewMerchantService(r *repository.MerchantRepository) *MerchantService {
	return &MerchantService{r: r}
}
