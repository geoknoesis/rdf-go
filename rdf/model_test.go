package rdf

import "testing"

func TestTermKindsAndStrings(t *testing.T) {
	iri := IRI{Value: "http://example.org/s"}
	if iri.Kind() != TermIRI {
		t.Fatalf("expected IRI kind")
	}
	if iri.String() != "http://example.org/s" {
		t.Fatalf("unexpected IRI string: %s", iri.String())
	}

	blank := BlankNode{ID: "b1"}
	if blank.Kind() != TermBlankNode {
		t.Fatalf("expected blank node kind")
	}
	if blank.String() != "_:b1" {
		t.Fatalf("unexpected blank node string: %s", blank.String())
	}

	litPlain := Literal{Lexical: "plain"}
	if litPlain.Kind() != TermLiteral {
		t.Fatalf("expected literal kind")
	}
	if litPlain.String() != "\"plain\"" {
		t.Fatalf("unexpected literal string: %s", litPlain.String())
	}

	litLang := Literal{Lexical: "hi", Lang: "en"}
	if litLang.String() != "\"hi\"@en" {
		t.Fatalf("unexpected lang literal: %s", litLang.String())
	}

	litDT := Literal{Lexical: "1", Datatype: IRI{Value: "http://example.org/int"}}
	if litDT.String() != "\"1\"^^<http://example.org/int>" {
		t.Fatalf("unexpected datatype literal: %s", litDT.String())
	}

	tt := TripleTerm{S: iri, P: IRI{Value: "http://example.org/p"}, O: litPlain}
	if tt.Kind() != TermTriple {
		t.Fatalf("expected triple term kind")
	}
	if tt.String() != "<<http://example.org/s http://example.org/p \"plain\">>" {
		t.Fatalf("unexpected triple term string: %s", tt.String())
	}
}

func TestQuadIsZero(t *testing.T) {
	var q Quad
	if !q.IsZero() {
		t.Fatal("expected zero quad")
	}
	q.S = IRI{Value: "http://example.org/s"}
	if q.IsZero() {
		t.Fatal("expected non-zero quad")
	}
}
