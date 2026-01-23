package rdf

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

type stubStatementReader struct {
	stmts []Statement
	err   error
	index int
}

func (s *stubStatementReader) Next() (Statement, error) {
	if s.err != nil {
		return Statement{}, s.err
	}
	if s.index >= len(s.stmts) {
		return Statement{}, io.EOF
	}
	stmt := s.stmts[s.index]
	s.index++
	return stmt, nil
}
func (s *stubStatementReader) Close() error { return nil }

func TestParse_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := Parse(ctx, strings.NewReader(""), FormatNTriples, func(Statement) error { return nil })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
}

func TestParse_HandlerError(t *testing.T) {
	input := "<http://example.org/s> <http://example.org/p> \"v\" .\n"
	expected := errors.New("handler")
	err := Parse(context.Background(), strings.NewReader(input), FormatNTriples, func(Statement) error { return expected })
	if !errors.Is(err, expected) {
		t.Fatalf("expected handler error, got %v", err)
	}
}

// ParseChan removed - use Parse with handler instead

func TestParse_EOF(t *testing.T) {
	input := "<http://example.org/s> <http://example.org/p> \"v\" .\n"
	err := Parse(context.Background(), strings.NewReader(input), FormatNTriples, func(Statement) error {
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ParseChan removed - use Parse with handler instead

func TestStatementReaderEOF(t *testing.T) {
	reader := &stubStatementReader{stmts: []Statement{{S: IRI{Value: "s"}, P: IRI{Value: "p"}, O: IRI{Value: "o"}, G: nil}}}
	if _, err := reader.Next(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := reader.Next(); err != io.EOF {
		t.Fatalf("expected EOF, got %v", err)
	}
}

func TestParseSkipsZero(t *testing.T) {
	// Test with actual parsing
	input := "<http://example.org/s> <http://example.org/p> <http://example.org/o> .\n"
	count := 0
	err := Parse(context.Background(), strings.NewReader(input), FormatNQuads, func(Statement) error {
		count++
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 handled statement, got %d", count)
	}
}
