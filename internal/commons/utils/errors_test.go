package utils

import (
	"errors"
	"strings"
	"testing"
)

var errOriginal = errors.New("original")

func TestWrapErrors_NoMessages(t *testing.T) {
	wrapped := WrapErrors(errOriginal)
	if !errors.Is(wrapped, errOriginal) {
		t.Errorf("expected wrapped to be original, got %v", wrapped)
	}
}

func TestWrapErrors_WithMessages(t *testing.T) {
	wrapped := WrapErrors(errOriginal, "msg1", "msg2")
	if wrapped == nil {
		t.Fatal("expected wrapped error, got nil")
	}

	errStr := wrapped.Error()
	if !strings.Contains(errStr, "original") ||
		!strings.Contains(errStr, "msg1") ||
		!strings.Contains(errStr, "msg2") {
		t.Errorf("expected Error() to contain all, got %q", errStr)
	}
}
