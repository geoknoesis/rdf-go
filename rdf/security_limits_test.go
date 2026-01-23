package rdf

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestMaxTriplesLimit(t *testing.T) {
	// Create input with many triples
	var lines []string
	for i := 0; i < 10; i++ {
		lines = append(lines, "<http://example.org/s> <http://example.org/p> <http://example.org/o> .\n")
	}
	input := strings.Join(lines, "")

	// Set limit to 5 triples
	opts := DecodeOptions{MaxTriples: 5}
	dec, err := NewTripleDecoderWithOptions(strings.NewReader(input), TripleFormatNTriples, DecodeOptionsToOptions(opts)...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should successfully read 5 triples
	for i := 0; i < 5; i++ {
		_, err := dec.Next()
		if err != nil {
			t.Fatalf("unexpected error reading triple %d: %v", i+1, err)
		}
	}

	// 6th triple should fail with ErrTripleLimitExceeded
	_, err = dec.Next()
	if err == nil {
		t.Fatal("expected error for exceeding MaxTriples limit")
	}
	var parseErr *ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("expected ParseError, got %T: %v", err, err)
	}
	if !errors.Is(parseErr.Err, ErrTripleLimitExceeded) {
		t.Fatalf("expected ErrTripleLimitExceeded, got: %v", parseErr.Err)
	}
	if parseErr.Line == 0 {
		t.Error("expected line number in error")
	}
}

func TestMaxTriplesLimitNQuads(t *testing.T) {
	// Create input with many quads
	var lines []string
	for i := 0; i < 10; i++ {
		lines = append(lines, "<http://example.org/s> <http://example.org/p> <http://example.org/o> <http://example.org/g> .\n")
	}
	input := strings.Join(lines, "")

	// Set limit to 3 quads
	opts := DecodeOptions{MaxTriples: 3}
	dec, err := NewQuadDecoderWithOptions(strings.NewReader(input), QuadFormatNQuads, DecodeOptionsToOptions(opts)...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should successfully read 3 quads
	for i := 0; i < 3; i++ {
		_, err := dec.Next()
		if err != nil {
			t.Fatalf("unexpected error reading quad %d: %v", i+1, err)
		}
	}

	// 4th quad should fail with ErrTripleLimitExceeded
	_, err = dec.Next()
	if err == nil {
		t.Fatal("expected error for exceeding MaxTriples limit")
	}
	var parseErr *ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("expected ParseError, got %T: %v", err, err)
	}
	if !errors.Is(parseErr.Err, ErrTripleLimitExceeded) {
		t.Fatalf("expected ErrTripleLimitExceeded, got: %v", parseErr.Err)
	}
}

func TestMaxDepthLimitTurtle(t *testing.T) {
	// Create deeply nested collection
	// Each level adds depth: ( ( ( ( ... ) ) ) )
	depth := 5
	input := strings.Repeat("(", depth) + "<http://example.org/o>" + strings.Repeat(")", depth) + " .\n"
	input = "<http://example.org/s> <http://example.org/p> " + input

	// Set limit to 3 levels
	opts := DecodeOptions{MaxDepth: 3}
	dec, err := NewTripleDecoderWithOptions(strings.NewReader(input), TripleFormatTurtle, DecodeOptionsToOptions(opts)...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should fail with ErrDepthExceeded
	_, err = dec.Next()
	if err == nil {
		t.Fatal("expected error for exceeding MaxDepth limit")
	}
	var parseErr *ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("expected ParseError, got %T: %v", err, err)
	}
	if !errors.Is(parseErr.Err, ErrDepthExceeded) {
		t.Fatalf("expected ErrDepthExceeded, got: %v", parseErr.Err)
	}
}

func TestMaxDepthLimitBlankNodeList(t *testing.T) {
	// Create deeply nested blank node property lists
	// Each level adds depth: [ [ [ [ ... ] ] ] ]
	depth := 5
	input := strings.Repeat("[ <http://example.org/p> ", depth) + "<http://example.org/o>" + strings.Repeat(" ]", depth) + " .\n"
	input = "<http://example.org/s> <http://example.org/p> " + input

	// Set limit to 3 levels
	opts := DecodeOptions{MaxDepth: 3}
	dec, err := NewTripleDecoderWithOptions(strings.NewReader(input), TripleFormatTurtle, DecodeOptionsToOptions(opts)...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should fail with ErrDepthExceeded
	_, err = dec.Next()
	if err == nil {
		t.Fatal("expected error for exceeding MaxDepth limit")
	}
	var parseErr *ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("expected ParseError, got %T: %v", err, err)
	}
	if !errors.Is(parseErr.Err, ErrDepthExceeded) {
		t.Fatalf("expected ErrDepthExceeded, got: %v", parseErr.Err)
	}
}

