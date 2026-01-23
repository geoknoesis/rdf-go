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

func TestNewReaderUnsupportedFormat(t *testing.T) {
	_, err := NewReader(bytes.NewReader(nil), Format("bogus"))
	if err != ErrUnsupportedFormat {
		t.Fatalf("expected ErrUnsupportedFormat, got %v", err)
	}
}

func TestNewWriterUnsupportedFormat(t *testing.T) {
	_, err := NewWriter(&bytes.Buffer{}, Format("bogus"))
	if err != ErrUnsupportedFormat {
		t.Fatalf("expected ErrUnsupportedFormat, got %v", err)
	}
}

func TestTripleEncodersWriteAndFlush(t *testing.T) {
	stmt := Statement{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: Literal{Lexical: "v"},
		G: nil,
	}
	formats := []Format{FormatNTriples, FormatTurtle, FormatRDFXML, FormatJSONLD}
	for _, format := range formats {
		var buf bytes.Buffer
		enc, err := NewWriter(&buf, format)
		if err != nil {
			t.Fatalf("format %s: %v", format, err)
		}
		if err := enc.Write(stmt); err != nil {
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
	stmt := Statement{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: Literal{Lexical: "v"},
		G: IRI{Value: "http://example.org/g"},
	}
	formats := []Format{FormatNQuads, FormatTriG}
	for _, format := range formats {
		var buf bytes.Buffer
		enc, err := NewWriter(&buf, format)
		if err != nil {
			t.Fatalf("format %s: %v", format, err)
		}
		if err := enc.Write(stmt); err != nil {
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
	enc, err := NewWriter(failingWriter{}, FormatNTriples)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	stmt := Statement{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: Literal{Lexical: "v"},
		G: nil,
	}
	if err := enc.Write(stmt); err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}
	if err := enc.Flush(); err == nil {
		t.Fatal("expected flush error")
	}
}
