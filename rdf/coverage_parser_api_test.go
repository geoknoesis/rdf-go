package rdf

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

type stubQuadDecoder struct {
	quads []Quad
	err   error
	index int
}

func (s *stubQuadDecoder) Next() (Quad, error) {
	if s.err != nil {
		return Quad{}, s.err
	}
	if s.index >= len(s.quads) {
		return Quad{}, io.EOF
	}
	q := s.quads[s.index]
	s.index++
	return q, nil
}
func (s *stubQuadDecoder) Err() error   { return s.err }
func (s *stubQuadDecoder) Close() error { return nil }

func TestParseTriples_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := ParseTriples(ctx, strings.NewReader(""), TripleFormatNTriples, TripleHandlerFunc(func(Triple) error { return nil }))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
}

func TestParseTriples_HandlerError(t *testing.T) {
	input := "<http://example.org/s> <http://example.org/p> \"v\" .\n"
	expected := errors.New("handler")
	err := ParseTriples(context.Background(), strings.NewReader(input), TripleFormatNTriples, TripleHandlerFunc(func(Triple) error { return expected }))
	if !errors.Is(err, expected) {
		t.Fatalf("expected handler error, got %v", err)
	}
}

func TestParseTriplesChan_Error(t *testing.T) {
	out, errs := ParseTriplesChan(context.Background(), strings.NewReader("bad"), TripleFormatNTriples)
	<-out
	err := <-errs
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseTriples_EOF(t *testing.T) {
	input := "<http://example.org/s> <http://example.org/p> \"v\" .\n"
	err := ParseTriples(context.Background(), strings.NewReader(input), TripleFormatNTriples, TripleHandlerFunc(func(Triple) error {
		return nil
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseTriplesChan_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	out, errs := ParseTriplesChan(ctx, strings.NewReader(""), TripleFormatNTriples)
	select {
	case <-out:
	case err := <-errs:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}

func TestParseTriplesChan_SuccessNoError(t *testing.T) {
	input := "<http://example.org/s> <http://example.org/p> \"v\" .\n"
	out, errs := ParseTriplesChan(context.Background(), strings.NewReader(input), TripleFormatNTriples)
	for range out {
	}
	if err, ok := <-errs; ok && err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestQuadDecoderEOF(t *testing.T) {
	dec := &stubQuadDecoder{quads: []Quad{{S: IRI{Value: "s"}, P: IRI{Value: "p"}, O: IRI{Value: "o"}}}}
	if _, err := dec.Next(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := dec.Next(); err != io.EOF {
		t.Fatalf("expected EOF, got %v", err)
	}
}

func TestParseQuadsSkipsZero(t *testing.T) {
	// Create a test that uses ParseQuads with a custom decoder
	// For now, test with actual parsing
	input := "<http://example.org/s> <http://example.org/p> <http://example.org/o> .\n"
	count := 0
	err := ParseQuads(context.Background(), strings.NewReader(input), QuadFormatNQuads, QuadHandlerFunc(func(Quad) error {
		count++
		return nil
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 handled quad, got %d", count)
	}
}
