package grpc

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"
	"go.temporal.io/sdk/client"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	orderv1 "github.com/cypherlabdev/cypherlabdev-protos/gen/go/order/v1"
	"github.com/cypherlabdev/order-validator-service/internal/models"
	"github.com/cypherlabdev/order-validator-service/internal/workflow"
)

// OrderHandler implements the gRPC ValidatorService server
type OrderHandler struct {
	orderv1.UnimplementedValidatorServiceServer
	temporalClient client.Client
	logger         zerolog.Logger
}

// NewOrderHandler creates a new order gRPC handler
func NewOrderHandler(temporalClient client.Client, logger zerolog.Logger) *OrderHandler {
	return &OrderHandler{
		temporalClient: temporalClient,
		logger:         logger.With().Str("component", "order_handler").Logger(),
	}
}

// PlaceBet handles bet placement requests by initiating Temporal workflow
func (h *OrderHandler) PlaceBet(ctx context.Context, req *orderv1.PlaceBetRequest) (*orderv1.PlaceBetResponse, error) {
	// Validate request
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		h.logger.Debug().Str("user_id", req.UserId).Msg("invalid user ID format")
		return nil, status.Error(codes.InvalidArgument, "invalid user ID format")
	}

	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		h.logger.Debug().Str("amount", req.Amount).Msg("invalid amount format")
		return nil, status.Error(codes.InvalidArgument, "invalid amount format")
	}

	if req.IdempotencyKey == "" {
		return nil, status.Error(codes.InvalidArgument, "idempotency_key is required")
	}

	// Map proto fields to internal model
	// Note: Proto has simplified fields (event_id, bet_type, selection)
	// We map them to our more detailed internal model (event_id, market_id, selection_id, side, odds, stake)
	// For now, we'll use event_id directly and derive market/selection from bet_type and selection
	orderRequest := &models.PlaceOrderRequest{
		UserID:         userID,
		EventID:        req.EventId,
		MarketID:       req.BetType,  // Map bet_type to market_id (e.g., "moneyline", "spread")
		SelectionID:    req.Selection, // Map selection to selection_id (e.g., "lakers_win")
		Side:           models.OrderSideBack, // Default to BACK bet
		Odds:           decimal.NewFromInt(2), // Default odds (should come from odds-optimizer later)
		Stake:          amount,
		Currency:       "USD", // Default currency
		IdempotencyKey: req.IdempotencyKey,
		Metadata:       nil,
	}

	// Generate saga ID (workflow ID)
	sagaID := fmt.Sprintf("place-bet-%s", uuid.New().String())

	// Start Temporal workflow
	workflowOptions := client.StartWorkflowOptions{
		ID:        sagaID,
		TaskQueue: "order-validator",
	}

	workflowInput := workflow.PlaceOrderWorkflowInput{
		OrderRequest: orderRequest,
		SagaID:       sagaID,
	}

	we, err := h.temporalClient.ExecuteWorkflow(ctx, workflowOptions, workflow.PlaceOrderWorkflow, workflowInput)
	if err != nil {
		h.logger.Error().Err(err).Str("saga_id", sagaID).Msg("failed to start workflow")
		return nil, status.Error(codes.Internal, "failed to start bet workflow")
	}

	h.logger.Info().
		Str("saga_id", sagaID).
		Str("workflow_id", we.GetID()).
		Str("run_id", we.GetRunID()).
		Msg("bet workflow started")

	// Return immediately with saga ID (async processing)
	// Status will be "processing" initially - actual outcome comes from workflow completion
	return &orderv1.PlaceBetResponse{
		SagaId:  sagaID,
		OrderId: "", // Will be populated when workflow completes
		Status:  "processing",
	}, nil
}
