package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the order-validator service
type Config struct {
	Server   ServerConfig
	Temporal TemporalConfig
	Services ServicesConfig
	Logger   LoggerConfig
	Metrics  MetricsConfig
}

// ServerConfig holds server configuration
type ServerConfig struct {
	GRPCPort        int
	HTTPPort        int
	MetricsPort     int
	ShutdownTimeout time.Duration
}

// TemporalConfig holds Temporal configuration
type TemporalConfig struct {
	ServerAddress string
	Namespace     string
	TaskQueue     string
	MaxConcurrent int
}

// ServicesConfig holds downstream service addresses
type ServicesConfig struct {
	WalletServiceAddr        string
	OrderBookServiceAddr     string
	DataNormalizerServiceAddr string
	RiskAnalyzerServiceAddr  string
}

// LoggerConfig holds logger configuration
type LoggerConfig struct {
	Level      string
	Format     string
	TimeFormat string
}

// MetricsConfig holds metrics configuration
type MetricsConfig struct {
	Enabled bool
	Path    string
}

// Load loads configuration from file and environment variables
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set default values
	setDefaults(v)

	// Read config file
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath("./config")
		v.AddConfigPath(".")
	}

	// Read config file (optional)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("read config file: %w", err)
		}
	}

	// Override with environment variables
	v.SetEnvPrefix("ORDER_VALIDATOR")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Unmarshal config
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	// Validate config
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.grpcport", 8090)
	v.SetDefault("server.httpport", 8091)
	v.SetDefault("server.metricsport", 9091)
	v.SetDefault("server.shutdowntimeout", "30s")

	// Temporal defaults
	v.SetDefault("temporal.serveraddress", "temporal-frontend.workflows.svc.cluster.local:7233")
	v.SetDefault("temporal.namespace", "tam-platform")
	v.SetDefault("temporal.taskqueue", "order-validator")
	v.SetDefault("temporal.maxconcurrent", 10)

	// Services defaults
	v.SetDefault("services.walletserviceaddr", "wallet-service.default.svc.cluster.local:8080")
	v.SetDefault("services.orderbookserviceaddr", "order-book.default.svc.cluster.local:8082")
	v.SetDefault("services.datanormalizerserviceaddr", "data-normalizer.default.svc.cluster.local:8083")
	v.SetDefault("services.riskanalyzerserviceaddr", "risk-analyzer.default.svc.cluster.local:8084")

	// Logger defaults
	v.SetDefault("logger.level", "info")
	v.SetDefault("logger.format", "json")
	v.SetDefault("logger.timeformat", "rfc3339")

	// Metrics defaults
	v.SetDefault("metrics.enabled", true)
	v.SetDefault("metrics.path", "/metrics")
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate server config
	if c.Server.GRPCPort <= 0 || c.Server.GRPCPort > 65535 {
		return fmt.Errorf("invalid gRPC port: %d", c.Server.GRPCPort)
	}
	if c.Server.HTTPPort <= 0 || c.Server.HTTPPort > 65535 {
		return fmt.Errorf("invalid HTTP port: %d", c.Server.HTTPPort)
	}

	// Validate Temporal config
	if c.Temporal.ServerAddress == "" {
		return fmt.Errorf("Temporal server address is required")
	}
	if c.Temporal.Namespace == "" {
		return fmt.Errorf("Temporal namespace is required")
	}
	if c.Temporal.TaskQueue == "" {
		return fmt.Errorf("Temporal task queue is required")
	}

	// Validate service addresses
	if c.Services.WalletServiceAddr == "" {
		return fmt.Errorf("wallet service address is required")
	}

	// Validate logger config
	validLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true, "fatal": true, "panic": true,
	}
	if !validLevels[strings.ToLower(c.Logger.Level)] {
		return fmt.Errorf("invalid log level: %s", c.Logger.Level)
	}

	return nil
}
