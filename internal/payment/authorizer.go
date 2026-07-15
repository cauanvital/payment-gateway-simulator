package payment

import (
	"strings"

	"github.com/cauanvital/payment-gateway-simulator/internal/models"
)

const fraudThreshold = 10000

type Decision struct {
	Approved bool
	Reason   string
	AuthCode string
}

type Authorizer struct{}

func (Authorizer) Authorize(tx *models.Transaction, card string) Decision {
	// Autoriza cartão de teste com final "1111" (esse fura tudo)
	if strings.HasSuffix(card, "1111") {
		return Decision{Approved: true, AuthCode: getAuthCode()}
	}

	// Desaprova cartão de teste com final "0000" (esse não fura nada)
	if strings.HasSuffix(card, "0000") {
		return Decision{Approved: false, Reason: "test card declined"}
	}

	// Bloqueia fraude baseado no limite definido
	if tx.Amount > fraudThreshold {
		return Decision{Approved: false, Reason: "fraud: amount above the limit"}
	}

	return Decision{Approved: true, AuthCode: getAuthCode()}
}

func getAuthCode() string {
	return ""
}
