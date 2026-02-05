package orchestrator

// RetryConfig tracks retry state for step execution.
type RetryConfig struct {
	MaxRetries int
	attempts   int
}

// newRetryConfig creates a new retry configuration.
func newRetryConfig(maxRetries int) *RetryConfig {
	return &RetryConfig{
		MaxRetries: maxRetries,
		attempts:   0,
	}
}

// ShouldRetry returns true if more attempts are available.
func (r *RetryConfig) ShouldRetry() bool {
	if r.MaxRetries == 0 {
		return false
	}
	return r.attempts <= r.MaxRetries
}

// RecordAttempt increments the attempt counter.
func (r *RetryConfig) RecordAttempt() {
	r.attempts++
}

// CurrentAttempt returns the current attempt number (0-indexed).
func (r *RetryConfig) CurrentAttempt() int {
	return r.attempts
}

// ExecuteWithRetry executes a function with retry support.
// Returns nil on success, or the last error if all retries are exhausted.
func ExecuteWithRetry(maxRetries int, fn func() error) error {
	var lastErr error

	// 1 initial attempt + maxRetries retries
	for attempt := 0; attempt <= maxRetries; attempt++ {
		lastErr = fn()
		if lastErr == nil {
			return nil
		}
	}

	return lastErr
}
