package payment

import (
	"errors"

	"github.com/cauanvital/payment-gateway-simulator/internal/models"
)

var allowedTransitions = map[models.TransactionStatus][]models.TransactionStatus{
	models.TransactionCreated:    {models.TransactionAuthorized, models.TransactionDeclined},
	models.TransactionAuthorized: {models.TransactionCaptured},
	models.TransactionCaptured:   {models.TransactionRefunded},
	models.TransactionRefunded:   {},
	models.TransactionDeclined:   {},
}

var ErrInvalidTransition = errors.New("invalid state transition")

type PaymentStateMachine struct{}

func (PaymentStateMachine) CanTransition(from, to models.TransactionStatus) error {
	for _, allowed := range allowedTransitions[from] {
		if allowed == to {
			return nil
		}
	}
	return ErrInvalidTransition
}

func (m PaymentStateMachine) Authorize(tx *models.Transaction) error {
	if err := m.CanTransition(tx.Status, models.TransactionAuthorized); err != nil {
		return err
	}
	tx.Status = models.TransactionAuthorized
	return nil
}

func (m PaymentStateMachine) Capture(tx *models.Transaction) error {
	if err := m.CanTransition(tx.Status, models.TransactionCaptured); err != nil {
		return err
	}
	tx.Status = models.TransactionCaptured
	return nil
}

func (m PaymentStateMachine) Refund(tx *models.Transaction) error {
	if err := m.CanTransition(tx.Status, models.TransactionRefunded); err != nil {
		return err
	}
	tx.Status = models.TransactionRefunded
	return nil
}

func (m PaymentStateMachine) Decline(tx *models.Transaction) error {
	if err := m.CanTransition(tx.Status, models.TransactionDeclined); err != nil {
		return err
	}
	tx.Status = models.TransactionDeclined
	return nil
}
