package rdf

import (
	"bytes"
	"strings"
	"testing"
)

func TestNTriplesDecodeErrors(t *testing.T) {
	input := "<http://example.org/s> <http://example.org/p> .\n"
	dec, err := NewTripleDecoder(strings.NewReader(input), TripleFormatNTriples)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := dec.Next(); err == nil {
		t.Fatal("expected error for missing object")
	}

	input = "<http://example.org/s> <http://example.org/p> <http://example.org/o>\n"
	dec, err = NewTripleDecoder(strings.NewReader(input), TripleFormatNTriples)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := dec.Next(); err == nil {
		t.Fatal("expected error for missing dot")
	}
}

func TestNQuadsRejectGraphInTriples(t *testing.T) {
	line := "<http://example.org/s> <http://example.org/p> <http://example.org/o> <http://example.org/g> .\n"
	dec, err := NewTripleDecoder(strings.NewReader(line), TripleFormatNTriples)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := dec.Next(); err == nil {
		t.Fatal("expected error for graph term in ntriples")
	}
}

func TestNTriplesDecodeBlankAndLiteral(t *testing.T) {
	line := "_:b1 <http://example.org/p> \"v\"@en .\n"
	dec, err := NewTripleDecoder(strings.NewReader(line), TripleFormatNTriples)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	triple, err := dec.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := triple.S.(BlankNode); !ok {
		t.Fatalf("expected blank node subject")
	}
	if lit, ok := triple.O.(Literal); !ok || lit.Lang != "en" {
		t.Fatalf("expected lang literal")
	}
}

func TestNTriplesDecodeDatatypeLiteral(t *testing.T) {
	line := "<http://example.org/s> <http://example.org/p> \"1\"^^<http://example.org/dt> .\n"
	dec, err := NewTripleDecoder(strings.NewReader(line), TripleFormatNTriples)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	triple, err := dec.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lit, ok := triple.O.(Literal); !ok || lit.Datatype.Value != "http://example.org/dt" {
		t.Fatalf("expected datatype literal")
	}
}

func TestNTriplesDecodeTripleTerm(t *testing.T) {
	line := "<< <http://example.org/s> <http://example.org/p> <http://example.org/o> >> <http://example.org/p2> <http://example.org/o2> .\n"
	dec, err := NewTripleDecoder(strings.NewReader(line), TripleFormatNTriples)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := dec.Next(); err == nil {
		t.Fatal("expected error for triple term subject")
	}
}

func TestNTriplesDecodeTripleTermError(t *testing.T) {
	line := "<< <http://example.org/s> <http://example.org/p> <http://example.org/o> <http://example.org/p2> <http://example.org/o2> .\n"
	dec, err := NewTripleDecoder(strings.NewReader(line), TripleFormatNTriples)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := dec.Next(); err == nil {
		t.Fatal("expected error for missing >>")
	}
}

func TestNTriplesDecodeUnterminatedIRI(t *testing.T) {
	line := "<http://example.org/s <http://example.org/p> <http://example.org/o> .\n"
	dec, err := NewTripleDecoder(strings.NewReader(line), TripleFormatNTriples)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := dec.Next(); err == nil {
		t.Fatal("expected unterminated IRI error")
	}
}

func TestNTriplesDecodeInvalidBlank(t *testing.T) {
	line := "_: <http://example.org/p> <http://example.org/o> .\n"
	dec, err := NewTripleDecoder(strings.NewReader(line), TripleFormatNTriples)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := dec.Next(); err == nil {
		t.Fatal("expected blank node id error")
	}
}

func TestNTriplesEncodeErrors(t *testing.T) {
	var buf bytes.Buffer
	enc, err := NewTripleEncoder(&buf, TripleFormatNTriples)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Test with valid triple
	if err := enc.Write(Triple{S: IRI{Value: "s"}, P: IRI{Value: "p"}, O: IRI{Value: "o"}}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Close to set error state
	_ = enc.Close()
	if err := enc.Write(Triple{S: IRI{Value: "s2"}, P: IRI{Value: "p2"}, O: IRI{Value: "o2"}}); err == nil {
		t.Fatal("expected cached error")
	}
}
