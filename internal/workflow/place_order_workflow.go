package workflow

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/workflow"

	"github.com/cypherlabdev/order-validator-service/internal/models"
)

// PlaceOrderWorkflowInput is the input for PlaceOrderWorkflow
type PlaceOrderWorkflowInput struct {
	OrderRequest *models.PlaceOrderRequest
	SagaID       string
}

// PlaceOrderWorkflowResult is the result of PlaceOrderWorkflow
type PlaceOrderWorkflowResult struct {
	OrderID       string
	Status        models.OrderStatus
	ReservationID string
	MatchID       string
	FailureReason string
}

// PlaceOrderWorkflow orchestrates the entire order placement saga
// This workflow implements the saga pattern with compensation
//
// Saga Steps:
// 1. Validate Order (validation rules)
// 2. Reserve Funds (wallet-service)
// 3. Place Order (order-book)
// 4. Commit Reservation (wallet-service)
//
// Compensation (on any failure):
// - Cancel Reservation (wallet-service)
// - Cancel Order (order-book)
func PlaceOrderWorkflow(ctx workflow.Context, input PlaceOrderWorkflowInput) (*PlaceOrderWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("PlaceOrderWorkflow started", "saga_id", input.SagaID)

	// Configure activity options
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		RetryPolicy: &workflow.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	result := &PlaceOrderWorkflowResult{
		SagaID: input.SagaID,
		Status: models.OrderStatusPending,
	}

	var reservationID string
	var orderID string

	// Step 1: Validate Order
	logger.Info("Step 1: Validating order")
	var validationResult *ValidationResult
	err := workflow.ExecuteActivity(ctx, ValidateOrderActivity, ValidateOrderInput{
		OrderRequest: input.OrderRequest,
	}).Get(ctx, &validationResult)

	if err != nil {
		logger.Error("Order validation failed", "error", err)
		result.Status = models.OrderStatusFailed
		result.FailureReason = fmt.Sprintf("validation failed: %v", err)
		return result, err
	}

	if !validationResult.Valid {
		logger.Warn("Order validation rejected", "reason", validationResult.Reason)
		result.Status = models.OrderStatusFailed
		result.FailureReason = validationResult.Reason
		return result, fmt.Errorf("validation failed: %s", validationResult.Reason)
	}

	logger.Info("Order validation passed")

	// Step 2: Reserve Funds in Wallet
	logger.Info("Step 2: Reserving funds", "amount", input.OrderRequest.Stake)
	result.Status = models.OrderStatusReserving

	var reserveResult *ReserveFundsResult
	err = workflow.ExecuteActivity(ctx, ReserveFundsActivity, ReserveFundsInput{
		UserID:         input.OrderRequest.UserID.String(),
		Amount:         input.OrderRequest.Stake.String(),
		Currency:       input.OrderRequest.Currency,
		SagaID:         input.SagaID,
		IdempotencyKey: input.OrderRequest.IdempotencyKey + "-reserve",
	}).Get(ctx, &reserveResult)

	if err != nil {
		logger.Error("Failed to reserve funds", "error", err)
		result.Status = models.OrderStatusFailed
		result.FailureReason = fmt.Sprintf("fund reservation failed: %v", err)
		return result, err
	}

	reservationID = reserveResult.ReservationID
	result.ReservationID = reservationID
	logger.Info("Funds reserved successfully", "reservation_id", reservationID)

	// From here on, we must compensate on failure
	defer func() {
		if result.Status == models.OrderStatusFailed || result.Status == models.OrderStatusCancelled {
			compensate(ctx, reservationID, orderID, input.SagaID, logger)
		}
	}()

	// Step 3: Place Order in Order Book
	logger.Info("Step 3: Placing order in order-book")
	result.Status = models.OrderStatusMatching

	var placeOrderResult *PlaceOrderInBookResult
	err = workflow.ExecuteActivity(ctx, PlaceOrderInBookActivity, PlaceOrderInBookInput{
		UserID:         input.OrderRequest.UserID.String(),
		EventID:        input.OrderRequest.EventID,
		MarketID:       input.OrderRequest.MarketID,
		SelectionID:    input.OrderRequest.SelectionID,
		Side:           string(input.OrderRequest.Side),
		Odds:           input.OrderRequest.Odds.String(),
		Stake:          input.OrderRequest.Stake.String(),
		Currency:       input.OrderRequest.Currency,
		ReservationID:  reservationID,
		SagaID:         input.SagaID,
		IdempotencyKey: input.OrderRequest.IdempotencyKey + "-order",
	}).Get(ctx, &placeOrderResult)

	if err != nil {
		logger.Error("Failed to place order in book", "error", err)
		result.Status = models.OrderStatusFailed
		result.FailureReason = fmt.Sprintf("order placement failed: %v", err)
		return result, err
	}

	orderID = placeOrderResult.OrderID
	result.OrderID = orderID
	logger.Info("Order placed in book", "order_id", orderID, "status", placeOrderResult.Status)

	// Step 4: Commit Reservation (funds are now committed)
	logger.Info("Step 4: Committing reservation")

	var commitResult *CommitReservationResult
	err = workflow.ExecuteActivity(ctx, CommitReservationActivity, CommitReservationInput{
		ReservationID:  reservationID,
		SagaID:         input.SagaID,
		IdempotencyKey: input.OrderRequest.IdempotencyKey + "-commit",
	}).Get(ctx, &commitResult)

	if err != nil {
		logger.Error("Failed to commit reservation", "error", err)
		result.Status = models.OrderStatusFailed
		result.FailureReason = fmt.Sprintf("reservation commit failed: %v", err)
		return result, err
	}

	logger.Info("Reservation committed successfully")

	// Success!
	result.Status = models.OrderStatusSettled
	result.MatchID = placeOrderResult.MatchID

	logger.Info("PlaceOrderWorkflow completed successfully",
		"order_id", orderID,
		"reservation_id", reservationID,
		"match_id", placeOrderResult.MatchID)

	return result, nil
}

