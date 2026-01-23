package rdf

import (
	"bytes"
	"io"
	"testing"
)

type failingWriter struct{}

func (f failingWriter) Write(p []byte) (int, error) {
	return 0, io.ErrClosedPipe
}

func TestNewTripleDecoderUnsupportedFormat(t *testing.T) {
	_, err := NewTripleDecoder(bytes.NewReader(nil), TripleFormat("bogus"))
	if err != ErrUnsupportedFormat {
		t.Fatalf("expected ErrUnsupportedFormat, got %v", err)
	}
}

func TestNewQuadDecoderUnsupportedFormat(t *testing.T) {
	_, err := NewQuadDecoder(bytes.NewReader(nil), QuadFormat("bogus"))
	if err != ErrUnsupportedFormat {
		t.Fatalf("expected ErrUnsupportedFormat, got %v", err)
	}
}

func TestNewTripleEncoderUnsupportedFormat(t *testing.T) {
	_, err := NewTripleEncoder(&bytes.Buffer{}, TripleFormat("bogus"))
	if err != ErrUnsupportedFormat {
		t.Fatalf("expected ErrUnsupportedFormat, got %v", err)
	}
}

func TestNewQuadEncoderUnsupportedFormat(t *testing.T) {
	_, err := NewQuadEncoder(&bytes.Buffer{}, QuadFormat("bogus"))
	if err != ErrUnsupportedFormat {
		t.Fatalf("expected ErrUnsupportedFormat, got %v", err)
	}
}

func TestTripleEncodersWriteAndFlush(t *testing.T) {
	triple := Triple{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: Literal{Lexical: "v"},
	}
	formats := []TripleFormat{TripleFormatNTriples, TripleFormatTurtle, TripleFormatRDFXML, TripleFormatJSONLD}
	for _, format := range formats {
		var buf bytes.Buffer
		enc, err := NewTripleEncoder(&buf, format)
		if err != nil {
			t.Fatalf("format %s: %v", format, err)
		}
		if err := enc.Write(triple); err != nil {
			t.Fatalf("format %s: write error %v", format, err)
		}
		if err := enc.Flush(); err != nil {
			t.Fatalf("format %s: flush error %v", format, err)
		}
		if err := enc.Close(); err != nil {
			t.Fatalf("format %s: close error %v", format, err)
		}
	}
}

func TestQuadEncodersWriteAndFlush(t *testing.T) {
	quad := Quad{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: Literal{Lexical: "v"},
		G: IRI{Value: "http://example.org/g"},
	}
	formats := []QuadFormat{QuadFormatNQuads, QuadFormatTriG}
	for _, format := range formats {
		var buf bytes.Buffer
		enc, err := NewQuadEncoder(&buf, format)
		if err != nil {
			t.Fatalf("format %s: %v", format, err)
		}
		if err := enc.Write(quad); err != nil {
			t.Fatalf("format %s: write error %v", format, err)
		}
		if err := enc.Flush(); err != nil {
			t.Fatalf("format %s: flush error %v", format, err)
		}
		if err := enc.Close(); err != nil {
			t.Fatalf("format %s: close error %v", format, err)
		}
	}
}

func TestTripleEncoderWriteError(t *testing.T) {
	enc, err := NewTripleEncoder(failingWriter{}, TripleFormatNTriples)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	triple := Triple{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: Literal{Lexical: "v"},
	}
	if err := enc.Write(triple); err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}
	if err := enc.Flush(); err == nil {
		t.Fatal("expected flush error")
	}
}
