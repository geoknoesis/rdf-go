package rdf

import (
	"context"
	"strings"
	"testing"
)

func TestParseAndParseChan(t *testing.T) {
	input := "<http://example.org/s> <http://example.org/p> \"v\" .\n"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	count := 0
	err := Parse(ctx, strings.NewReader(input), FormatNTriples, HandlerFunc(func(q Quad) error {
		count++
		cancel()
		return nil
	}))
	if err != context.Canceled {
		t.Fatalf("expected context canceled, got %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 quad, got %d", count)
	}

	out, errs := ParseChan(context.Background(), strings.NewReader(input), FormatNTriples)
	select {
	case err := <-errs:
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-out:
	}
}

func TestParseHandlerError(t *testing.T) {
	input := "<http://example.org/s> <http://example.org/p> \"v\" .\n"
	err := Parse(context.Background(), strings.NewReader(input), FormatNTriples, HandlerFunc(func(q Quad) error {
		return ErrUnsupportedFormat
	}))
	if err != ErrUnsupportedFormat {
		t.Fatalf("expected handler error, got %v", err)
	}
}
