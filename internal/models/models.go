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

type TransactionEvent struct {
	ID            uuid.UUID
	TransactionID uuid.UUID
	Event         string
	Payload       json.RawMessage
	CreatedAt     time.Time
}

type IdempotencyKey struct {
	Key       string
	Endpoint  string
	Response  json.RawMessage
	CreatedAt time.Time
}
