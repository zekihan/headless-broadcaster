package web

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCheckError_NoError(t *testing.T) {
	rec := httptest.NewRecorder()
	if CheckError(t.Context(), rec, nil, http.StatusBadRequest) {
		t.Error("expected false when err is nil")
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if rec.Body.Len() != 0 {
		t.Errorf("expected no writes, got %q", rec.Body.String())
	}
}

var errFail = errors.New("fail")

func TestCheckError_WithError(t *testing.T) {
	rec := httptest.NewRecorder()
	if !CheckError(t.Context(), rec, errFail, 418, "msg1", "msg2") {
		t.Error("expected true when err occurs")
	}

	if rec.Code != 418 {
		t.Errorf("expected status 418, got %d", rec.Code)
	}

	if rec.Body.String() != "fail\nmsg1\nmsg2\n" {
		t.Errorf("expected body %q, got %q", "fail\nmsg1\nmsg2\n", rec.Body.String())
	}
}

var errBadWrite = errors.New("bad write")

// badWriter Simulates Write failures.
type badWriter struct{ *httptest.ResponseRecorder }

func (b *badWriter) Write(p []byte) (int, error) {
	if strings.Contains(string(p), "trigger error") {
		return 0, errBadWrite
	}

	n, err := b.ResponseRecorder.Write(p)
	if err != nil {
		return 0, fmt.Errorf("failed to write bytes: %w", err)
	}

	return n, nil
}

func (b *badWriter) WriteString(s string) (int, error) {
	if strings.Contains(s, "trigger error") {
		return 0, errBadWrite
	}

	n, err := b.ResponseRecorder.WriteString(s)
	if err != nil {
		return 0, fmt.Errorf("failed to write string: %w", err)
	}

	return n, nil
}

func TestWrite_Positive(t *testing.T) {
	rec := httptest.NewRecorder()
	if Write(t.Context(), rec, []byte("hello")) {
		t.Error("expected false on successful write")
	}

	if rec.Code != http.StatusOK || rec.Body.String() != "hello" {
		t.Errorf("got code %d body %q", rec.Code, rec.Body.String())
	}
}

func TestWrite_Error(t *testing.T) {
	bad := &badWriter{httptest.NewRecorder()}
	if !Write(t.Context(), bad, []byte("trigger error")) {
		t.Error("expected true on write error")
	}

	if bad.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", bad.Code)
	}

	if bad.Body.String() != "bad write\nfailed to write string\n" {
		t.Errorf("expected %q, got %q", "bad write\nfailed to write string\n", bad.Body.String())
	}
}

func TestWriteString_Positive(t *testing.T) {
	rec := httptest.NewRecorder()
	if WriteString(t.Context(), rec, "world") {
		t.Error("expected false on successful writeString")
	}

	if rec.Code != http.StatusOK || rec.Body.String() != "world" {
		t.Errorf("got code %d body %q", rec.Code, rec.Body.String())
	}
}

func TestWriteString_Error(t *testing.T) {
	bad := &badWriter{httptest.NewRecorder()}
	if !WriteString(t.Context(), bad, "trigger error") {
		t.Error("expected true on writeString error")
	}

	if bad.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", bad.Code)
	}

	if bad.Body.String() != "bad write\nfailed to write string\n" {
		t.Errorf("expected %q, got %q", "bad write\nfailed to write string\n", bad.Body.String())
	}
}
