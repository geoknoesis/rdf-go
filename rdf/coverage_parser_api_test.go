package rdf

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

type stubDecoder struct {
	quads []Quad
	err   error
	index int
}

func (s *stubDecoder) Next() (Quad, error) {
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
func (s *stubDecoder) Err() error   { return s.err }
func (s *stubDecoder) Close() error { return nil }

func TestParse_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := Parse(ctx, strings.NewReader(""), FormatNTriples, HandlerFunc(func(Quad) error { return nil }))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
}

func TestParse_HandlerError(t *testing.T) {
	input := "<http://example.org/s> <http://example.org/p> \"v\" .\n"
	expected := errors.New("handler")
	err := Parse(context.Background(), strings.NewReader(input), FormatNTriples, HandlerFunc(func(Quad) error { return expected }))
	if !errors.Is(err, expected) {
		t.Fatalf("expected handler error, got %v", err)
	}
}

func TestParseChan_Error(t *testing.T) {
	out, errs := ParseChan(context.Background(), strings.NewReader("bad"), FormatNTriples)
	<-out
	err := <-errs
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParse_EOF(t *testing.T) {
	input := "<http://example.org/s> <http://example.org/p> \"v\" .\n"
	err := Parse(context.Background(), strings.NewReader(input), FormatNTriples, HandlerFunc(func(Quad) error {
		return nil
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParse_UnsupportedFormat(t *testing.T) {
	err := Parse(context.Background(), strings.NewReader(""), Format("bad"), HandlerFunc(func(Quad) error { return nil }))
	if err != ErrUnsupportedFormat {
		t.Fatalf("expected ErrUnsupportedFormat, got %v", err)
	}
}

func TestParseChan_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	out, errs := ParseChan(ctx, strings.NewReader(""), FormatNTriples)
	select {
	case <-out:
	case err := <-errs:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}

func TestParseChan_SuccessNoError(t *testing.T) {
	input := "<http://example.org/s> <http://example.org/p> \"v\" .\n"
	out, errs := ParseChan(context.Background(), strings.NewReader(input), FormatNTriples)
	for range out {
	}
	if err, ok := <-errs; ok && err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDecoderEOF(t *testing.T) {
	dec := &stubDecoder{quads: []Quad{{S: IRI{Value: "s"}, P: IRI{Value: "p"}, O: IRI{Value: "o"}}}}
	if _, err := dec.Next(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := dec.Next(); err != io.EOF {
		t.Fatalf("expected EOF, got %v", err)
	}
}

func TestEncoderUnsupportedFormat(t *testing.T) {
	_, err := NewEncoder(&bytes.Buffer{}, Format("bad"))
	if err != ErrUnsupportedFormat {
		t.Fatalf("expected unsupported format error, got %v", err)
	}
}

func TestParseWithDecoderSkipsZero(t *testing.T) {
	dec := &stubDecoder{
		quads: []Quad{
			{},
			{S: IRI{Value: "s"}, P: IRI{Value: "p"}, O: IRI{Value: "o"}},
		},
	}
	count := 0
	err := parseWithDecoder(context.Background(), dec, HandlerFunc(func(Quad) error {
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