// compensate performs compensation actions when the saga fails
func compensate(ctx workflow.Context, reservationID, orderID, sagaID string, logger workflow.Logger) {
	logger.Warn("Saga failed, starting compensation")

	// Configure compensation activity options (no retries, best effort)
	compensationOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		RetryPolicy: &workflow.RetryPolicy{
			InitialInterval: time.Second,
			MaximumAttempts: 1, // Best effort, don't retry compensation
		},
	}
	compensationCtx := workflow.WithActivityOptions(ctx, compensationOptions)

	// Compensate in reverse order of execution

	// 1. Cancel order in order-book (if placed)
	if orderID != "" {
		logger.Info("Compensating: Cancelling order", "order_id", orderID)
		var cancelOrderResult *CancelOrderResult
		err := workflow.ExecuteActivity(compensationCtx, CancelOrderActivity, CancelOrderInput{
			OrderID:        orderID,
			SagaID:         sagaID,
			IdempotencyKey: sagaID + "-cancel-order",
		}).Get(compensationCtx, &cancelOrderResult)

		if err != nil {
			logger.Error("Compensation failed: Unable to cancel order", "order_id", orderID, "error", err)
			// Continue with other compensations
		} else {
			logger.Info("Order cancelled successfully", "order_id", orderID)
		}
	}

	// 2. Cancel reservation in wallet (if reserved)
	if reservationID != "" {
		logger.Info("Compensating: Cancelling reservation", "reservation_id", reservationID)
		var cancelReservationResult *CancelReservationResult
		err := workflow.ExecuteActivity(compensationCtx, CancelReservationActivity, CancelReservationInput{
			ReservationID:  reservationID,
			SagaID:         sagaID,
			IdempotencyKey: sagaID + "-cancel-reservation",
		}).Get(compensationCtx, &cancelReservationResult)

		if err != nil {
			logger.Error("Compensation failed: Unable to cancel reservation", "reservation_id", reservationID, "error", err)
			// This is critical - should alert operations
		} else {
			logger.Info("Reservation cancelled successfully", "reservation_id", reservationID)
		}
	}

	logger.Info("Compensation completed")
}
