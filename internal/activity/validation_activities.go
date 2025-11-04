package activity

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"
	"go.temporal.io/sdk/activity"

	"github.com/cypherlabdev/order-validator-service/internal/workflow"
)

// ValidationActivities implements order validation activities
type ValidationActivities struct {
	logger zerolog.Logger
	// TODO: Add data-normalizer client when available
	// dataNormalizerClient datanormalizerpb.DataNormalizerServiceClient

	// Configuration
	minStake decimal.Decimal
	maxStake decimal.Decimal
	minOdds  decimal.Decimal
	maxOdds  decimal.Decimal
}

// NewValidationActivities creates a new validation activities instance
func NewValidationActivities(logger zerolog.Logger) *ValidationActivities {
	return &ValidationActivities{
		logger:   logger.With().Str("component", "validation_activities").Logger(),
		minStake: decimal.NewFromFloat(1.0),   // $1 minimum
		maxStake: decimal.NewFromFloat(10000.0), // $10,000 maximum
		minOdds:  decimal.NewFromFloat(1.01),  // 1.01 minimum odds
		maxOdds:  decimal.NewFromFloat(1000.0), // 1000.0 maximum odds
	}
}

// ValidateOrderActivity validates an order request
func (a *ValidationActivities) ValidateOrder(ctx context.Context, input workflow.ValidateOrderInput) (*workflow.ValidationResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("ValidateOrder activity started")

	req := input.OrderRequest

	// 1. Basic validation (field validation)
	if err := req.Validate(); err != nil {
		logger.Warn("Order failed basic validation", "error", err)
		return &workflow.ValidationResult{
			Valid:  false,
			Reason: err.Error(),
		}, nil
	}

	// 2. Stake limits
	if req.Stake.LessThan(a.minStake) {
		reason := fmt.Sprintf("stake too low: minimum is %s", a.minStake.String())
		logger.Warn("Order rejected", "reason", reason)
		return &workflow.ValidationResult{
			Valid:  false,
			Reason: reason,
		}, nil
	}

	if req.Stake.GreaterThan(a.maxStake) {
		reason := fmt.Sprintf("stake too high: maximum is %s", a.maxStake.String())
		logger.Warn("Order rejected", "reason", reason)
		return &workflow.ValidationResult{
			Valid:  false,
			Reason: reason,
		}, nil
	}

	// 3. Odds limits
	if req.Odds.LessThan(a.minOdds) {
		reason := fmt.Sprintf("odds too low: minimum is %s", a.minOdds.String())
		logger.Warn("Order rejected", "reason", reason)
		return &workflow.ValidationResult{
			Valid:  false,
			Reason: reason,
		}, nil
	}

	if req.Odds.GreaterThan(a.maxOdds) {
		reason := fmt.Sprintf("odds too high: maximum is %s", a.maxOdds.String())
		logger.Warn("Order rejected", "reason", reason)
		return &workflow.ValidationResult{
			Valid:  false,
			Reason: reason,
		}, nil
	}

	// 4. Market validation (TODO: Call data-normalizer to check if market exists and is open)
	// For now, we'll assume the market is valid
	// In production, this would make a gRPC call to data-normalizer-service:
	//
	// marketResp, err := a.dataNormalizerClient.GetMarket(ctx, &datanormalizerpb.GetMarketRequest{
	//     EventId:  req.EventID,
	//     MarketId: req.MarketID,
	// })
	// if err != nil || !marketResp.IsOpen {
	//     return &workflow.ValidationResult{
	//         Valid:  false,
	//         Reason: "market is closed or does not exist",
	//     }, nil
	// }

	// 5. Risk checks (TODO: Call risk-analyzer for fraud detection)
	// For now, we'll assume the order passes risk checks
	// In production, this would make a gRPC call to risk-analyzer-service:
	//
	// riskResp, err := a.riskAnalyzerClient.AnalyzeOrder(ctx, &riskpb.AnalyzeOrderRequest{
	//     UserId:  req.UserID.String(),
	//     Stake:   req.Stake.String(),
	//     EventId: req.EventID,
	// })
	// if err != nil || riskResp.RiskScore > riskThreshold {
	//     return &workflow.ValidationResult{
	//         Valid:  false,
	//         Reason: "order flagged by risk analysis",
	//     }, nil
	// }

	logger.Info("Order validation passed")

	return &workflow.ValidationResult{
		Valid:  true,
		Reason: "",
	}, nil
}
