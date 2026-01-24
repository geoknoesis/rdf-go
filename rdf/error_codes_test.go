package rdf

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestErrorCode_UnsupportedFormat(t *testing.T) {
	_, err := NewReader(strings.NewReader(""), Format("unknown"))
	if err == nil {
		t.Fatal("expected error")
	}
	code := Code(err)
	if code != ErrCodeUnsupportedFormat {
		t.Errorf("expected ErrCodeUnsupportedFormat, got %v", code)
	}
}

func TestErrorCode_LineTooLong(t *testing.T) {
	// Create input that exceeds line limit
	longLine := strings.Repeat("a", 65<<10) // 65KB line
	input := longLine + "\n"

	dec, err := NewReader(strings.NewReader(input), FormatNTriples, OptMaxLineBytes(64<<10))
	if err != nil {
		t.Fatalf("unexpected error creating reader: %v", err)
	}
	defer dec.Close()

	_, err = dec.Next()
	if err == nil {
		t.Fatal("expected error")
	}
	code := Code(err)
	if code != ErrCodeLineTooLong {
		t.Errorf("expected ErrCodeLineTooLong, got %v", code)
	}
}

func TestErrorCode_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := Parse(ctx, strings.NewReader("<s> <p> <o> ."), FormatNTriples, func(Statement) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected error")
	}
	code := Code(err)
	if code != ErrCodeContextCanceled {
		t.Errorf("expected ErrCodeContextCanceled, got %v", code)
	}
}

func TestErrorCode_ParseError(t *testing.T) {
	// Invalid Turtle input
	input := "@prefix ex: <http://example.org/> .\nex:s ex:p invalid ."
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
	if err != nil {
		t.Fatalf("unexpected error creating reader: %v", err)
	}
	defer dec.Close()

	_, err = dec.Next()
	if err == nil {
		t.Fatal("expected error")
	}
	code := Code(err)
	if code != ErrCodeParseError {
		t.Errorf("expected ErrCodeParseError, got %v", code)
	}
}

func TestErrorCode_EOF(t *testing.T) {
	// EOF should not have an error code
	input := "<http://example.org/s> <http://example.org/p> <http://example.org/o> .\n"
	dec, err := NewReader(strings.NewReader(input), FormatNTriples)
	if err != nil {
		t.Fatalf("unexpected error creating reader: %v", err)
	}
	defer dec.Close()

	// Read first statement
	_, err = dec.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Read EOF
	_, err = dec.Next()
	if err != io.EOF {
		t.Fatalf("expected EOF, got %v", err)
	}

	code := Code(err)
	if code != "" {
		t.Errorf("expected empty code for EOF, got %v", code)
	}
}

func TestErrorCode_NilError(t *testing.T) {
	code := Code(nil)
	if code != "" {
		t.Errorf("expected empty code for nil error, got %v", code)
	}
}

func TestErrorCode_WrappedError(t *testing.T) {
	// Test that wrapped errors preserve error codes
	baseErr := ErrLineTooLong
	wrapped := wrapParseError("turtle", "test", 0, baseErr)

	code := Code(wrapped)
	if code != ErrCodeLineTooLong {
		t.Errorf("expected ErrCodeLineTooLong for wrapped error, got %v", code)
	}
}

func TestErrorCode_UnknownError(t *testing.T) {
	unknownErr := errors.New("unknown error")
	code := Code(unknownErr)
	if code != ErrCodeParseError {
		t.Errorf("expected ErrCodeParseError for unknown error, got %v", code)
	}
}
