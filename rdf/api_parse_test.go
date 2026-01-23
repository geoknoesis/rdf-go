package rdf

import (
	"context"
	"strings"
	"testing"
)

func TestParseTriplesAndParseTriplesChan(t *testing.T) {
	input := "<http://example.org/s> <http://example.org/p> \"v\" .\n"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	count := 0
	err := ParseTriples(ctx, strings.NewReader(input), TripleFormatNTriples, TripleHandlerFunc(func(t Triple) error {
		count++
		cancel()
		return nil
	}))
	if err != context.Canceled {
		t.Fatalf("expected context canceled, got %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 triple, got %d", count)
	}

	out, errs := ParseTriplesChan(context.Background(), strings.NewReader(input), TripleFormatNTriples)
	select {
	case err := <-errs:
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-out:
	}
}

func TestParseTriplesHandlerError(t *testing.T) {
	input := "<http://example.org/s> <http://example.org/p> \"v\" .\n"
	err := ParseTriples(context.Background(), strings.NewReader(input), TripleFormatNTriples, TripleHandlerFunc(func(t Triple) error {
		return ErrUnsupportedFormat
	}))
	if err != ErrUnsupportedFormat {
		t.Fatalf("expected handler error, got %v", err)
	}
}
