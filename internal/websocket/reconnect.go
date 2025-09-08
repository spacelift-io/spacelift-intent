package websocket

import (
	"context"
	"fmt"
	"math"
	"time"
)

// RetryConfig holds configuration for reconnection attempts
type RetryConfig struct {
	MaxAttempts    int
	InitialDelay   time.Duration
	ResetPeriod    time.Duration
	UseExponential bool
}

// DefaultRetryConfig returns sensible default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:    3,
		InitialDelay:   5 * time.Second,
		ResetPeriod:    -1, // No reset by default
		UseExponential: true,
	}
}

// RetryOperation performs an operation with exponential backoff retry logic
func RetryOperation(ctx context.Context, config RetryConfig, log func(format string, v ...any), operation func() error) error {
	for i := 0; i <= config.MaxAttempts; {
		start := time.Now()
		err := operation()
		if err == nil {
			return nil
		}
		log("websocket disconnected")

		duration := time.Since(start)
		if config.ResetPeriod > 0 && duration > config.ResetPeriod {
			// Reset attempt counter if operation ran long enough
			i = 0
			log("websocket was connected for %d seconds, reseting reconnection counter", int(duration.Seconds()))
		}

		// Don't sleep on the last attempt
		if i != config.MaxAttempts {
			var delay time.Duration
			if config.UseExponential {
				delay = time.Duration(math.Pow(2, float64(i))) * config.InitialDelay
			} else {
				delay = config.InitialDelay
			}
			i++
			log("reconnect attempt %d of %d in %d seconds", i, config.MaxAttempts, int(delay.Seconds()))

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				continue
			}
		}
		return fmt.Errorf("reconnect failed after %d attempts: %w", config.MaxAttempts, err)
	}
	return nil
}
