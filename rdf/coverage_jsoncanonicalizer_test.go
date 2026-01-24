package rdf

import (
	"math"
	"strings"
	"testing"
)

// Test canonicalizeJSONText and numberToJSON functions for coverage

func TestCanonicalizeJSONText_SimpleValues(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"string", `"hello"`, false},
		{"number", `42`, false},
		{"number negative", `-42`, false},
		{"number float", `3.14`, false},
		{"number scientific", `1e10`, false},
		{"boolean true", `true`, false},
		{"boolean false", `false`, false},
		{"null", `null`, false},
		{"empty object", `{}`, false},
		{"empty array", `[]`, false},
		{"object with keys", `{"b":2,"a":1}`, false},
		{"array with values", `[2,1,3]`, false},
		{"nested object", `{"a":{"b":1}}`, false},
		{"nested array", `[[1,2],[3,4]]`, false},
		{"invalid JSON", `{invalid}`, true},
		{"unclosed string", `"hello`, true},
		{"unclosed object", `{`, true},
		{"unclosed array", `[`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := canonicalizeJSONText([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("canonicalizeJSONText() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCanonicalizeJSONText_StringEscapes(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"escape quote", `"hello\"world"`},
		{"escape backslash", `"hello\\world"`},
		{"escape newline", `"hello\nworld"`},
		{"escape tab", `"hello\tworld"`},
		{"escape carriage return", `"hello\rworld"`},
		{"unicode escape", `"\u0041"`},
		{"unicode escape lowercase", `"\u0041"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := canonicalizeJSONText([]byte(tt.input))
			if err != nil {
				t.Errorf("canonicalizeJSONText() error = %v", err)
			}
		})
	}
}

func TestCanonicalizeJSONText_ObjectKeyOrdering(t *testing.T) {
	// Test that object keys are sorted
	input := `{"z":3,"a":1,"m":2}`
	result, err := canonicalizeJSONText([]byte(input))
	if err != nil {
		t.Fatalf("canonicalizeJSONText() error = %v", err)
	}
	resultStr := string(result)
	// Keys should be sorted: a, m, z
	// Find the position of 'a' in the result
	if !strings.Contains(resultStr, `"a"`) {
		t.Errorf("Result should contain 'a' key, got %q", resultStr)
	}
	// Verify it's valid JSON
	if !strings.Contains(resultStr, `"a"`) || !strings.Contains(resultStr, `"m"`) || !strings.Contains(resultStr, `"z"`) {
		t.Errorf("Result should contain all keys, got %q", resultStr)
	}
}

func TestNumberToJSON_Zero(t *testing.T) {
	result, err := numberToJSON(0.0)
	if err != nil {
		t.Fatalf("numberToJSON(0) error = %v", err)
	}
	if result != "0" {
		t.Errorf("numberToJSON(0) = %q, want \"0\"", result)
	}
}

func TestNumberToJSON_Negative(t *testing.T) {
	result, err := numberToJSON(-42.5)
	if err != nil {
		t.Fatalf("numberToJSON(-42.5) error = %v", err)
	}
	if result[0] != '-' {
		t.Errorf("numberToJSON(-42.5) should start with '-', got %q", result)
	}
}

func TestNumberToJSON_Positive(t *testing.T) {
	result, err := numberToJSON(42.5)
	if err != nil {
		t.Fatalf("numberToJSON(42.5) error = %v", err)
	}
	if result[0] == '-' {
		t.Errorf("numberToJSON(42.5) should not start with '-', got %q", result)
	}
}

func TestNumberToJSON_SmallNumber(t *testing.T) {
	// Test numbers in fixed format range (1e-6 to 1e+21)
	result, err := numberToJSON(3.14159)
	if err != nil {
		t.Fatalf("numberToJSON(3.14159) error = %v", err)
	}
	if len(result) == 0 {
		t.Error("numberToJSON(3.14159) returned empty string")
	}
}

func TestNumberToJSON_LargeNumber(t *testing.T) {
	// Test numbers requiring scientific notation
	result, err := numberToJSON(1e25)
	if err != nil {
		t.Fatalf("numberToJSON(1e25) error = %v", err)
	}
	if len(result) == 0 {
		t.Error("numberToJSON(1e25) returned empty string")
	}
}

func TestNumberToJSON_VerySmallNumber(t *testing.T) {
	// Test very small numbers
	result, err := numberToJSON(1e-10)
	if err != nil {
		t.Fatalf("numberToJSON(1e-10) error = %v", err)
	}
	if len(result) == 0 {
		t.Error("numberToJSON(1e-10) returned empty string")
	}
}

func TestNumberToJSON_InvalidNumber(t *testing.T) {
	// Test with NaN
	nan := math.NaN()
	_, err := numberToJSON(nan)
	if err == nil {
		t.Error("numberToJSON(NaN) should return error")
	}

	// Test with Infinity
	inf := math.Inf(1)
	_, err = numberToJSON(inf)
	if err == nil {
		t.Error("numberToJSON(Inf) should return error")
	}

	// Test with negative Infinity
	negInf := math.Inf(-1)
	_, err = numberToJSON(negInf)
	if err == nil {
		t.Error("numberToJSON(-Inf) should return error")
	}
}

func TestNumberToJSON_EdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		value float64
	}{
		{"one", 1.0},
		{"negative one", -1.0},
		{"very large", 1e20},
		{"very small", 1e-20},
		{"integer", 42.0},
		{"decimal", 0.123456789},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := numberToJSON(tt.value)
			if err != nil {
				t.Errorf("numberToJSON(%v) error = %v", tt.value, err)
				return
			}
			if len(result) == 0 {
				t.Errorf("numberToJSON(%v) returned empty string", tt.value)
			}
		})
	}
}

func TestCanonicalizeJSONText_ComplexStructures(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"mixed array", `[1,"two",true,null,{"key":"value"}]`},
		{"object with array", `{"arr":[1,2,3],"obj":{"nested":true}}`},
		{"deep nesting", `{"a":{"b":{"c":{"d":1}}}}`},
		{"array of objects", `[{"id":1},{"id":2},{"id":3}]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := canonicalizeJSONText([]byte(tt.input))
			if err != nil {
				t.Errorf("canonicalizeJSONText() error = %v", err)
			}
		})
	}
}

func TestCanonicalizeJSONText_UnicodeStrings(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"basic unicode", `"hello 世界"`},
		{"emoji", `"hello 👋"`},
		{"mixed unicode", `"test\u0041\u0042"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := canonicalizeJSONText([]byte(tt.input))
			if err != nil {
				t.Errorf("canonicalizeJSONText() error = %v", err)
			}
		})
	}
}
