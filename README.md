# logger

This package provides a unified, high-performance, and structured logging system for **gath-stack** web applications.  
It is designed to be dropped directly into your project under `internal/logger`, offering a consistent and production-ready logging setup across services.

The package is built on top of **Uber’s [zap](https://github.com/uber-go/zap)** library, providing both structured JSON logging for production environments and colorized console logging for local development.

---

## Overview

The `logger` package abstracts away the complexity of configuring and managing loggers.  
It automatically adapts to the runtime environment and provides both global and contextual logging interfaces.

### Key features

- Fast, structured, leveled logging using `zap`
- JSON output in production for ingestion by log pipelines (Loki, Elasticsearch, etc.)
- Colorized human-readable console output in development
- Global singleton logger for easy application-wide access
- Contextual structured fields for rich, queryable logs
- Environment-aware configuration via environment variables

---

## Installation

Copy this package into your web application under:

```plaintext
internal/logger
```

No external configuration is required; it works out of the box.

---

## Usage

### 1. Initialize the logger at startup

You should initialize the global logger as early as possible in your application, typically in `main()`:

```go
package main

import (
    "fmt"
    "go.uber.org/zap"
    "myapp/internal/logger"
)

func main() {
    cfg := logger.FromEnv()
    if err := logger.InitGlobal(cfg); err != nil {
        panic(fmt.Sprintf("failed to initialize logger: %v", err))
    }
    defer logger.Get().Sync()

    logger.Info("application started", zap.String("version", "1.0.0"))
}
```

---

### 2. Using the global logger

The package provides convenience functions for leveled logging:

```go
logger.Debug("debugging connection", zap.String("endpoint", "/api/v1"))
logger.Info("user authenticated", zap.String("user_id", "abc123"))
logger.Warn("cache miss", zap.String("key", "session_token"))
logger.Error("database query failed", zap.Error(err))
```

Each log entry includes contextual metadata such as the service name and environment.

---

### 3. Using contextual loggers

You can derive new loggers with additional structured fields for contextual enrichment:

```go
log := logger.WithFields(
    zap.String("component", "auth"),
    zap.String("request_id", "req-42a"),
)

log.Info("authentication succeeded", zap.String("user", "alice"))
```

This produces a structured log entry:

```json
{
  "timestamp": "2025-10-16T12:34:56Z",
  "level": "info",
  "message": "authentication succeeded",
  "component": "auth",
  "request_id": "req-42a",
  "user": "alice",
  "service": "gath-stack-todo",
  "environment": "production"
}
```

---

### 4. Configuration via environment variables

The logger reads its configuration from environment variables when using `logger.FromEnv()`:

| Variable    | Description                                      | Default           |
| ----------- | ------------------------------------------------ | ----------------- |
| `LOG_LEVEL` | Logging level (`DEBUG`, `INFO`, `WARN`, `ERROR`) | `INFO`            |
| `APP_ENV`   | Environment (`development` or `production`)      | `development`     |
| `APP_NAME`  | Service name used for log enrichment             | `gath-stack-todo` |

Example:

```bash
export LOG_LEVEL=DEBUG
export APP_ENV=production
export APP_NAME=api-service
```

---

### 5. Flushing logs

Always ensure that buffered logs are flushed before the process exits:

```go
defer logger.Get().Sync()
```

This is especially important in production to prevent loss of pending log entries.

---

## Integration guidelines

* Always initialize the global logger at the start of your application.
* In production environments, prefer JSON output for ingestion by observability pipelines such as Loki or Elasticsearch.
* Avoid using `logger.Fatal` except during startup or non-recoverable failures.
* Use structured fields (`zap.Field`) for all dynamic data instead of string concatenation.
* When integrating across packages, **never create new logger instances directly**; use the global logger or derive contextual loggers with `WithFields`.

---

## Example directory layout

```plaintext
myapp/
├── cmd/
│   └── api/
│       └── main.go
├── internal/
│   ├── logger/
│   │   ├── logger.go
│   │   └── ...
│   └── server/
│       └── handler.go
```

In `handler.go`:

```go
package server

import (
    "go.uber.org/zap"
    "myapp/internal/logger"
)

func HandleRequest() {
    log := logger.WithFields(zap.String("component", "server"))
    log.Info("request processed successfully")
}
```

---

## License

This package is distributed as part of the **gath-stack** internal tooling.
You are free to copy and use it within gath-stack applications or related projects.
