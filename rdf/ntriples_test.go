package rdf

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestNTriplesDecodeErrors(t *testing.T) {
	input := "<http://example.org/s> <http://example.org/p> .\n"
	dec := newNTriplesDecoder(strings.NewReader(input))
	if _, err := dec.Next(); err == nil {
		t.Fatal("expected error for missing object")
	}

	input = "<http://example.org/s> <http://example.org/p> <http://example.org/o>\n"
	dec = newNTriplesDecoder(strings.NewReader(input))
	if _, err := dec.Next(); err == nil {
		t.Fatal("expected error for missing dot")
	}
}

func TestNQuadsRejectGraphInTriples(t *testing.T) {
	line := "<http://example.org/s> <http://example.org/p> <http://example.org/o> <http://example.org/g> .\n"
	dec := newNTriplesDecoder(strings.NewReader(line))
	if _, err := dec.Next(); err == nil {
		t.Fatal("expected error for graph term in ntriples")
	}
}

func TestNTriplesDecodeBlankAndLiteral(t *testing.T) {
	line := "_:b1 <http://example.org/p> \"v\"@en .\n"
	dec := newNTriplesDecoder(strings.NewReader(line))
	quad, err := dec.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := quad.S.(BlankNode); !ok {
		t.Fatalf("expected blank node subject")
	}
	if lit, ok := quad.O.(Literal); !ok || lit.Lang != "en" {
		t.Fatalf("expected lang literal")
	}
}

func TestNTriplesDecodeDatatypeLiteral(t *testing.T) {
	line := "<http://example.org/s> <http://example.org/p> \"1\"^^<http://example.org/dt> .\n"
	dec := newNTriplesDecoder(strings.NewReader(line))
	quad, err := dec.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lit, ok := quad.O.(Literal); !ok || lit.Datatype.Value != "http://example.org/dt" {
		t.Fatalf("expected datatype literal")
	}
}

func TestNTriplesDecodeTripleTerm(t *testing.T) {
	line := "<< <http://example.org/s> <http://example.org/p> <http://example.org/o> >> <http://example.org/p2> <http://example.org/o2> .\n"
	dec := newNTriplesDecoder(strings.NewReader(line))
	quad, err := dec.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := quad.S.(TripleTerm); !ok {
		t.Fatalf("expected triple term subject")
	}
}

func TestNTriplesDecodeTripleTermError(t *testing.T) {
	line := "<< <http://example.org/s> <http://example.org/p> <http://example.org/o> <http://example.org/p2> <http://example.org/o2> .\n"
	dec := newNTriplesDecoder(strings.NewReader(line))
	if _, err := dec.Next(); err == nil {
		t.Fatal("expected error for missing >>")
	}
}

func TestNTriplesDecodeUnterminatedIRI(t *testing.T) {
	line := "<http://example.org/s <http://example.org/p> <http://example.org/o> .\n"
	dec := newNTriplesDecoder(strings.NewReader(line))
	if _, err := dec.Next(); err == nil {
		t.Fatal("expected unterminated IRI error")
	}
}

func TestNTriplesDecodeInvalidBlank(t *testing.T) {
	line := "_: <http://example.org/p> <http://example.org/o> .\n"
	dec := newNTriplesDecoder(strings.NewReader(line))
	if _, err := dec.Next(); err == nil {
		t.Fatal("expected blank node id error")
	}
}

func TestNTriplesEncodeErrors(t *testing.T) {
	var buf bytes.Buffer
	enc := newNTriplesEncoder(&buf).(*ntEncoder)
	if err := enc.Write(Quad{}); err == nil {
		t.Fatal("expected error for empty quad")
	}
	if err := enc.Write(Quad{S: IRI{Value: "s"}}); err == nil {
		t.Fatal("expected error for missing fields")
	}
	enc.err = io.ErrClosedPipe
	if err := enc.Write(Quad{S: IRI{Value: "s"}, P: IRI{Value: "p"}, O: IRI{Value: "o"}}); err == nil {
		t.Fatal("expected cached error")
	}
}
