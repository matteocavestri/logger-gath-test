// Package logger initializes and manages structured logging for the application.
//
// This package provides a unified and high-performance logging abstraction built on top of
// Uber’s zap library. It supports both human-readable console logs (for development)
// and JSON-formatted structured logs (for production), which can be directly ingested
// by observability backends such as Loki or Elasticsearch.
//
// The logger is environment-aware, with configurable log levels and output encoders,
// and supports both global and contextual loggers for flexible usage across modules.
//
// Key features:
//   - Fast, structured, leveled logging using zap.
//   - JSON output in production for seamless integration with Loki and other log pipelines.
//   - Colorized console output in development for easier debugging.
//   - Global singleton logger for simple application-wide access.
//   - Contextual field injection for structured log enrichment.
//
// Example usage:
//
//	func main() {
//	    cfg := logger.FromEnv()
//	    if err := logger.InitGlobal(cfg); err != nil {
//	        panic(fmt.Sprintf("failed to initialize logger: %v", err))
//	    }
//	    defer logger.Get().Sync()
//
//	    log := logger.Get()
//	    log.Info("application started", zap.String("version", "1.0.0"))
//	}
//
// Production recommendations:
//   - Always initialize the global logger early in application startup.
//   - In production, prefer JSON encoding for structured log ingestion.
//   - Call `Sync()` before process exit to flush any buffered log entries.
package logger

import (
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger wraps zap.Logger to provide application-specific structured logging functionality.
//
// It supports contextual enrichment via `WithContext()` and integrates with the
// global logger pattern used throughout the application.
type Logger struct {
	*zap.Logger
}

// LogLevel represents the verbosity level for the logger.
type LogLevel string

const (
	// LevelDebug enables detailed debug and above level logging.
	LevelDebug LogLevel = "DEBUG"
	// LevelInfo enables informational and above level logging (default).
	LevelInfo LogLevel = "INFO"
	// LevelWarn enables warning and above level logging.
	LevelWarn LogLevel = "WARN"
	// LevelError enables error level logging only.
	LevelError LogLevel = "ERROR"
)

var (
	// globalLogger is the shared singleton logger instance for the application.
	globalLogger *Logger
)

// Config defines the configuration parameters for the logger.
//
// Environment can be either "development" or "production", which determines the encoder type
// and default output behavior.
type Config struct {
	Level       LogLevel
	Environment string // "development" or "production"
	ServiceName string // Service identifier for log enrichment
}

// New creates a new logger instance according to the given configuration.
//
// In production mode, logs are formatted as structured JSON suitable for ingestion by Loki,
// FluentBit, or Elasticsearch. In development mode, logs use a colorized console encoder.
//
// Example:
//
//	logger, err := logger.New(logger.Config{
//	    Level:       logger.LevelDebug,
//	    Environment: "production",
//	    ServiceName: "api-service",
//	})
//	if err != nil {
//	    panic(err)
//	}
func New(cfg Config) (*Logger, error) {
	var zapLevel zapcore.Level

	// Map custom log levels to zap internal levels
	switch strings.ToUpper(string(cfg.Level)) {
	case string(LevelDebug):
		zapLevel = zapcore.DebugLevel
	case string(LevelInfo):
		zapLevel = zapcore.InfoLevel
	case string(LevelWarn):
		zapLevel = zapcore.WarnLevel
	case string(LevelError):
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	var zapConfig zap.Config
	if cfg.Environment == "production" {
		zapConfig = zap.Config{
			Level:            zap.NewAtomicLevelAt(zapLevel),
			Development:      false,
			Encoding:         "json",
			EncoderConfig:    productionEncoderConfig(),
			OutputPaths:      []string{"stdout"},
			ErrorOutputPaths: []string{"stderr"},
		}
	} else {
		zapConfig = zap.Config{
			Level:            zap.NewAtomicLevelAt(zapLevel),
			Development:      true,
			Encoding:         "console",
			EncoderConfig:    developmentEncoderConfig(),
			OutputPaths:      []string{"stdout"},
			ErrorOutputPaths: []string{"stderr"},
		}
	}

	zapLogger, err := zapConfig.Build(
		zap.AddCallerSkip(1),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build logger: %w", err)
	}

	zapLogger = zapLogger.With(
		zap.String("service", cfg.ServiceName),
		zap.String("environment", cfg.Environment),
	)

	return &Logger{Logger: zapLogger}, nil
}

// productionEncoderConfig defines the encoder settings for production JSON logs.
//
// The output schema is compatible with Loki and other structured logging systems.
func productionEncoderConfig() zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
}

// developmentEncoderConfig defines the encoder settings for development console logs.
//
// The output is colorized and human-readable for local debugging convenience.
func developmentEncoderConfig() zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		TimeKey:        "T",
		LevelKey:       "L",
		NameKey:        "N",
		CallerKey:      "C",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "M",
		StacktraceKey:  "S",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
}