func TestMaxDepthLimitTriG(t *testing.T) {
	// Create deeply nested collection in TriG
	depth := 5
	collection := strings.Repeat("(", depth) + "<http://example.org/o>" + strings.Repeat(")", depth)
	input := "<http://example.org/s> <http://example.org/p> " + collection + " .\n"
	input = "@prefix ex: <http://example.org/> .\n" + input

	// Set limit to 3 levels
	opts := DecodeOptions{MaxDepth: 3}
	dec, err := NewQuadDecoderWithOptions(strings.NewReader(input), QuadFormatTriG, DecodeOptionsToOptions(opts)...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should fail with ErrDepthExceeded
	_, err = dec.Next()
	if err == nil {
		t.Fatal("expected error for exceeding MaxDepth limit")
	}
	var parseErr *ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("expected ParseError, got %T: %v", err, err)
	}
	if !errors.Is(parseErr.Err, ErrDepthExceeded) {
		t.Fatalf("expected ErrDepthExceeded, got: %v", parseErr.Err)
	}
}

func TestContextCancellation(t *testing.T) {
	// Create input with many triples
	var lines []string
	for i := 0; i < 100; i++ {
		lines = append(lines, "<http://example.org/s> <http://example.org/p> <http://example.org/o> .\n")
	}
	input := strings.Join(lines, "")

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Give it a moment to expire
	time.Sleep(2 * time.Millisecond)

	opts := DecodeOptions{Context: ctx}
	dec, err := NewTripleDecoderWithOptions(strings.NewReader(input), TripleFormatNTriples, DecodeOptionsToOptions(opts)...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should fail with context error
	_, err = dec.Next()
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context error, got: %v", err)
	}
}

func TestSafeDecodeOptions(t *testing.T) {
	safe := SafeDecodeOptions()

	// Verify safe limits are stricter than defaults
	if safe.MaxLineBytes >= DefaultMaxLineBytes {
		t.Errorf("SafeDecodeOptions.MaxLineBytes (%d) should be < DefaultMaxLineBytes (%d)", safe.MaxLineBytes, DefaultMaxLineBytes)
	}
	if safe.MaxStatementBytes >= DefaultMaxStatementBytes {
		t.Errorf("SafeDecodeOptions.MaxStatementBytes (%d) should be < DefaultMaxStatementBytes (%d)", safe.MaxStatementBytes, DefaultMaxStatementBytes)
	}
	if safe.MaxDepth >= DefaultMaxDepth {
		t.Errorf("SafeDecodeOptions.MaxDepth (%d) should be < DefaultMaxDepth (%d)", safe.MaxDepth, DefaultMaxDepth)
	}
	if safe.MaxTriples >= DefaultMaxTriples {
		t.Errorf("SafeDecodeOptions.MaxTriples (%d) should be < DefaultMaxTriples (%d)", safe.MaxTriples, DefaultMaxTriples)
	}

	// Verify safe limits are positive
	if safe.MaxLineBytes <= 0 {
		t.Error("SafeDecodeOptions.MaxLineBytes should be positive")
	}
	if safe.MaxStatementBytes <= 0 {
		t.Error("SafeDecodeOptions.MaxStatementBytes should be positive")
	}
	if safe.MaxDepth <= 0 {
		t.Error("SafeDecodeOptions.MaxDepth should be positive")
	}
	if safe.MaxTriples <= 0 {
		t.Error("SafeDecodeOptions.MaxTriples should be positive")
	}
}

func TestErrorLineNumberTracking(t *testing.T) {
	// Create input with valid line, then invalid line
	input := "<http://example.org/s> <http://example.org/p> <http://example.org/o> .\n" +
		"<http://example.org/s2> <http://example.org/p2> <http://example.org/o2>\n" // Missing dot

	dec, err := NewTripleDecoder(strings.NewReader(input), TripleFormatNTriples)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// First triple should succeed
	_, err = dec.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Second triple should fail with line number
	_, err = dec.Next()
	if err == nil {
		t.Fatal("expected error for missing dot")
	}
	var parseErr *ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("expected ParseError, got %T: %v", err, err)
	}
	if parseErr.Line != 2 {
		t.Errorf("expected line 2, got %d", parseErr.Line)
	}
}

func TestFunctionalOptions(t *testing.T) {
	// Test that functional options work correctly
	input := "<http://example.org/s> <http://example.org/p> <http://example.org/o> .\n"

	dec, err := NewTripleDecoderWithOptions(
		strings.NewReader(input),
		TripleFormatNTriples,
		WithMaxTriples(1),
		WithMaxDepth(50),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// First triple should succeed
	_, err = dec.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Second triple should fail (MaxTriples = 1)
	_, err = dec.Next()
	if err == nil {
		t.Fatal("expected error for exceeding MaxTriples")
	}
}

func TestWithSafeLimits(t *testing.T) {
	input := "<http://example.org/s> <http://example.org/p> <http://example.org/o> .\n"

	dec, err := NewTripleDecoderWithOptions(
		strings.NewReader(input),
		TripleFormatNTriples,
		WithSafeLimits(),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should work with safe limits
	_, err = dec.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

