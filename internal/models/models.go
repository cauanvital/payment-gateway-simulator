package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type TerminalStatus string

const (
	TerminalActive  TerminalStatus = "ACTIVE"
	TerminalBlocked TerminalStatus = "BLOCKED"
)

type TransactionStatus string

const (
	TransactionCreated    TransactionStatus = "CREATED"
	TransactionAuthorized TransactionStatus = "AUTHORIZED"
	TransactionCaptured   TransactionStatus = "CAPTURED"
	TransactionRefunded   TransactionStatus = "REFUNDED"
	TransactionDeclined   TransactionStatus = "DECLINED"
)

type PaymentMethod string

const (
	MethodCreditCard PaymentMethod = "CREDIT_CARD"
	MethodDebitCard  PaymentMethod = "DEBIT_CARD"
	MethodPix        PaymentMethod = "PIX"
)

type Merchant struct {
	ID        uuid.UUID
	Name      string
	CreatedAt time.Time
}

type Terminal struct {
	ID         uuid.UUID
	Serial     string
	MerchantID uuid.UUID
	Status     TerminalStatus
	CreatedAt  time.Time
}

type Transaction struct {
	ID                uuid.UUID
	MerchantID        uuid.UUID
	TerminalID        uuid.UUID
	Amount            int64
	Currency          string
	PaymentMethod     PaymentMethod
	Status            TransactionStatus
	AuthorizationCode *string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// TransactionResponse é o contrato público de uma transação na API.
// Ele também é persistido para que uma repetição idempotente receba
// exatamente a mesma resposta da primeira requisição.
type TransactionResponse struct {
	ID            uuid.UUID         `json:"id"`
	MerchantID    uuid.UUID         `json:"merchant_id"`
	TerminalID    uuid.UUID         `json:"terminal_id"`
	Amount        int64             `json:"amount"`
	Currency      string            `json:"currency"`
	PaymentMethod PaymentMethod     `json:"payment_method"`
	Status        TransactionStatus `json:"status"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

func NewTransactionResponse(transaction Transaction) TransactionResponse {
	return TransactionResponse{
		ID:            transaction.ID,
		MerchantID:    transaction.MerchantID,
		TerminalID:    transaction.TerminalID,
		Amount:        transaction.Amount,
		Currency:      transaction.Currency,
		PaymentMethod: transaction.PaymentMethod,
		Status:        transaction.Status,
		CreatedAt:     transaction.CreatedAt,
		UpdatedAt:     transaction.UpdatedAt,
	}
}

type TransactionEvent struct {
	ID            uuid.UUID
	TransactionID uuid.UUID
	Event         string
	Payload       json.RawMessage
	CreatedAt     time.Time
}

type IdempotencyKey struct {
	Key         string
	Endpoint    string
	RequestHash string
	StatusCode  int16
	Response    json.RawMessage
	CreatedAt   time.Time
}
