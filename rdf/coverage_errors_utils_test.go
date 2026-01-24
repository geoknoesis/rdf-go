package rdf

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

// Test error handling and utility functions for maximum coverage

func TestWrapParseError_Simple(t *testing.T) {
	err := errors.New("test error")
	wrapped := wrapParseError("turtle", "line", 0, err)
	if wrapped == nil {
		t.Fatal("wrapParseError should not return nil")
	}
	if !strings.Contains(wrapped.Error(), "turtle") {
		t.Error("wrapParseError should include format name")
	}
}

func TestWrapParseErrorWithPosition_Simple(t *testing.T) {
	err := errors.New("test error")
	wrapped := wrapParseErrorWithPosition("turtle", "line", 1, 5, 10, err)
	if wrapped == nil {
		t.Fatal("wrapParseErrorWithPosition should not return nil")
	}
	if !strings.Contains(wrapped.Error(), "turtle") {
		t.Error("wrapParseErrorWithPosition should include format name")
	}
}

func TestCode_Simple(t *testing.T) {
	err := &ParseError{
		Format:    "turtle",
		Statement: "test",
		Err:       errors.New("test error"),
	}
	code := Code(err)
	if code != ErrCodeParseError {
		t.Errorf("Code = %v, want %v", code, ErrCodeParseError)
	}
}

func TestCode_WithUnderlyingError(t *testing.T) {
	underlying := ErrLineTooLong
	err := &ParseError{
		Format:    "turtle",
		Statement: "test",
		Err:       underlying,
	}
	code := Code(err)
	if code != ErrCodeLineTooLong {
		t.Errorf("Code = %v, want %v", code, ErrCodeLineTooLong)
	}
}

func TestCode_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := ctx.Err()
	code := Code(err)
	if code != ErrCodeContextCanceled {
		t.Errorf("Code = %v, want %v", code, ErrCodeContextCanceled)
	}
}

func TestCode_EOF(t *testing.T) {
	code := Code(io.EOF)
	if code != "" {
		t.Errorf("Code(EOF) = %v, want empty string", code)
	}
}

func TestCode_Nil(t *testing.T) {
	code := Code(nil)
	if code != "" {
		t.Errorf("Code(nil) = %v, want empty string", code)
	}
}

func TestParseError_Error(t *testing.T) {
	err := &ParseError{
		Format:    "turtle",
		Statement: "test statement",
		Line:      5,
		Column:    10,
		Offset:    100,
		Err:       errors.New("parse error"),
	}
	msg := err.Error()
	if !strings.Contains(msg, "turtle") {
		t.Error("ParseError.Error should include format name")
	}
	if !strings.Contains(msg, "parse error") {
		t.Error("ParseError.Error should include underlying error")
	}
}

func TestParseError_ErrorWithPosition(t *testing.T) {
	err := &ParseError{
		Format:    "turtle",
		Statement: "test statement",
		Line:      5,
		Column:    10,
		Err:       errors.New("parse error"),
	}
	msg := err.Error()
	if !strings.Contains(msg, "5:10") {
		t.Error("ParseError.Error should include line:column")
	}
}

func TestParseError_ErrorWithOffset(t *testing.T) {
	err := &ParseError{
		Format:    "turtle",
		Statement: "test statement",
		Offset:    100,
		Err:       errors.New("parse error"),
	}
	msg := err.Error()
	if !strings.Contains(msg, "offset 100") {
		t.Error("ParseError.Error should include offset")
	}
}

func TestParseError_Unwrap(t *testing.T) {
	underlying := errors.New("underlying error")
	err := &ParseError{
		Format:    "turtle",
		Statement: "test",
		Err:       underlying,
	}
	if err.Unwrap() != underlying {
		t.Error("ParseError.Unwrap should return underlying error")
	}
}

// Tests for readLineWithLimit, discardLine, and checkDecodeContext are in coverage_parse_utils_test.go
