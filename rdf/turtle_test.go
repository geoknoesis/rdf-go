package rdf

import (
	"strings"
	"testing"
)

func TestTurtleDirectiveAndPrefixedName(t *testing.T) {
	input := "@prefix ex: <http://example.org/> .\nex:s ex:p \"v\" .\n"
	dec := newTurtleDecoder(strings.NewReader(input))
	quad, err := dec.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if quad.P.Value != "http://example.org/p" {
		t.Fatalf("unexpected predicate: %s", quad.P.Value)
	}
}

func TestTurtleBaseIRI(t *testing.T) {
	input := "@base <http://example.org/> .\n<rel> <http://example.org/p> <http://example.org/o> .\n"
	dec := newTurtleDecoder(strings.NewReader(input))
	quad, err := dec.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if iri, ok := quad.S.(IRI); !ok || iri.Value != "http://example.org/rel" {
		t.Fatalf("unexpected base IRI resolution: %#v", quad.S)
	}
}

func TestTurtleTripleTerm(t *testing.T) {
	input := "<< <http://example.org/s> <http://example.org/p> <http://example.org/o> >> <http://example.org/p2> <http://example.org/o2> .\n"
	dec := newTurtleDecoder(strings.NewReader(input))
	quad, err := dec.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := quad.S.(TripleTerm); !ok {
		t.Fatalf("expected triple term")
	}
}

func TestTurtleInvalidPredicate(t *testing.T) {
	input := "_:b1 \"literal\" <http://example.org/o> .\n"
	dec := newTurtleDecoder(strings.NewReader(input))
	if _, err := dec.Next(); err == nil {
		t.Fatal("expected predicate error")
	}
}

func TestTurtleUnknownPrefix(t *testing.T) {
	input := "ex:s ex:p ex:o .\n"
	dec := newTurtleDecoder(strings.NewReader(input))
	if _, err := dec.Next(); err == nil {
		t.Fatal("expected unknown prefix error")
	}
}

func TestTriGGraphBlock(t *testing.T) {
	input := "@prefix ex: <http://example.org/> .\nex:g { ex:s ex:p ex:o . }\n"
	dec := newTriGDecoder(strings.NewReader(input))
	quad, err := dec.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if quad.G == nil {
		t.Fatal("expected graph term")
	}
}

func TestTurtleLiteralDatatype(t *testing.T) {
	input := "@prefix ex: <http://example.org/> .\nex:s ex:p \"1\"^^ex:dt .\n"
	dec := newTurtleDecoder(strings.NewReader(input))
	quad, err := dec.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lit, ok := quad.O.(Literal); !ok || lit.Datatype.Value != "http://example.org/dt" {
		t.Fatalf("expected datatype literal")
	}
}

func TestTurtleLangLiteral(t *testing.T) {
	input := "@prefix ex: <http://example.org/> .\nex:s ex:p \"hello\"@en .\n"
	dec := newTurtleDecoder(strings.NewReader(input))
	quad, err := dec.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lit, ok := quad.O.(Literal); !ok || lit.Lang != "en" {
		t.Fatalf("expected lang literal")
	}
}

func TestTurtleBadTripleTerm(t *testing.T) {
	input := "<< <http://example.org/s> <http://example.org/p> <http://example.org/o> <http://example.org/p2> <http://example.org/o2> .\n"
	dec := newTurtleDecoder(strings.NewReader(input))
	if _, err := dec.Next(); err == nil {
		t.Fatal("expected triple term error")
	}
}
