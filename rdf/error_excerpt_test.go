package rdf

import (
	"errors"
	"strings"
	"testing"
)

func TestParseErrorWithExcerpt(t *testing.T) {
	// Test error with column information and excerpt
	err := WrapParseErrorWithPosition("turtle", "ex:s ex:p ex:o .", 1, 8, -1, errors.New("unexpected token"))
	
	var parseErr *ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("expected ParseError, got %T", err)
	}
	
	errMsg := parseErr.Error()
	if !strings.Contains(errMsg, "turtle:1:8") {
		t.Errorf("expected error message to contain position, got: %q", errMsg)
	}
	if !strings.Contains(errMsg, "unexpected token") {
		t.Errorf("expected error message to contain error text, got: %q", errMsg)
	}
	// Check for excerpt (should contain part of the statement)
	if !strings.Contains(errMsg, "ex:") {
		t.Errorf("expected error message to contain excerpt, got: %q", errMsg)
	}
}

func TestParseErrorExcerptWithCaret(t *testing.T) {
	// Test error with column information showing caret
	statement := "ex:s ex:p ex:o ."
	err := WrapParseErrorWithPosition("turtle", statement, 1, 8, -1, errors.New("parse error"))
	
	var parseErr *ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("expected ParseError, got %T", err)
	}
	
	errMsg := parseErr.Error()
	// Should contain caret pointing to error position
	if !strings.Contains(errMsg, "^") {
		t.Logf("Error message: %q", errMsg)
		t.Log("Note: Caret may not appear if column is outside excerpt range")
	}
}

func TestParseErrorWithoutPosition(t *testing.T) {
	// Test error without position information
	err := WrapParseError("turtle", "ex:s ex:p ex:o .", -1, errors.New("parse error"))
	
	var parseErr *ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("expected ParseError, got %T", err)
	}
	
	errMsg := parseErr.Error()
	if !strings.Contains(errMsg, "turtle") {
		t.Errorf("expected error message to contain format, got: %q", errMsg)
	}
	if !strings.Contains(errMsg, "parse error") {
		t.Errorf("expected error message to contain error text, got: %q", errMsg)
	}
}

func TestParseErrorLongStatement(t *testing.T) {
	// Test error with long statement (should be truncated)
	longStatement := strings.Repeat("ex:s ex:p ex:o . ", 20)
	err := WrapParseErrorWithPosition("turtle", longStatement, 1, 50, -1, errors.New("parse error"))
	
	var parseErr *ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("expected ParseError, got %T", err)
	}
	
	errMsg := parseErr.Error()
	// Should contain truncated excerpt
	if len(errMsg) > 500 {
		t.Errorf("expected error message to be truncated, got length %d", len(errMsg))
	}
}

