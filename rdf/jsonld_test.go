package rdf

import (
	"bytes"
	"strings"
	"testing"
)

func TestJSONLDParseArrayGraph(t *testing.T) {
	input := `{"@context":{"ex":"http://example.org/"},"@graph":[{"@id":"ex:s","ex:p":{"@id":"ex:o"}}]}`
	dec, err := NewTripleDecoder(strings.NewReader(input), TripleFormatJSONLD)
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
	dec, err := NewTripleDecoder(strings.NewReader(input), TripleFormatJSONLD)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := dec.Next(); err == nil {
		t.Fatal("expected missing @id error")
	}
}

func TestJSONLDUnsupportedValue(t *testing.T) {
	input := `{"@context":{"ex":"http://example.org/"},"@id":"ex:s","ex:p":{"unexpected":1}}`
	dec, err := NewTripleDecoder(strings.NewReader(input), TripleFormatJSONLD)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := dec.Next(); err == nil {
		t.Fatal("expected unsupported value error")
	}
}

func TestJSONLDEncoderCloseWithoutWrite(t *testing.T) {
	enc, err := NewTripleEncoder(&bytes.Buffer{}, TripleFormatJSONLD)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := enc.Close(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJSONLDEncoderClosedError(t *testing.T) {
	enc, err := NewTripleEncoder(&bytes.Buffer{}, TripleFormatJSONLD)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = enc.Close()
	if err := enc.Write(Triple{S: IRI{Value: "s"}, P: IRI{Value: "p"}, O: Literal{Lexical: "v"}}); err == nil {
		t.Fatal("expected closed writer error")
	}
}
