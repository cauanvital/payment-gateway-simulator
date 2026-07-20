package payment

import (
	"errors"
	"strings"
	"testing"

	"github.com/cauanvital/payment-gateway-simulator/internal/models"
)

func TestPaymentStateMachineTransitions(t *testing.T) {
	t.Parallel()

	machine := PaymentStateMachine{}
	tests := []struct {
		name string
		from models.TransactionStatus
		to   models.TransactionStatus
		want error
	}{
		{"authorize created", models.TransactionCreated, models.TransactionAuthorized, nil},
		{"decline created", models.TransactionCreated, models.TransactionDeclined, nil},
		{"capture authorized", models.TransactionAuthorized, models.TransactionCaptured, nil},
		{"refund captured", models.TransactionCaptured, models.TransactionRefunded, nil},
		{"authorize declined", models.TransactionDeclined, models.TransactionAuthorized, ErrInvalidTransition},
		{"capture refunded", models.TransactionRefunded, models.TransactionCaptured, ErrInvalidTransition},
		{"refund authorized", models.TransactionAuthorized, models.TransactionRefunded, ErrInvalidTransition},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := machine.CanTransition(tt.from, tt.to)
			if !errors.Is(err, tt.want) {
				t.Fatalf("CanTransition(%s, %s) error = %v, want %v", tt.from, tt.to, err, tt.want)
			}
		})
	}
}

func TestPaymentStateMachineUpdatesTransaction(t *testing.T) {
	t.Parallel()

	machine := PaymentStateMachine{}
	tx := &models.Transaction{Status: models.TransactionCreated}

	if err := machine.Authorize(tx); err != nil {
		t.Fatalf("Authorize() error = %v", err)
	}
	if tx.Status != models.TransactionAuthorized {
		t.Fatalf("status = %s, want AUTHORIZED", tx.Status)
	}
	if err := machine.Capture(tx); err != nil {
		t.Fatalf("Capture() error = %v", err)
	}
	if err := machine.Refund(tx); err != nil {
		t.Fatalf("Refund() error = %v", err)
	}
	if tx.Status != models.TransactionRefunded {
		t.Fatalf("status = %s, want REFUNDED", tx.Status)
	}
}

func TestAuthorizerRules(t *testing.T) {
	t.Parallel()

	authorizer := Authorizer{}
	tests := []struct {
		name         string
		amount       int64
		card         string
		wantApproved bool
		wantReason   string
	}{
		{"approved test card bypasses fraud threshold", 20000, "4111111111111111", true, ""},
		{"declined test card", 100, "4000000000000000", false, "test card declined"},
		{"fraud threshold", 10001, "4000000000001234", false, "fraud: amount above the limit"},
		{"regular approval", 10000, "4000000000001234", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := authorizer.Authorize(&models.Transaction{Amount: tt.amount}, tt.card)
			if decision.Approved != tt.wantApproved || decision.Reason != tt.wantReason {
				t.Fatalf("decision = %#v, want approved=%t reason=%q", decision, tt.wantApproved, tt.wantReason)
			}
			if decision.Approved {
				if len(decision.AuthCode) != 6 || strings.Trim(decision.AuthCode, "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789") != "" {
					t.Fatalf("invalid authorization code %q", decision.AuthCode)
				}
			}
		})
	}
}
