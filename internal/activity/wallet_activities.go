package activity

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"go.temporal.io/sdk/activity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	walletpb "github.com/cypherlabdev/cypherlabdev-protos/gen/go/wallet/v1"
	"github.com/cypherlabdev/order-validator-service/internal/workflow"
)

// WalletActivities implements wallet-related activities
type WalletActivities struct {
	walletClient walletpb.WalletServiceClient
	logger       zerolog.Logger
}

// NewWalletActivities creates a new wallet activities instance
func NewWalletActivities(walletServiceAddr string, logger zerolog.Logger) (*WalletActivities, error) {
	// Connect to wallet-service
	conn, err := grpc.NewClient(
		walletServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to wallet-service: %w", err)
	}

	client := walletpb.NewWalletServiceClient(conn)

	return &WalletActivities{
		walletClient: client,
		logger:       logger.With().Str("component", "wallet_activities").Logger(),
	}, nil
}

// ReserveFundsActivity reserves funds in the wallet
func (a *WalletActivities) ReserveFunds(ctx context.Context, input workflow.ReserveFundsInput) (*workflow.ReserveFundsResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("ReserveFunds activity started", "user_id", input.UserID, "amount", input.Amount)

	req := &walletpb.ReserveBalanceRequest{
		UserId:         input.UserID,
		Amount:         input.Amount,
		SagaId:         input.SagaID,
		IdempotencyKey: input.IdempotencyKey,
	}

	resp, err := a.walletClient.ReserveBalance(ctx, req)
	if err != nil {
		logger.Error("Failed to reserve funds", "error", err)
		return nil, fmt.Errorf("reserve funds: %w", err)
	}

	logger.Info("Funds reserved successfully", "reservation_id", resp.ReservationId)

	return &workflow.ReserveFundsResult{
		ReservationID: resp.ReservationId,
		Status:        resp.Status,
	}, nil
}

// CommitReservationActivity commits a reservation
func (a *WalletActivities) CommitReservation(ctx context.Context, input workflow.CommitReservationInput) (*workflow.CommitReservationResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("CommitReservation activity started", "reservation_id", input.ReservationID)

	req := &walletpb.CommitReservationRequest{
		ReservationId:  input.ReservationID,
		SagaId:         input.SagaID,
		IdempotencyKey: input.IdempotencyKey,
	}

	resp, err := a.walletClient.CommitReservation(ctx, req)
	if err != nil {
		logger.Error("Failed to commit reservation", "error", err)
		return nil, fmt.Errorf("commit reservation: %w", err)
	}

	logger.Info("Reservation committed successfully")

	return &workflow.CommitReservationResult{
		Status: resp.Status,
	}, nil
}

// CancelReservationActivity cancels a reservation
func (a *WalletActivities) CancelReservation(ctx context.Context, input workflow.CancelReservationInput) (*workflow.CancelReservationResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("CancelReservation activity started", "reservation_id", input.ReservationID)

	req := &walletpb.CancelReservationRequest{
		ReservationId:  input.ReservationID,
		SagaId:         input.SagaID,
		IdempotencyKey: input.IdempotencyKey,
	}

	resp, err := a.walletClient.CancelReservation(ctx, req)
	if err != nil {
		logger.Error("Failed to cancel reservation", "error", err)
		return nil, fmt.Errorf("cancel reservation: %w", err)
	}

	logger.Info("Reservation cancelled successfully")

	return &workflow.CancelReservationResult{
		Status: resp.Status,
	}, nil
}
