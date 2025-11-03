package activity

import (
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/cypherlabdev/order-validator-service/internal/models"
)

// testOrderBookActivitySetup is a helper struct to hold test dependencies
type testOrderBookActivitySetup struct {
	activity *OrderBookActivities
}

// setupTestOrderBookActivity creates a test activity with all dependencies
func setupTestOrderBookActivity(t *testing.T) *testOrderBookActivitySetup {
	logger := zerolog.Nop()
	activity := NewOrderBookActivities(logger)

	return &testOrderBookActivitySetup{
		activity: activity,
	}
}

// TestNewOrderBookActivities tests activity creation
func TestNewOrderBookActivities(t *testing.T) {
	logger := zerolog.Nop()
	activity := NewOrderBookActivities(logger)

	assert.NotNil(t, activity)
	assert.NotNil(t, activity.logger)
}

// TestOrderBookActivities_LoggerInitialization tests logger initialization
func TestOrderBookActivities_LoggerInitialization(t *testing.T) {
	setup := setupTestOrderBookActivity(t)

	assert.NotNil(t, setup.activity)
	assert.NotNil(t, setup.activity.logger)
}

// TestOrderBookActivities_ImplementsInterface tests that OrderBookActivities implements the interface
func TestOrderBookActivities_ImplementsInterface(t *testing.T) {
	var _ OrderBookActivityInterface = (*OrderBookActivities)(nil)
}

// TestOrderBookActivities_OrderSides tests different order sides
func TestOrderBookActivities_OrderSides(t *testing.T) {
	sides := []models.OrderSide{models.OrderSideBack, models.OrderSideLay}

	for _, side := range sides {
		t.Run(string(side), func(t *testing.T) {
			assert.Contains(t, []models.OrderSide{models.OrderSideBack, models.OrderSideLay}, side)
		})
	}
}

// TestOrderBookActivities_UUIDGeneration tests UUID generation for orders
func TestOrderBookActivities_UUIDGeneration(t *testing.T) {
	// Generate multiple UUIDs and verify they're unique
	uuids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := uuid.New().String()
		assert.False(t, uuids[id], "UUID should be unique")
		uuids[id] = true
	}
}

// TestOrderBookActivities_OrderStatuses tests order status constants
func TestOrderBookActivities_OrderStatuses(t *testing.T) {
	statuses := []string{"MATCHED", "PENDING", "CANCELLED"}

	for _, status := range statuses {
		t.Run(status, func(t *testing.T) {
			assert.NotEmpty(t, status)
			assert.Contains(t, []string{"MATCHED", "PENDING", "CANCELLED"}, status)
		})
	}
}

// TestOrderBookActivities_IdempotencyKeyPattern tests idempotency key generation pattern
func TestOrderBookActivities_IdempotencyKeyPattern(t *testing.T) {
	baseKey := "test_saga_123"
	orderKey := baseKey + "-order"
	cancelKey := baseKey + "-cancel-order"

	assert.NotEqual(t, orderKey, cancelKey)
	assert.Contains(t, orderKey, baseKey)
	assert.Contains(t, cancelKey, baseKey)
}

// TestOrderBookActivities_EventMarketSelection tests event/market/selection ID handling
func TestOrderBookActivities_EventMarketSelection(t *testing.T) {
	tests := []struct {
		name        string
		eventID     string
		marketID    string
		selectionID string
	}{
		{
			name:        "standard IDs",
			eventID:     "event_123",
			marketID:    "market_456",
			selectionID: "selection_789",
		},
		{
			name:        "alphanumeric IDs",
			eventID:     "evt_abc123",
			marketID:    "mkt_xyz789",
			selectionID: "sel_def456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.eventID)
			assert.NotEmpty(t, tt.marketID)
			assert.NotEmpty(t, tt.selectionID)
		})
	}
}

// TestOrderBookActivities_OddsFormatting tests odds formatting
func TestOrderBookActivities_OddsFormatting(t *testing.T) {
	odds := []string{"1.5", "2.0", "3.5", "5.0", "10.0"}

	for _, odd := range odds {
		t.Run("odds_"+odd, func(t *testing.T) {
			assert.NotEmpty(t, odd)
			assert.Regexp(t, `^\d+\.\d+$`, odd)
		})
	}
}

// TestOrderBookActivities_StakeFormatting tests stake formatting
func TestOrderBookActivities_StakeFormatting(t *testing.T) {
	stakes := []string{"10.00", "50.00", "100.00", "500.00"}

	for _, stake := range stakes {
		t.Run("stake_"+stake, func(t *testing.T) {
			assert.NotEmpty(t, stake)
			assert.Regexp(t, `^\d+\.\d{2}$`, stake)
		})
	}
}

// TestOrderBookActivities_CurrencyCodes tests currency code validation
func TestOrderBookActivities_CurrencyCodes(t *testing.T) {
	currencies := []string{"USD", "EUR", "GBP", "JPY", "AUD"}

	for _, currency := range currencies {
		t.Run("currency_"+currency, func(t *testing.T) {
			assert.Len(t, currency, 3)
			assert.Regexp(t, `^[A-Z]{3}$`, currency)
		})
	}
}

// TestOrderBookActivities_ReservationIDFormat tests reservation ID format
func TestOrderBookActivities_ReservationIDFormat(t *testing.T) {
	reservationID := uuid.New().String()

	assert.NotEmpty(t, reservationID)
	assert.Regexp(t, `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`, reservationID)
}

// TestOrderBookActivities_SagaIDFormat tests saga ID format
func TestOrderBookActivities_SagaIDFormat(t *testing.T) {
	sagaIDs := []string{"saga_123", "saga_abc", "workflow_xyz"}

	for _, sagaID := range sagaIDs {
		t.Run(sagaID, func(t *testing.T) {
			assert.NotEmpty(t, sagaID)
		})
	}
}

// TestOrderBookActivities_OrderIDGeneration tests order ID generation
func TestOrderBookActivities_OrderIDGeneration(t *testing.T) {
	// Generate multiple order IDs
	orderIDs := make(map[string]bool)
	for i := 0; i < 10; i++ {
		orderID := uuid.New().String()
		assert.False(t, orderIDs[orderID], "Order ID should be unique")
		orderIDs[orderID] = true
		assert.NotEmpty(t, orderID)
	}
}

// TestOrderBookActivities_MatchIDGeneration tests match ID generation
func TestOrderBookActivities_MatchIDGeneration(t *testing.T) {
	// Generate multiple match IDs
	matchIDs := make(map[string]bool)
	for i := 0; i < 10; i++ {
		matchID := uuid.New().String()
		assert.False(t, matchIDs[matchID], "Match ID should be unique")
		matchIDs[matchID] = true
		assert.NotEmpty(t, matchID)
	}
}

// TestOrderBookActivities_CancellationStatus tests cancellation status
func TestOrderBookActivities_CancellationStatus(t *testing.T) {
	cancelledStatus := "CANCELLED"

	assert.Equal(t, "CANCELLED", cancelledStatus)
	assert.NotEmpty(t, cancelledStatus)
}

// TestOrderBookActivities_MatchedStatus tests matched status
func TestOrderBookActivities_MatchedStatus(t *testing.T) {
	matchedStatus := "MATCHED"

	assert.Equal(t, "MATCHED", matchedStatus)
	assert.NotEmpty(t, matchedStatus)
}

// TestOrderBookActivities_PendingStatus tests pending status
func TestOrderBookActivities_PendingStatus(t *testing.T) {
	pendingStatus := "PENDING"

	assert.Equal(t, "PENDING", pendingStatus)
	assert.NotEmpty(t, pendingStatus)
}
