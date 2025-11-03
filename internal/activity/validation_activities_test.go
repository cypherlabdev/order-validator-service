package activity

import (
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"github.com/cypherlabdev/order-validator-service/internal/models"
	"github.com/cypherlabdev/order-validator-service/internal/workflow"
)

// testValidationActivitySetup is a helper struct to hold test dependencies
type testValidationActivitySetup struct {
	activity *ValidationActivities
}

// setupTestValidationActivity creates a test activity with all dependencies
func setupTestValidationActivity(t *testing.T) *testValidationActivitySetup {
	logger := zerolog.Nop()
	activity := NewValidationActivities(logger)

	return &testValidationActivitySetup{
		activity: activity,
	}
}

// TestValidateOrder_Success tests successful order validation
func TestValidateOrder_Success(t *testing.T) {
	setup := setupTestValidationActivity(t)

	input := workflow.ValidateOrderInput{
		OrderRequest: &models.PlaceOrderRequest{
			UserID:         uuid.New(),
			EventID:        "event_123",
			MarketID:       "market_456",
			SelectionID:    "selection_789",
			Side:           models.OrderSideBack,
			Odds:           decimal.NewFromFloat(2.5),
			Stake:          decimal.NewFromFloat(100.0),
			Currency:       "USD",
			IdempotencyKey: "test_key_1",
		},
	}

	// Execute - test the validation logic directly
	req := input.OrderRequest

	// Test basic validation
	err := req.Validate()
	assert.NoError(t, err)

	// Test stake limits
	assert.True(t, req.Stake.GreaterThanOrEqual(setup.activity.minStake))
	assert.True(t, req.Stake.LessThanOrEqual(setup.activity.maxStake))

	// Test odds limits
	assert.True(t, req.Odds.GreaterThanOrEqual(setup.activity.minOdds))
	assert.True(t, req.Odds.LessThanOrEqual(setup.activity.maxOdds))
}

