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

	"github.com/cypherlabdev/order-validator-service/internal/models"
	"github.com/cypherlabdev/order-validator-service/internal/workflow"
)

// OrderHandler implements the gRPC OrderValidatorService server
type OrderHandler struct {
	// UnimplementedOrderValidatorServiceServer - will be added when proto is defined
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

// PlaceOrder handles order placement requests
func (h *OrderHandler) PlaceOrder(ctx context.Context, req *PlaceOrderRequest) (*PlaceOrderResponse, error) {
	// Validate request
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		h.logger.Debug().Str("user_id", req.UserId).Msg("invalid user ID format")
		return nil, status.Error(codes.InvalidArgument, "invalid user ID format")
	}

	stake, err := decimal.NewFromString(req.Stake)
	if err != nil {
		h.logger.Debug().Str("stake", req.Stake).Msg("invalid stake format")
		return nil, status.Error(codes.InvalidArgument, "invalid stake format")
	}

	odds, err := decimal.NewFromString(req.Odds)
	if err != nil {
		h.logger.Debug().Str("odds", req.Odds).Msg("invalid odds format")
		return nil, status.Error(codes.InvalidArgument, "invalid odds format")
	}

	if req.IdempotencyKey == "" {
		return nil, status.Error(codes.InvalidArgument, "idempotency_key is required")
	}

	// Create order request
	orderRequest := &models.PlaceOrderRequest{
		UserID:         userID,
		EventID:        req.EventId,
		MarketID:       req.MarketId,
		SelectionID:    req.SelectionId,
		Side:           models.OrderSide(req.Side),
		Odds:           odds,
		Stake:          stake,
		Currency:       req.Currency,
		IdempotencyKey: req.IdempotencyKey,
		Metadata:       nil, // TODO: Parse from proto if needed
	}

	// Generate saga ID (workflow ID)
	sagaID := fmt.Sprintf("place-order-%s", uuid.New().String())

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
		return nil, status.Error(codes.Internal, "failed to start order workflow")
	}

	h.logger.Info().
		Str("saga_id", sagaID).
		Str("workflow_id", we.GetID()).
		Str("run_id", we.GetRunID()).
		Msg("order workflow started")

	// Return immediately with saga ID (async processing)
	return &PlaceOrderResponse{
		SagaId: sagaID,
		Status: string(models.OrderStatusPending),
	}, nil
}

// GetOrderStatus retrieves the status of an order workflow
func (h *OrderHandler) GetOrderStatus(ctx context.Context, req *GetOrderStatusRequest) (*GetOrderStatusResponse, error) {
	if req.SagaId == "" {
		return nil, status.Error(codes.InvalidArgument, "saga_id is required")
	}

	// Query workflow execution
	workflowRun := h.temporalClient.GetWorkflow(ctx, req.SagaId, "")

	var result workflow.PlaceOrderWorkflowResult
	err := workflowRun.Get(ctx, &result)

	if err != nil {
		// Workflow still running or failed
		h.logger.Debug().Str("saga_id", req.SagaId).Err(err).Msg("workflow not completed")

		// Check if workflow is still running
		description, descErr := h.temporalClient.DescribeWorkflowExecution(ctx, req.SagaId, "")
		if descErr != nil {
			return nil, status.Error(codes.NotFound, "workflow not found")
		}

		// Return current status
		return &GetOrderStatusResponse{
			SagaId: req.SagaId,
			Status: description.WorkflowExecutionInfo.Status.String(),
		}, nil
	}

	// Workflow completed
	return &GetOrderStatusResponse{
		SagaId:        req.SagaId,
		Status:        string(result.Status),
		OrderId:       result.OrderID,
		ReservationId: result.ReservationID,
		MatchId:       result.MatchID,
		FailureReason: result.FailureReason,
	}, nil
}

// Temporary request/response types (will be replaced with proto-generated types)
type PlaceOrderRequest struct {
	UserId         string
	EventId        string
	MarketId       string
	SelectionId    string
	Side           string
	Odds           string
	Stake          string
	Currency       string
	IdempotencyKey string
}

type PlaceOrderResponse struct {
	SagaId string
	Status string
}

type GetOrderStatusRequest struct {
	SagaId string
}

type GetOrderStatusResponse struct {
	SagaId        string
	Status        string
	OrderId       string
	ReservationId string
	MatchId       string
	FailureReason string
}
