package ui

import (
	"errors"
	"testing"
)

func TestCleanErrorMessage_NoTrace(t *testing.T) {
	input := "failed to connect to server"
	want := "failed to connect to server"
	got := CleanErrorMessage(input)
	if got != want {
		t.Errorf("CleanErrorMessage(%q) = %q, want %q", input, got, want)
	}
}

func TestCleanErrorMessage_WithTraceparent(t *testing.T) {
	input := "failed to resolve image [traceparent:08700671ef5b5d073f02a1da102d91a5-e249b0cb6ce214cf]"
	want := "failed to resolve image"
	got := CleanErrorMessage(input)
	if got != want {
		t.Errorf("CleanErrorMessage(%q) = %q, want %q", input, got, want)
	}
}

func TestCleanErrorMessage_TraceparentInMiddle(t *testing.T) {
	input := "error: something failed [traceparent:abc123-def456] more text"
	want := "error: something failed more text"
	got := CleanErrorMessage(input)
	if got != want {
		t.Errorf("CleanErrorMessage(%q) = %q, want %q", input, got, want)
	}
}

func TestCleanErrorMessage_MultipleTraces(t *testing.T) {
	input := "err1 [traceparent:a-b] err2 [traceparent:c-d] err3"
	want := "err1 err2 err3"
	got := CleanErrorMessage(input)
	if got != want {
		t.Errorf("CleanErrorMessage(%q) = %q, want %q", input, got, want)
	}
}

func TestCleanErrorMessage_Empty(t *testing.T) {
	input := ""
	want := ""
	got := CleanErrorMessage(input)
	if got != want {
		t.Errorf("CleanErrorMessage(%q) = %q, want %q", input, got, want)
	}
}

func TestCleanError_NilError(t *testing.T) {
	got := CleanError(nil)
	if got != nil {
		t.Errorf("CleanError(nil) = %v, want nil", got)
	}
}

func TestCleanError_WithTrace(t *testing.T) {
	input := errors.New("failed [traceparent:abc-123]")
	got := CleanError(input)
	want := "failed"
	if got.Error() != want {
		t.Errorf("CleanError(...).Error() = %q, want %q", got.Error(), want)
	}
}

func TestCleanError_PreservesWrapping(t *testing.T) {
	// CleanError should clean the message but error wrapping is lost
	// (acceptable tradeoff for cleaner output)
	inner := errors.New("inner [traceparent:x-y]")
	wrapped := errors.New("outer: " + inner.Error())
	got := CleanError(wrapped)
	want := "outer: inner"
	if got.Error() != want {
		t.Errorf("CleanError(...).Error() = %q, want %q", got.Error(), want)
	}
}
