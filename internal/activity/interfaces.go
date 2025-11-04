package activity

import (
	"context"

	"github.com/cypherlabdev/order-validator-service/internal/workflow"
)

// ValidationActivityInterface defines the interface for validation activities
type ValidationActivityInterface interface {
	ValidateOrder(ctx context.Context, input workflow.ValidateOrderInput) (*workflow.ValidationResult, error)
}

// WalletActivityInterface defines the interface for wallet activities
type WalletActivityInterface interface {
	ReserveFunds(ctx context.Context, input workflow.ReserveFundsInput) (*workflow.ReserveFundsResult, error)
	CommitReservation(ctx context.Context, input workflow.CommitReservationInput) (*workflow.CommitReservationResult, error)
	CancelReservation(ctx context.Context, input workflow.CancelReservationInput) (*workflow.CancelReservationResult, error)
}

// OrderBookActivityInterface defines the interface for order book activities
type OrderBookActivityInterface interface {
	PlaceOrderInBook(ctx context.Context, input workflow.PlaceOrderInBookInput) (*workflow.PlaceOrderInBookResult, error)
	CancelOrder(ctx context.Context, input workflow.CancelOrderInput) (*workflow.CancelOrderResult, error)
}
