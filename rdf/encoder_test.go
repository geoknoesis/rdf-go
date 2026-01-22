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

func TestNewDecoderUnsupportedFormat(t *testing.T) {
	_, err := NewDecoder(bytes.NewReader(nil), Format("bogus"))
	if err != ErrUnsupportedFormat {
		t.Fatalf("expected ErrUnsupportedFormat, got %v", err)
	}
}

func TestNewEncoderUnsupportedFormat(t *testing.T) {
	_, err := NewEncoder(&bytes.Buffer{}, Format("bogus"))
	if err != ErrUnsupportedFormat {
		t.Fatalf("expected ErrUnsupportedFormat, got %v", err)
	}
}

func TestEncodersWriteAndFlush(t *testing.T) {
	quad := Quad{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: Literal{Lexical: "v"},
	}
	formats := []Format{FormatNTriples, FormatNQuads, FormatTurtle, FormatTriG, FormatRDFXML, FormatJSONLD}
	for _, format := range formats {
		var buf bytes.Buffer
		enc, err := NewEncoder(&buf, format)
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

func TestEncoderWriteError(t *testing.T) {
	enc, err := NewEncoder(failingWriter{}, FormatNTriples)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	quad := Quad{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: Literal{Lexical: "v"},
	}
	if err := enc.Write(quad); err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}
	if err := enc.Flush(); err == nil {
		t.Fatal("expected flush error")
	}
}
