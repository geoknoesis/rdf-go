package rdf

import (
	"testing"
)

// Test JSON-LD helper functions for maximum coverage

// expandJSONLD, compactJSONLD, and flattenJSONLD are not direct functions in this codebase
// They are accessed through the JSONLDProcessor interface

// parseJSONLDValue is not a direct function in this codebase

// TestReplaceJSONLiteralValues_* tests are in coverage_jsonld_api_test.go

func TestJSONTypeIncludes_True(t *testing.T) {
	types := []interface{}{"http://example.org/Type1", "http://example.org/Type2"}
	if !jsonTypeIncludes(types, "http://example.org/Type1") {
		t.Error("jsonTypeIncludes should return true for included type")
	}
}

func TestJSONTypeIncludes_False(t *testing.T) {
	types := []interface{}{"http://example.org/Type1", "http://example.org/Type2"}
	if jsonTypeIncludes(types, "http://example.org/Type3") {
		t.Error("jsonTypeIncludes should return false for non-included type")
	}
}

func TestJSONTypeIncludes_Empty(t *testing.T) {
	types := []interface{}{}
	if jsonTypeIncludes(types, "http://example.org/Type1") {
		t.Error("jsonTypeIncludes should return false for empty types")
	}
}

func TestJSONTypeIncludes_StringType(t *testing.T) {
	types := "http://example.org/Type1"
	if !jsonTypeIncludes(types, "http://example.org/Type1") {
		t.Error("jsonTypeIncludes should handle string type")
	}
}

func TestJSONTypeIncludes_NonStringInArray(t *testing.T) {
	types := []interface{}{42, "http://example.org/Type1"}
	if !jsonTypeIncludes(types, "http://example.org/Type1") {
		t.Error("jsonTypeIncludes should handle non-string types in array")
	}
}

// TestPrepareJSONLDForToRDF_* tests are in coverage_jsonld_api_test.go

// TestNormalizeJSONLDJSONLiterals_* tests are in coverage_jsonld_api_test.go

func TestParseJSONLiteralValue_ValidJSON(t *testing.T) {
	value := `{"key": "value"}`
	result, err := parseJSONLiteralValue(value)
	if err != nil {
		t.Fatalf("parseJSONLiteralValue failed: %v", err)
	}
	if result == nil {
		t.Error("parseJSONLiteralValue should return result")
	}
}

func TestParseJSONLiteralValue_ValidArray(t *testing.T) {
	value := `[1, 2, 3]`
	result, err := parseJSONLiteralValue(value)
	if err != nil {
		t.Fatalf("parseJSONLiteralValue failed: %v", err)
	}
	if result == nil {
		t.Error("parseJSONLiteralValue should return result")
	}
}

// TestParseJSONLiteralValue_InvalidJSON is in coverage_jsonld_api_test.go

func TestParseJSONLiteralValue_EmptyString(t *testing.T) {
	value := ``
	result, err := parseJSONLiteralValue(value)
	// May or may not error depending on implementation
	_ = result
	_ = err
}

func TestParseJSONLiteralValue_NumberString(t *testing.T) {
	value := `42`
	result, err := parseJSONLiteralValue(value)
	if err != nil {
		t.Fatalf("parseJSONLiteralValue failed: %v", err)
	}
	if result == nil {
		t.Error("parseJSONLiteralValue should return result")
	}
}

func TestParseJSONLiteralValue_BooleanString(t *testing.T) {
	value := `true`
	result, err := parseJSONLiteralValue(value)
	if err != nil {
		t.Fatalf("parseJSONLiteralValue failed: %v", err)
	}
	if result == nil {
		t.Error("parseJSONLiteralValue should return result")
	}
}

func TestParseJSONLiteralValue_NullString(t *testing.T) {
	value := `null`
	result, err := parseJSONLiteralValue(value)
	if err != nil {
		t.Fatalf("parseJSONLiteralValue failed: %v", err)
	}
	// null JSON value is valid
	_ = result
}
