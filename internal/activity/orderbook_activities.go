package activity

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"go.temporal.io/sdk/activity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	orderbookpb "github.com/cypherlabdev/cypherlabdev-protos/gen/go/orderbook/v1"
	"github.com/cypherlabdev/order-validator-service/internal/workflow"
)

// OrderBookActivities implements order-book-related activities
type OrderBookActivities struct {
	orderBookClient orderbookpb.OrderBookServiceClient
	logger          zerolog.Logger
}

// NewOrderBookActivities creates a new order-book activities instance
func NewOrderBookActivities(orderBookServiceAddr string, logger zerolog.Logger) (*OrderBookActivities, error) {
	// Connect to order-book-service
	conn, err := grpc.NewClient(
		orderBookServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to order-book-service: %w", err)
	}

	client := orderbookpb.NewOrderBookServiceClient(conn)

	return &OrderBookActivities{
		orderBookClient: client,
		logger:          logger.With().Str("component", "orderbook_activities").Logger(),
	}, nil
}

// PlaceOrderInBookActivity places an order in the order book
func (a *OrderBookActivities) PlaceOrderInBook(ctx context.Context, input workflow.PlaceOrderInBookInput) (*workflow.PlaceOrderInBookResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("PlaceOrderInBook activity started", "event_id", input.EventID, "market_id", input.MarketID)

	req := &orderbookpb.PlaceBetRequest{
		UserId:         input.UserID,
		EventId:        input.EventID,
		MarketId:       input.MarketID,
		SelectionId:    input.SelectionID,
		BetType:        input.Side, // "BACK" or "LAY"
		Odds:           input.Odds,
		Stake:          input.Stake,
		ReservationId:  input.ReservationID,
		SagaId:         input.SagaID,
		IdempotencyKey: input.IdempotencyKey,
	}

	resp, err := a.orderBookClient.PlaceBet(ctx, req)
	if err != nil {
		logger.Error("Failed to place order in book", "error", err)
		return nil, fmt.Errorf("place order in book: %w", err)
	}

	logger.Info("Order placed in book successfully", "order_id", resp.OrderId, "status", resp.Status)

	// Determine match status from response
	matchID := ""
	if resp.MatchId != "" {
		matchID = resp.MatchId
	}

	return &workflow.PlaceOrderInBookResult{
		OrderID: resp.OrderId,
		MatchID: matchID,
		Status:  resp.Status, // "MATCHED", "PENDING", or "PARTIALLY_FILLED"
	}, nil
}

// CancelOrderActivity cancels an order in the order book
func (a *OrderBookActivities) CancelOrder(ctx context.Context, input workflow.CancelOrderInput) (*workflow.CancelOrderResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("CancelOrder activity started", "order_id", input.OrderID)

	req := &orderbookpb.CancelBetRequest{
		OrderId:        input.OrderID,
		SagaId:         input.SagaID,
		IdempotencyKey: input.IdempotencyKey,
	}

	resp, err := a.orderBookClient.CancelBet(ctx, req)
	if err != nil {
		logger.Error("Failed to cancel order", "error", err)
		return nil, fmt.Errorf("cancel order: %w", err)
	}

	logger.Info("Order cancelled successfully", "status", resp.Status)

	return &workflow.CancelOrderResult{
		Status: resp.Status,
	}, nil
}
