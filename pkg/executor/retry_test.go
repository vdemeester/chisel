package executor

import (
	"errors"
	"testing"
)

func TestRetryConfig_Defaults(t *testing.T) {
	config := newRetryConfig(0)

	if config.MaxRetries != 0 {
		t.Errorf("MaxRetries = %d, want 0", config.MaxRetries)
	}
	if config.ShouldRetry() {
		t.Error("ShouldRetry() = true, want false with 0 retries")
	}
}

func TestRetryConfig_WithRetries(t *testing.T) {
	config := newRetryConfig(3)

	if config.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", config.MaxRetries)
	}
	if !config.ShouldRetry() {
		t.Error("ShouldRetry() = false, want true with 3 retries")
	}
}

func TestRetryConfig_Attempts(t *testing.T) {
	config := newRetryConfig(2)

	// First attempt
	if !config.ShouldRetry() {
		t.Error("attempt 0: ShouldRetry() = false, want true")
	}

	// Record failure, second attempt
	config.RecordAttempt()
	if !config.ShouldRetry() {
		t.Error("attempt 1: ShouldRetry() = false, want true")
	}

	// Record failure, third attempt (last retry)
	config.RecordAttempt()
	if !config.ShouldRetry() {
		t.Error("attempt 2: ShouldRetry() = false, want true")
	}

	// Record failure, no more retries
	config.RecordAttempt()
	if config.ShouldRetry() {
		t.Error("attempt 3: ShouldRetry() = true, want false (exhausted)")
	}
}

func TestRetryConfig_CurrentAttempt(t *testing.T) {
	config := newRetryConfig(2)

	if config.CurrentAttempt() != 0 {
		t.Errorf("initial CurrentAttempt() = %d, want 0", config.CurrentAttempt())
	}

	config.RecordAttempt()
	if config.CurrentAttempt() != 1 {
		t.Errorf("after 1 attempt: CurrentAttempt() = %d, want 1", config.CurrentAttempt())
	}

	config.RecordAttempt()
	if config.CurrentAttempt() != 2 {
		t.Errorf("after 2 attempts: CurrentAttempt() = %d, want 2", config.CurrentAttempt())
	}
}

func TestExecuteWithRetry_NoRetries(t *testing.T) {
	attempts := 0
	fn := func() error {
		attempts++
		return errors.New("always fails")
	}

	err := executeWithRetry(0, fn)
	if err == nil {
		t.Error("expected error, got nil")
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1", attempts)
	}
}

func TestExecuteWithRetry_Success(t *testing.T) {
	attempts := 0
	fn := func() error {
		attempts++
		return nil
	}

	err := executeWithRetry(3, fn)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1", attempts)
	}
}

func TestExecuteWithRetry_SuccessAfterRetries(t *testing.T) {
	attempts := 0
	fn := func() error {
		attempts++
		if attempts < 3 {
			return errors.New("transient failure")
		}
		return nil
	}

	err := executeWithRetry(5, fn)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func TestExecuteWithRetry_ExhaustedRetries(t *testing.T) {
	attempts := 0
	fn := func() error {
		attempts++
		return errors.New("always fails")
	}

	err := executeWithRetry(2, fn)
	if err == nil {
		t.Error("expected error, got nil")
	}
	// 1 initial + 2 retries = 3 attempts
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}
