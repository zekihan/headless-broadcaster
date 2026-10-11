package utils

import (
	"go/constant"
	"testing"
)

func TestRandString(t *testing.T) {
	tests := []struct {
		name    string
		nByte   int
		wantErr bool
	}{
		{"ValidLength10", 10, false},
		{"ValidLength0", 0, false},
		{"NegativeLength", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RandString(tt.nByte)
			if (err != nil) != tt.wantErr {
				t.Errorf("RandString() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if !tt.wantErr && got == "" && tt.nByte > 0 {
				t.Errorf("RandString() returned empty string for nByte %d", tt.nByte)
			}

			if !tt.wantErr && len(got) < tt.nByte*4/3 {
				t.Errorf("RandString() got length %d, want %d. String=%s", len(got), tt.nByte, got)
			}
		})
	}
}

func TestContains(t *testing.T) {
	ints := []int{1, 2, 3, 4}
	if !Contains(ints, 3) {
		t.Errorf("Contains failed: expected true for 3 in %v", ints)
	}

	if Contains(ints, 5) {
		t.Errorf("Contains failed: expected false for 5 in %v", ints)
	}

	strs := []string{"a", "b", "c"}
	if !Contains(strs, "b") {
		t.Errorf("Contains failed: expected true for 'b' in %v", strs)
	}

	if Contains(strs, "d") {
		t.Errorf("Contains failed: expected false for 'd' in %v", strs)
	}
}

func TestRemoveEmptyItems(t *testing.T) {
	input := []string{"a", " ", "", "b", "\t", "c", "\n"}
	expected := []string{"a", "b", "c"}

	result := RemoveEmptyItems(input)
	if len(result) != len(expected) {
		t.Fatalf("RemoveEmptyItems failed: expected length %d, got %d", len(expected), len(result))
	}

	for i, v := range expected {
		if result[i] != v {
			t.Errorf("RemoveEmptyItems failed: expected %v, got %v", expected, result)
		}
	}

	emptyInput := []string{"", " ", "\t"}

	result = RemoveEmptyItems(emptyInput)
	if len(result) != 0 {
		t.Errorf("RemoveEmptyItems failed: expected empty slice, got %v", result)
	}
}

func TestMaskString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"a", MaskedString},
		{"ab", MaskedString},
		{"abcd", MaskedString},
		{"abcde", "abcd****"},
		{"abcdefgh", "abcd****"},
		{"1234", MaskedString},
		{"12345", "1234****"},
	}

	for _, tt := range tests {
		got := MaskString(tt.input)
		if got != tt.expected {
			t.Errorf("MaskString(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestFilter_Ints(t *testing.T) {
	input := []int{1, 2, 3, 4, 5, 6}
	result := Filter(input, func(n int) bool { return n%2 == 0 })

	if len(result) != 3 {
		t.Fatalf("expected 3 items, got %d", len(result))
	}

	for _, v := range result {
		if v%2 != 0 {
			t.Errorf("expected even number, got %d", v)
		}
	}
}

func TestFilter_Strings(t *testing.T) {
	input := []string{"apple", "banana", "apricot", "cherry"}
	result := Filter(input, func(s string) bool { return s[0] == 'a' })

	if len(result) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result))
	}

	if result[0] != "apple" || result[1] != "apricot" {
		t.Errorf("unexpected result: %v", result)
	}
}

func TestFilter_Empty(t *testing.T) {
	result := Filter([]int{}, func(n int) bool { return n > 0 })
	if len(result) != 0 {
		t.Errorf("expected empty result, got %v", result)
	}
}

func TestFilter_KeepNone(t *testing.T) {
	result := Filter([]int{1, 2, 3}, func(n int) bool { return false })
	if len(result) != 0 {
		t.Errorf("expected empty result, got %v", result)
	}
}

func TestFilter_KeepAll(t *testing.T) {
	input := []int{1, 2, 3}
	result := Filter(input, func(n int) bool { return true })

	if len(result) != len(input) {
		t.Fatalf("expected %d items, got %d", len(input), len(result))
	}

	for i, v := range result {
		if v != input[i] {
			t.Errorf("result[%d] = %d, want %d", i, v, input[i])
		}
	}
}

func TestPointerTo(t *testing.T) {
	type custom struct {
		Y string
		X int
	}

	tests := []struct {
		input any
		name  string
	}{
		{42, constant.Int.String()},
		{"hello", constant.String.String()},
		{custom{X: 1, Y: "a"}, "Struct"},
		{[]int{1, 2, 3}, "Slice"},
		{(*int)(nil), "NilPointer"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch value := tt.input.(type) {
			case int, string, custom, *int:
				ptr := new(value)
				if *ptr != value {
					t.Errorf("PointerTo(%v) = %v, want pointer to %v", value, ptr, value)
				}
			case []int:
				ptr := new(value)
				if len(*ptr) != len(value) {
					t.Errorf("PointerTo(%v) = %v, want pointer to %v", value, ptr, value)
				}
			}
		})
	}
}
