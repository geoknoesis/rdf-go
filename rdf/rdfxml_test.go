package rdf

import (
	"bytes"
	"strings"
	"testing"
)

func TestRDFXMLResourceObject(t *testing.T) {
	input := `<?xml version="1.0"?><rdf:RDF xmlns:rdf="` + rdfXMLNS + `"><rdf:Description rdf:about="http://example.org/s"><ex:p xmlns:ex="http://example.org/" rdf:resource="http://example.org/o"/></rdf:Description></rdf:RDF>`
	dec := newRDFXMLDecoder(strings.NewReader(input))
	quad, err := dec.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if iri, ok := quad.O.(IRI); !ok || iri.Value != "http://example.org/o" {
		t.Fatalf("expected IRI object")
	}
}

func TestRDFXMLNodeIDObject(t *testing.T) {
	input := `<?xml version="1.0"?><rdf:RDF xmlns:rdf="` + rdfXMLNS + `"><rdf:Description rdf:about="http://example.org/s"><ex:p xmlns:ex="http://example.org/" rdf:nodeID="b1"/></rdf:Description></rdf:RDF>`
	dec := newRDFXMLDecoder(strings.NewReader(input))
	quad, err := dec.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := quad.O.(BlankNode); !ok {
		t.Fatalf("expected blank node object")
	}
}

func TestRDFXMLUnsupportedNestedElement(t *testing.T) {
	input := `<?xml version="1.0"?><rdf:RDF xmlns:rdf="` + rdfXMLNS + `"><rdf:Description rdf:about="http://example.org/s"><ex:p xmlns:ex="http://example.org/"><ex:nested>v</ex:nested></ex:p></rdf:Description></rdf:RDF>`
	dec := newRDFXMLDecoder(strings.NewReader(input))
	if _, err := dec.Next(); err == nil {
		t.Fatal("expected nested element error")
	}
}

func TestRDFXMLEncoderUnsupportedObject(t *testing.T) {
	enc := newRDFXMLEncoder(&bytes.Buffer{})
	err := enc.Write(Quad{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: BlankNode{ID: "b1"},
	})
	if err == nil {
		t.Fatal("expected unsupported object error")
	}
}
