package rdf

import (
	"bytes"
	"strings"
	"testing"
)

func TestJSONLDParseArrayGraph(t *testing.T) {
	input := `{"@context":{"ex":"http://example.org/"},"@graph":[{"@id":"ex:s","ex:p":{"@id":"ex:o"}}]}`
	dec := newJSONLDDecoder(strings.NewReader(input))
	quad, err := dec.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if quad.P.Value != "http://example.org/p" {
		t.Fatalf("unexpected predicate: %s", quad.P.Value)
	}
}

func TestJSONLDMissingID(t *testing.T) {
	input := `{"@context":{"ex":"http://example.org/"},"ex:p":"v"}`
	dec := newJSONLDDecoder(strings.NewReader(input))
	if _, err := dec.Next(); err == nil {
		t.Fatal("expected missing @id error")
	}
}

func TestJSONLDUnsupportedValue(t *testing.T) {
	input := `{"@context":{"ex":"http://example.org/"},"@id":"ex:s","ex:p":{"unexpected":1}}`
	dec := newJSONLDDecoder(strings.NewReader(input))
	if _, err := dec.Next(); err == nil {
		t.Fatal("expected unsupported value error")
	}
}

func TestJSONLDEncoderCloseWithoutWrite(t *testing.T) {
	enc := newJSONLDEncoder(&bytes.Buffer{})
	if err := enc.Close(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJSONLDEncoderClosedError(t *testing.T) {
	enc := newJSONLDEncoder(&bytes.Buffer{})
	_ = enc.Close()
	if err := enc.Write(Quad{S: IRI{Value: "s"}, P: IRI{Value: "p"}, O: Literal{Lexical: "v"}}); err == nil {
		t.Fatal("expected closed writer error")
	}
}
