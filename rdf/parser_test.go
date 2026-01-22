package rdf

import (
	"strings"
	"testing"
)

func TestNTriplesParser_Parse(t *testing.T) {
	input := "<http://example.org/s> <http://example.org/p> <http://example.org/o> .\n"
	dec, err := NewDecoder(strings.NewReader(input), FormatNTriples)
	if err != nil {
		t.Fatalf("decoder error: %v", err)
	}
	_, err = dec.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNQuadsParser_Parse(t *testing.T) {
	input := "<http://example.org/s> <http://example.org/p> <http://example.org/o> <http://example.org/g> .\n"
	dec, err := NewDecoder(strings.NewReader(input), FormatNQuads)
	if err != nil {
		t.Fatalf("decoder error: %v", err)
	}
	quad, err := dec.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if quad.G == nil {
		t.Fatal("expected graph term to be set")
	}
}

func TestTurtleParser_Parse(t *testing.T) {
	input := "@prefix ex: <http://example.org/> .\nex:s ex:p ex:o .\n"
	dec, err := NewDecoder(strings.NewReader(input), FormatTurtle)
	if err != nil {
		t.Fatalf("decoder error: %v", err)
	}
	if _, err := dec.Next(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
