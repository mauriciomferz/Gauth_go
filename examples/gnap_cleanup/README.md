# GNAP Cleanup Manager Example

This example demonstrates how to integrate the GNAP CleanupManager into a production application.

## Features

- Environment-based cleanup interval configuration
- Graceful shutdown handling
- Periodic metrics reporting
- Activity simulation for testing

## Running the Example

```bash
cd examples/gnap_cleanup

# Development mode (5 min interval)
ENVIRONMENT=development go run main.go

# Production mode (15 min interval)
ENVIRONMENT=production go run main.go

# Default (5 min interval)
go run main.go
```

## Configuration

The cleanup interval is automatically configured based on the `ENVIRONMENT` variable:

| Environment | Interval | Use Case |
|-------------|----------|----------|
| development | 5 min | Fast feedback during development |
| staging | 10 min | Balanced performance testing |
| production | 15 min | Production optimization |
| (default) | 5 min | High-volume / safe default |

## Output

The example will:
1. Start the cleanup manager
2. Create sample grants and tokens (some expired)
3. Report cleanup statistics every 30 seconds
4. Run until interrupted (Ctrl+C)
5. Display final statistics on shutdown

## Production Integration

To integrate this into your application:

```go
import "github.com/mauriciomferz/AgentAuth/pkg/gnap"

// In your main function or server setup
cleanup := gnap.NewCleanupManager(grantStore, tokenStore, 10*time.Minute)
cleanup.Start(ctx)
defer cleanup.Stop()
```

## Monitoring

Access cleanup statistics at any time:

```go
stats := cleanup.Stats()
fmt.Printf("Cleaned: %d grants, %d tokens\n", 
    stats.TotalGrantsCleaned, 
    stats.TotalTokensCleaned)
```

## See Also

- [MAINTENANCE.md](../../docs/guides/MAINTENANCE.md) - Full maintenance guide
- [pkg/gnap/cleanup_manager.go](../../pkg/gnap/cleanup_manager.go) - Implementation
- [pkg/gnap/cleanup_manager_test.go](../../pkg/gnap/cleanup_manager_test.go) - Tests