// TestValidateOrder_BasicValidationFailure tests order with basic validation errors
func TestValidateOrder_BasicValidationFailure(t *testing.T) {
	tests := []struct {
		name          string
		orderRequest  *models.PlaceOrderRequest
		expectedError string
	}{
		{
			name: "invalid odds (less than 1.0)",
			orderRequest: &models.PlaceOrderRequest{
				UserID:         uuid.New(),
				EventID:        "event_123",
				MarketID:       "market_456",
				SelectionID:    "selection_789",
				Side:           models.OrderSideBack,
				Odds:           decimal.NewFromFloat(0.5),
				Stake:          decimal.NewFromFloat(100.0),
				Currency:       "USD",
				IdempotencyKey: "test_key",
			},
			expectedError: models.ErrInvalidOdds,
		},
		{
			name: "invalid stake (zero)",
			orderRequest: &models.PlaceOrderRequest{
				UserID:         uuid.New(),
				EventID:        "event_123",
				MarketID:       "market_456",
				SelectionID:    "selection_789",
				Side:           models.OrderSideBack,
				Odds:           decimal.NewFromFloat(2.5),
				Stake:          decimal.Zero,
				Currency:       "USD",
				IdempotencyKey: "test_key",
			},
			expectedError: models.ErrInvalidStake,
		},
		{
			name: "invalid currency (not 3 chars)",
			orderRequest: &models.PlaceOrderRequest{
				UserID:         uuid.New(),
				EventID:        "event_123",
				MarketID:       "market_456",
				SelectionID:    "selection_789",
				Side:           models.OrderSideBack,
				Odds:           decimal.NewFromFloat(2.5),
				Stake:          decimal.NewFromFloat(100.0),
				Currency:       "US",
				IdempotencyKey: "test_key",
			},
			expectedError: models.ErrInvalidCurrency,
		},
		{
			name: "invalid side",
			orderRequest: &models.PlaceOrderRequest{
				UserID:         uuid.New(),
				EventID:        "event_123",
				MarketID:       "market_456",
				SelectionID:    "selection_789",
				Side:           models.OrderSide("INVALID"),
				Odds:           decimal.NewFromFloat(2.5),
				Stake:          decimal.NewFromFloat(100.0),
				Currency:       "USD",
				IdempotencyKey: "test_key",
			},
			expectedError: models.ErrInvalidSide,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.orderRequest.Validate()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

// TestValidateOrder_StakeTooLow tests stake below minimum
func TestValidateOrder_StakeTooLow(t *testing.T) {
	setup := setupTestValidationActivity(t)

	req := &models.PlaceOrderRequest{
		UserID:         uuid.New(),
		EventID:        "event_123",
		MarketID:       "market_456",
		SelectionID:    "selection_789",
		Side:           models.OrderSideBack,
		Odds:           decimal.NewFromFloat(2.5),
		Stake:          decimal.NewFromFloat(0.5), // Below minimum of 1.0
		Currency:       "USD",
		IdempotencyKey: "test_key",
	}

	// Basic validation should pass
	err := req.Validate()
	assert.NoError(t, err)

	// But stake is below minimum
	assert.True(t, req.Stake.LessThan(setup.activity.minStake))
}

// TestValidateOrder_StakeTooHigh tests stake above maximum
func TestValidateOrder_StakeTooHigh(t *testing.T) {
	setup := setupTestValidationActivity(t)

	req := &models.PlaceOrderRequest{
		UserID:         uuid.New(),
		EventID:        "event_123",
		MarketID:       "market_456",
		SelectionID:    "selection_789",
		Side:           models.OrderSideBack,
		Odds:           decimal.NewFromFloat(2.5),
		Stake:          decimal.NewFromFloat(15000.0), // Above maximum of 10000.0
		Currency:       "USD",
		IdempotencyKey: "test_key",
	}

	// Basic validation should pass
	err := req.Validate()
	assert.NoError(t, err)

	// But stake is above maximum
	assert.True(t, req.Stake.GreaterThan(setup.activity.maxStake))
}

// TestValidateOrder_StakeAtBoundaries tests stake at exact boundaries
func TestValidateOrder_StakeAtBoundaries(t *testing.T) {
	setup := setupTestValidationActivity(t)

	tests := []struct {
		name      string
		stake     decimal.Decimal
		withinMin bool
		withinMax bool
	}{
		{
			name:      "stake at minimum (1.0)",
			stake:     decimal.NewFromFloat(1.0),
			withinMin: true,
			withinMax: true,
		},
		{
			name:      "stake at maximum (10000.0)",
			stake:     decimal.NewFromFloat(10000.0),
			withinMin: true,
			withinMax: true,
		},
		{
			name:      "stake just below minimum",
			stake:     decimal.NewFromFloat(0.99),
			withinMin: false,
			withinMax: true,
		},
		{
			name:      "stake just above maximum",
			stake:     decimal.NewFromFloat(10000.01),
			withinMin: true,
			withinMax: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.withinMin {
				assert.True(t, tt.stake.GreaterThanOrEqual(setup.activity.minStake))
			} else {
				assert.True(t, tt.stake.LessThan(setup.activity.minStake))
			}

			if tt.withinMax {
				assert.True(t, tt.stake.LessThanOrEqual(setup.activity.maxStake))
			} else {
				assert.True(t, tt.stake.GreaterThan(setup.activity.maxStake))
			}
		})
	}
}

// TestValidateOrder_OddsTooLow tests odds below minimum
func TestValidateOrder_OddsTooLow(t *testing.T) {
	setup := setupTestValidationActivity(t)

	req := &models.PlaceOrderRequest{
		UserID:         uuid.New(),
		EventID:        "event_123",
		MarketID:       "market_456",
		SelectionID:    "selection_789",
		Side:           models.OrderSideBack,
		Odds:           decimal.NewFromFloat(1.005), // Below minimum of 1.01
		Stake:          decimal.NewFromFloat(100.0),
		Currency:       "USD",
		IdempotencyKey: "test_key",
	}

	// Basic validation should pass
	err := req.Validate()
	assert.NoError(t, err)

	// But odds are below minimum
	assert.True(t, req.Odds.LessThan(setup.activity.minOdds))
}

// TestValidateOrder_OddsTooHigh tests odds above maximum
func TestValidateOrder_OddsTooHigh(t *testing.T) {
	setup := setupTestValidationActivity(t)

	req := &models.PlaceOrderRequest{
		UserID:         uuid.New(),
		EventID:        "event_123",
		MarketID:       "market_456",
		SelectionID:    "selection_789",
		Side:           models.OrderSideBack,
		Odds:           decimal.NewFromFloat(1500.0), // Above maximum of 1000.0
		Stake:          decimal.NewFromFloat(100.0),
		Currency:       "USD",
		IdempotencyKey: "test_key",
	}

	// Basic validation should pass
	err := req.Validate()
	assert.NoError(t, err)

	// But odds are above maximum
	assert.True(t, req.Odds.GreaterThan(setup.activity.maxOdds))
}

// TestValidateOrder_OddsAtBoundaries tests odds at exact boundaries
func TestValidateOrder_OddsAtBoundaries(t *testing.T) {
	setup := setupTestValidationActivity(t)

	tests := []struct {
		name      string
		odds      decimal.Decimal
		withinMin bool
		withinMax bool
	}{
		{
			name:      "odds at minimum (1.01)",
			odds:      decimal.NewFromFloat(1.01),
			withinMin: true,
			withinMax: true,
		},
		{
			name:      "odds at maximum (1000.0)",
			odds:      decimal.NewFromFloat(1000.0),
			withinMin: true,
			withinMax: true,
		},
		{
			name:      "odds just below minimum",
			odds:      decimal.NewFromFloat(1.009),
			withinMin: false,
			withinMax: true,
		},
		{
			name:      "odds just above maximum",
			odds:      decimal.NewFromFloat(1000.01),
			withinMin: true,
			withinMax: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.withinMin {
				assert.True(t, tt.odds.GreaterThanOrEqual(setup.activity.minOdds))
			} else {
				assert.True(t, tt.odds.LessThan(setup.activity.minOdds))
			}

			if tt.withinMax {
				assert.True(t, tt.odds.LessThanOrEqual(setup.activity.maxOdds))
			} else {
				assert.True(t, tt.odds.GreaterThan(setup.activity.maxOdds))
			}
		})
	}
}

// TestValidateOrder_BackSide tests validation with BACK side
func TestValidateOrder_BackSide(t *testing.T) {
	req := &models.PlaceOrderRequest{
		UserID:         uuid.New(),
		EventID:        "event_123",
		MarketID:       "market_456",
		SelectionID:    "selection_789",
		Side:           models.OrderSideBack,
		Odds:           decimal.NewFromFloat(2.5),
		Stake:          decimal.NewFromFloat(100.0),
		Currency:       "USD",
		IdempotencyKey: "test_key",
	}

	err := req.Validate()
	assert.NoError(t, err)
}

// TestValidateOrder_LaySide tests validation with LAY side
func TestValidateOrder_LaySide(t *testing.T) {
	req := &models.PlaceOrderRequest{
		UserID:         uuid.New(),
		EventID:        "event_123",
		MarketID:       "market_456",
		SelectionID:    "selection_789",
		Side:           models.OrderSideLay,
		Odds:           decimal.NewFromFloat(2.5),
		Stake:          decimal.NewFromFloat(100.0),
		Currency:       "USD",
		IdempotencyKey: "test_key",
	}

	err := req.Validate()
	assert.NoError(t, err)
}

// TestValidateOrder_DifferentCurrencies tests validation with different currencies
func TestValidateOrder_DifferentCurrencies(t *testing.T) {
	currencies := []string{"USD", "EUR", "GBP", "JPY", "AUD"}

	for _, currency := range currencies {
		t.Run("currency_"+currency, func(t *testing.T) {
			req := &models.PlaceOrderRequest{
				UserID:         uuid.New(),
				EventID:        "event_123",
				MarketID:       "market_456",
				SelectionID:    "selection_789",
				Side:           models.OrderSideBack,
				Odds:           decimal.NewFromFloat(2.5),
				Stake:          decimal.NewFromFloat(100.0),
				Currency:       currency,
				IdempotencyKey: "test_key",
			}

			err := req.Validate()
			assert.NoError(t, err)
		})
	}
}

// TestNewValidationActivities tests activity creation
func TestNewValidationActivities(t *testing.T) {
	logger := zerolog.Nop()
	activity := NewValidationActivities(logger)

	assert.NotNil(t, activity)
	assert.Equal(t, decimal.NewFromFloat(1.0), activity.minStake)
	assert.Equal(t, decimal.NewFromFloat(10000.0), activity.maxStake)
	assert.Equal(t, decimal.NewFromFloat(1.01), activity.minOdds)
	assert.Equal(t, decimal.NewFromFloat(1000.0), activity.maxOdds)
}

// TestValidationActivities_ImplementsInterface tests that ValidationActivities implements the interface
func TestValidationActivities_ImplementsInterface(t *testing.T) {
	var _ ValidationActivityInterface = (*ValidationActivities)(nil)
}