// InitGlobal initializes the global singleton logger.
//
// This should be called during application startup to make the logger globally accessible.
// It replaces any existing global logger instance.
func InitGlobal(cfg Config) error {
	logger, err := New(cfg)
	if err != nil {
		return err
	}
	globalLogger = logger
	return nil
}

// Get retrieves the global logger instance.
//
// If no global logger is initialized, it automatically creates a development-mode logger
// with default parameters. This ensures logging always works even in early initialization stages.
func Get() *Logger {
	if globalLogger == nil {
		cfg := Config{
			Level:       LevelInfo,
			Environment: "development",
			ServiceName: "gath-stack",
		}
		logger, _ := New(cfg)
		globalLogger = logger
	}
	return globalLogger
}

// WithContext returns a derived logger enriched with additional structured fields.
//
// Example:
//
//	log := logger.Get().WithContext(zap.String("user_id", "abc123"))
//	log.Info("User login succeeded")
func (l *Logger) WithContext(fields ...zap.Field) *Logger {
	return &Logger{Logger: l.With(fields...)}
}

// FromEnv builds a logger configuration using environment variables.
//
// Supported variables:
//   - LOG_LEVEL: sets log level (DEBUG, INFO, WARN, ERROR)
//   - APP_ENV: defines environment ("development" or "production")
//   - APP_NAME: sets the service name field
func FromEnv() Config {
	return Config{
		Level:       LogLevel(getEnv("LOG_LEVEL", "INFO")),
		Environment: getEnv("APP_ENV", "development"),
		ServiceName: getEnv("APP_NAME", "gath-stack"),
	}
}

// getEnv retrieves an environment variable or returns a default value.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Sync flushes any buffered log entries to the underlying writer.
//
// This should be deferred before program exit to avoid data loss.
func (l *Logger) Sync() error {
	return l.Logger.Sync()
}

// Debug logs a message at the DEBUG level using the global logger.
func Debug(msg string, fields ...zap.Field) {
	Get().Debug(msg, fields...)
}

// Info logs a message at the INFO level using the global logger.
func Info(msg string, fields ...zap.Field) {
	Get().Info(msg, fields...)
}

// Warn logs a message at the WARN level using the global logger.
func Warn(msg string, fields ...zap.Field) {
	Get().Warn(msg, fields...)
}

// Error logs a message at the ERROR level using the global logger.
func Error(msg string, fields ...zap.Field) {
	Get().Error(msg, fields...)
}

// Fatal logs a message at the FATAL level and terminates the application.
//
// Use this sparingly—prefer returning errors whenever possible.
func Fatal(msg string, fields ...zap.Field) {
	Get().Fatal(msg, fields...)
}

// WithFields creates a derived logger with pre-attached structured fields using the global logger.
//
// Example:
//
//	log := logger.WithFields(zap.String("component", "auth"))
//	log.Info("Authentication service started")
func WithFields(fields ...zap.Field) *Logger {
	return Get().WithContext(fields...)
}
