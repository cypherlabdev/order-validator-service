package activity

import (
	"context"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"go.temporal.io/sdk/activity"

	"github.com/cypherlabdev/order-validator-service/internal/workflow"
)

// OrderBookActivities implements order-book-related activities
type OrderBookActivities struct {
	logger zerolog.Logger
	// TODO: Add order-book client when available
	// orderBookClient orderbookpb.OrderBookServiceClient
}

// NewOrderBookActivities creates a new order-book activities instance
func NewOrderBookActivities(logger zerolog.Logger) *OrderBookActivities {
	return &OrderBookActivities{
		logger: logger.With().Str("component", "orderbook_activities").Logger(),
	}
}

// PlaceOrderInBookActivity places an order in the order book
func (a *OrderBookActivities) PlaceOrderInBook(ctx context.Context, input workflow.PlaceOrderInBookInput) (*workflow.PlaceOrderInBookResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("PlaceOrderInBook activity started", "event_id", input.EventID, "market_id", input.MarketID)

	// TODO: Call order-book-service to place the order
	// For now, we'll simulate a successful placement
	//
	// req := &orderbookpb.PlaceOrderRequest{
	//     UserId:        input.UserID,
	//     EventId:       input.EventID,
	//     MarketId:      input.MarketID,
	//     SelectionId:   input.SelectionID,
	//     Side:          input.Side,
	//     Odds:          input.Odds,
	//     Stake:         input.Stake,
	//     Currency:      input.Currency,
	//     ReservationId: input.ReservationID,
	//     SagaId:        input.SagaID,
	//     IdempotencyKey: input.IdempotencyKey,
	// }
	//
	// resp, err := a.orderBookClient.PlaceOrder(ctx, req)
	// if err != nil {
	//     logger.Error("Failed to place order in book", "error", err)
	//     return nil, fmt.Errorf("place order in book: %w", err)
	// }

	// Simulate successful placement
	orderID := uuid.New().String()
	matchID := uuid.New().String()

	logger.Info("Order placed in book successfully", "order_id", orderID, "match_id", matchID)

	return &workflow.PlaceOrderInBookResult{
		OrderID: orderID,
		MatchID: matchID,
		Status:  "MATCHED", // or "PENDING" if partially filled
	}, nil
}

// CancelOrderActivity cancels an order in the order book
func (a *OrderBookActivities) CancelOrder(ctx context.Context, input workflow.CancelOrderInput) (*workflow.CancelOrderResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("CancelOrder activity started", "order_id", input.OrderID)

	// TODO: Call order-book-service to cancel the order
	// For now, we'll simulate a successful cancellation
	//
	// req := &orderbookpb.CancelOrderRequest{
	//     OrderId:        input.OrderID,
	//     SagaId:         input.SagaID,
	//     IdempotencyKey: input.IdempotencyKey,
	// }
	//
	// resp, err := a.orderBookClient.CancelOrder(ctx, req)
	// if err != nil {
	//     logger.Error("Failed to cancel order", "error", err)
	//     return nil, fmt.Errorf("cancel order: %w", err)
	// }

	logger.Info("Order cancelled successfully")

	return &workflow.CancelOrderResult{
		Status: "CANCELLED",
	}, nil
}
