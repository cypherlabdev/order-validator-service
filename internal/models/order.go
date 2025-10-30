package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// OrderStatus represents the lifecycle state of an order
type OrderStatus string

const (
	OrderStatusPending    OrderStatus = "PENDING"     // Order received, validation starting
	OrderStatusValidating OrderStatus = "VALIDATING"  // Running validations
	OrderStatusReserving  OrderStatus = "RESERVING"   // Reserving funds in wallet
	OrderStatusMatching   OrderStatus = "MATCHING"    // Matching order in order-book
	OrderStatusSettled    OrderStatus = "SETTLED"     // Order settled successfully
	OrderStatusFailed     OrderStatus = "FAILED"      // Order failed validation or processing
	OrderStatusCancelled  OrderStatus = "CANCELLED"   // Order cancelled by user
)

// OrderSide represents whether order is backing or laying
type OrderSide string

const (
	OrderSideBack OrderSide = "BACK" // Betting for outcome
	OrderSideLay  OrderSide = "LAY"  // Betting against outcome
)

// Order represents a bet order in the system
type Order struct {
	ID              uuid.UUID       `json:"id"`
	UserID          uuid.UUID       `json:"user_id"`
	EventID         string          `json:"event_id"`          // From data-normalizer
	MarketID        string          `json:"market_id"`         // From data-normalizer
	SelectionID     string          `json:"selection_id"`      // From data-normalizer
	Side            OrderSide       `json:"side"`              // BACK or LAY
	Odds            decimal.Decimal `json:"odds"`              // Decimal odds (e.g., 2.5)
	Stake           decimal.Decimal `json:"stake"`             // Amount wagered
	Currency        string          `json:"currency"`          // USD, EUR, etc.
	Status          OrderStatus     `json:"status"`
	ReservationID   *uuid.UUID      `json:"reservation_id,omitempty"`   // Wallet reservation ID
	MatchID         *uuid.UUID      `json:"match_id,omitempty"`         // Order-book match ID
	SagaID          string          `json:"saga_id"`                    // Temporal workflow ID
	IdempotencyKey  string          `json:"idempotency_key"`            // For idempotent operations
	FailureReason   string          `json:"failure_reason,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	SettledAt       *time.Time      `json:"settled_at,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// PlaceOrderRequest is the request to place a new order
type PlaceOrderRequest struct {
	UserID         uuid.UUID       `json:"user_id" validate:"required"`
	EventID        string          `json:"event_id" validate:"required"`
	MarketID       string          `json:"market_id" validate:"required"`
	SelectionID    string          `json:"selection_id" validate:"required"`
	Side           OrderSide       `json:"side" validate:"required,oneof=BACK LAY"`
	Odds           decimal.Decimal `json:"odds" validate:"required,gt=0"`
	Stake          decimal.Decimal `json:"stake" validate:"required,gt=0"`
	Currency       string          `json:"currency" validate:"required,len=3"`
	IdempotencyKey string          `json:"idempotency_key" validate:"required"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// Validation errors
const (
	ErrInvalidOdds      = "invalid odds: must be greater than 1.0"
	ErrInvalidStake     = "invalid stake: must be positive"
	ErrInvalidCurrency  = "invalid currency: must be 3-letter code"
	ErrInvalidSide      = "invalid side: must be BACK or LAY"
	ErrMarketClosed     = "market is closed for betting"
	ErrOddsOutOfRange   = "odds out of acceptable range"
	ErrStakeTooLow      = "stake below minimum"
	ErrStakeTooHigh     = "stake exceeds maximum"
)

// Validate validates the order request
func (r *PlaceOrderRequest) Validate() error {
	// Odds must be greater than 1.0 (even money or better)
	if r.Odds.LessThanOrEqual(decimal.NewFromInt(1)) {
		return &ValidationError{Field: "odds", Message: ErrInvalidOdds}
	}

	// Stake must be positive
	if r.Stake.LessThanOrEqual(decimal.Zero) {
		return &ValidationError{Field: "stake", Message: ErrInvalidStake}
	}

	// Currency must be 3 characters
	if len(r.Currency) != 3 {
		return &ValidationError{Field: "currency", Message: ErrInvalidCurrency}
	}

	// Side must be BACK or LAY
	if r.Side != OrderSideBack && r.Side != OrderSideLay {
		return &ValidationError{Field: "side", Message: ErrInvalidSide}
	}

	return nil
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return e.Message
}

// CalculatePotentialWinnings calculates potential winnings for the order
func (o *Order) CalculatePotentialWinnings() decimal.Decimal {
	if o.Side == OrderSideBack {
		// Back bet: stake * (odds - 1)
		return o.Stake.Mul(o.Odds.Sub(decimal.NewFromInt(1)))
	}
	// Lay bet: stake * odds (liability)
	return o.Stake.Mul(o.Odds)
}

// CalculateRisk calculates the amount at risk
func (o *Order) CalculateRisk() decimal.Decimal {
	if o.Side == OrderSideBack {
		// Back bet: risk is the stake
		return o.Stake
	}
	// Lay bet: risk is the liability (stake * odds)
	return o.Stake.Mul(o.Odds)
}
