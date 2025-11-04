package workflow

import (
	"github.com/cypherlabdev/order-validator-service/internal/models"
)

// Activity input/output types for PlaceOrderWorkflow

// ValidateOrderInput is input for ValidateOrderActivity
type ValidateOrderInput struct {
	OrderRequest *models.PlaceOrderRequest
}

// ValidationResult is output for ValidateOrderActivity
type ValidationResult struct {
	Valid  bool
	Reason string
}

// ReserveFundsInput is input for ReserveFundsActivity
type ReserveFundsInput struct {
	UserID         string
	Amount         string
	Currency       string
	SagaID         string
	IdempotencyKey string
}

// ReserveFundsResult is output for ReserveFundsActivity
type ReserveFundsResult struct {
	ReservationID string
	Status        string
}

// CommitReservationInput is input for CommitReservationActivity
type CommitReservationInput struct {
	ReservationID  string
	SagaID         string
	IdempotencyKey string
}

// CommitReservationResult is output for CommitReservationActivity
type CommitReservationResult struct {
	Status string
}

// CancelReservationInput is input for CancelReservationActivity
type CancelReservationInput struct {
	ReservationID  string
	SagaID         string
	IdempotencyKey string
}

// CancelReservationResult is output for CancelReservationActivity
type CancelReservationResult struct {
	Status string
}

// PlaceOrderInBookInput is input for PlaceOrderInBookActivity
type PlaceOrderInBookInput struct {
	UserID         string
	EventID        string
	MarketID       string
	SelectionID    string
	Side           string
	Odds           string
	Stake          string
	Currency       string
	ReservationID  string
	SagaID         string
	IdempotencyKey string
}

// PlaceOrderInBookResult is output for PlaceOrderInBookActivity
type PlaceOrderInBookResult struct {
	OrderID string
	MatchID string
	Status  string
}

// CancelOrderInput is input for CancelOrderActivity
type CancelOrderInput struct {
	OrderID        string
	SagaID         string
	IdempotencyKey string
}

// CancelOrderResult is output for CancelOrderActivity
type CancelOrderResult struct {
	Status string
}
