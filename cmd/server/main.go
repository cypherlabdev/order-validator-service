package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"google.golang.org/grpc"

	orderv1 "github.com/cypherlabdev/cypherlabdev-protos/gen/go/order/v1"
	"github.com/cypherlabdev/order-validator-service/internal/activity"
	"github.com/cypherlabdev/order-validator-service/internal/config"
	grpcHandler "github.com/cypherlabdev/order-validator-service/internal/handler/grpc"
	"github.com/cypherlabdev/order-validator-service/internal/workflow"
)

func main() {
	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load configuration")
	}

	// Initialize logger
	logger := initLogger(cfg.Logger)
	logger.Info().Msg("order-validator-service starting")

	// Initialize Temporal client
	temporalClient, err := client.Dial(client.Options{
		HostPort:  cfg.Temporal.ServerAddress,
		Namespace: cfg.Temporal.Namespace,
		Logger:    newTemporalLogger(logger),
	})
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create Temporal client")
	}
	defer temporalClient.Close()

	logger.Info().
		Str("address", cfg.Temporal.ServerAddress).
		Str("namespace", cfg.Temporal.Namespace).
		Msg("connected to Temporal server")

	// Initialize activities
	walletActivities, err := activity.NewWalletActivities(cfg.Services.WalletServiceAddr, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize wallet activities")
	}

	validationActivities := activity.NewValidationActivities(logger)

	orderBookActivities, err := activity.NewOrderBookActivities(cfg.Services.OrderBookServiceAddr, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize order-book activities")
	}

	// Create Temporal worker
	w := worker.New(temporalClient, cfg.Temporal.TaskQueue, worker.Options{
		MaxConcurrentActivityExecutionSize: cfg.Temporal.MaxConcurrent,
	})

	// Register workflows
	w.RegisterWorkflow(workflow.PlaceOrderWorkflow)

	// Register activities
	w.RegisterActivity(validationActivities.ValidateOrder)
	w.RegisterActivity(walletActivities.ReserveFunds)
	w.RegisterActivity(walletActivities.CommitReservation)
	w.RegisterActivity(walletActivities.CancelReservation)
	w.RegisterActivity(orderBookActivities.PlaceOrderInBook)
	w.RegisterActivity(orderBookActivities.CancelOrder)

	logger.Info().Str("task_queue", cfg.Temporal.TaskQueue).Msg("Temporal worker registered")

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start servers
	errChan := make(chan error, 3)

	// Start Temporal worker
	go func() {
		logger.Info().Msg("starting Temporal worker")
		if err := w.Run(worker.InterruptCh()); err != nil {
			errChan <- fmt.Errorf("worker error: %w", err)
		}
	}()

	// Start gRPC server
	go func() {
		errChan <- startGRPCServer(cfg, temporalClient, logger)
	}()

	// Start metrics server
	if cfg.Metrics.Enabled {
		go func() {
			errChan <- startMetricsServer(cfg, logger)
		}()
	}

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errChan:
		logger.Error().Err(err).Msg("server error")
		os.Exit(1)
	case sig := <-sigChan:
		logger.Info().Str("signal", sig.String()).Msg("received shutdown signal")
	}

	// Graceful shutdown
	logger.Info().Msg("shutting down gracefully")
	cancel()

	// Stop Temporal worker
	w.Stop()

	logger.Info().Msg("shutdown complete")
}

// startGRPCServer starts the gRPC server
func startGRPCServer(cfg *config.Config, temporalClient client.Client, logger zerolog.Logger) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.GRPCPort))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	grpcServer := grpc.NewServer()

	// Register services
	orderHandler := grpcHandler.NewOrderHandler(temporalClient, logger)
	orderv1.RegisterValidatorServiceServer(grpcServer, orderHandler)

	logger.Info().Int("port", cfg.Server.GRPCPort).Msg("gRPC server listening")

	if err := grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}

	return nil
}

// startMetricsServer starts the Prometheus metrics server
func startMetricsServer(cfg *config.Config, logger zerolog.Logger) error {
	mux := http.NewServeMux()
	mux.Handle(cfg.Metrics.Path, promhttp.Handler())

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.MetricsPort),
		Handler: mux,
	}

	logger.Info().
		Int("port", cfg.Server.MetricsPort).
		Str("path", cfg.Metrics.Path).
		Msg("metrics server listening")

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("serve metrics: %w", err)
	}

	return nil
}

// initLogger initializes the logger
func initLogger(cfg config.LoggerConfig) zerolog.Logger {
	level, err := zerolog.ParseLevel(cfg.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}

	zerolog.SetGlobalLevel(level)

	logger := zerolog.New(os.Stdout).
		With().
		Timestamp().
		Str("service", "order-validator-service").
		Caller().
		Logger()

	log.Logger = logger
	return logger
}

// temporalLogger adapts zerolog to Temporal's logger interface
type temporalLogger struct {
	logger zerolog.Logger
}

func newTemporalLogger(logger zerolog.Logger) *temporalLogger {
	return &temporalLogger{
		logger: logger.With().Str("component", "temporal_sdk").Logger(),
	}
}

func (l *temporalLogger) Debug(msg string, keyvals ...interface{}) {
	l.logger.Debug().Fields(keyvals).Msg(msg)
}

func (l *temporalLogger) Info(msg string, keyvals ...interface{}) {
	l.logger.Info().Fields(keyvals).Msg(msg)
}

func (l *temporalLogger) Warn(msg string, keyvals ...interface{}) {
	l.logger.Warn().Fields(keyvals).Msg(msg)
}

func (l *temporalLogger) Error(msg string, keyvals ...interface{}) {
	l.logger.Error().Fields(keyvals).Msg(msg)
}
