package utils

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
)

const MaskedString = "****"

var ErrNegativeLength = errors.New("length cannot be negative")

func RandString(nByte int) (string, error) {
	if nByte < 0 {
		return "", ErrNegativeLength
	}

	randomBytes := make([]byte, nByte)
	if nByte == 0 {
		return "", nil // Return empty string for zero length
	}

	_, err := io.ReadFull(rand.Reader, randomBytes)
	if err != nil {
		return "", fmt.Errorf("failed to read random bytes: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(randomBytes), nil
}

func Contains[T comparable](commons []T, item T) bool {
	return slices.Contains(commons, item)
}

func Filter[T any](list []T, predicate func(T) bool) []T {
	var result []T

	for _, item := range list {
		if predicate(item) {
			result = append(result, item)
		}
	}

	return result
}

func RemoveEmptyItems(list []string) []string {
	var result []string

	for _, item := range list {
		if strings.TrimSpace(item) != "" {
			result = append(result, item)
		}
	}

	return result
}

const defaultUnmaskedStringLength = 4

func MaskString(secret string) string {
	if secret == "" {
		return ""
	}

	if len(secret) <= defaultUnmaskedStringLength {
		return MaskedString
	}

	return secret[:defaultUnmaskedStringLength] + MaskedString
}

// PointerTo returns a pointer to the given value.
func PointerTo[T any](value T) *T {
	return new(value)
}
