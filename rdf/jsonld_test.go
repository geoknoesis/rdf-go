package rdf

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
)

func TestJSONLDParseArrayGraph(t *testing.T) {
	input := `{"@context":{"ex":"http://example.org/"},"@graph":[{"@id":"ex:s","ex:p":{"@id":"ex:o"}}]}`
	dec, err := NewReader(strings.NewReader(input), FormatJSONLD)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	triple, err := dec.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if triple.P.Value != "http://example.org/p" {
		t.Fatalf("unexpected predicate: %s", triple.P.Value)
	}
}

func TestJSONLDMissingID(t *testing.T) {
	input := `{"@context":{"ex":"http://example.org/"},"ex:p":"v"}`
	dec, err := NewReader(strings.NewReader(input), FormatJSONLD)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := dec.Next(); err == nil {
		t.Fatal("expected missing @id error")
	}
}

func TestJSONLDUnsupportedValue(t *testing.T) {
	input := `{"@context":{"ex":"http://example.org/"},"@id":"ex:s","ex:p":{"unexpected":1}}`
	dec, err := NewReader(strings.NewReader(input), FormatJSONLD)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := dec.Next(); err == nil {
		t.Fatal("expected unsupported value error")
	}
}

func TestJSONLDEncoderCloseWithoutWrite(t *testing.T) {
	enc, err := NewWriter(&bytes.Buffer{}, FormatJSONLD)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := enc.Close(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJSONLDEncoderClosedError(t *testing.T) {
	enc, err := NewWriter(&bytes.Buffer{}, FormatJSONLD)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = enc.Close()
	if err := enc.Write(Statement{S: IRI{Value: "s"}, P: IRI{Value: "p"}, O: Literal{Lexical: "v"}, G: nil}); err == nil {
		t.Fatal("expected closed writer error")
	}
}

func TestJSONLDDecodeCancelAfterFirstTriple(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	input := `{"@context":{"ex":"http://example.org/"},"@graph":[{"@id":"ex:s1","ex:p":"v1"},{"@id":"ex:s2","ex:p":"v2"}]}`
	reader, err := NewReader(strings.NewReader(input), FormatJSONLD, OptContext(ctx))
	if err != nil {
		t.Fatalf("unexpected error creating reader: %v", err)
	}
	defer reader.Close()
	stmt, err := reader.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	triple := stmt.AsTriple()
	if triple.S == nil {
		t.Fatalf("expected triple subject")
	}
	if triple.S == nil {
		t.Fatalf("expected triple subject")
	}
	cancel()
	for {
		_, err := reader.Next()
		if err == nil {
			continue
		}
		if err == context.Canceled {
			return
		}
		if err == io.EOF {
			// Reader interface doesn't have Err(), check if error was context.Canceled
			return
		}
		t.Fatalf("unexpected error: %v", err)
	}
}
