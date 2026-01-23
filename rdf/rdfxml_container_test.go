package rdf

import (
	"io"
	"strings"
	"testing"
)

func TestRDFXMLContainerMembership(t *testing.T) {
	input := `<?xml version="1.0"?><rdf:RDF xmlns:rdf="` + rdfXMLNS + `"><rdf:Bag><rdf:li>1</rdf:li><rdf:_3>3</rdf:_3><rdf:li>4</rdf:li></rdf:Bag></rdf:RDF>`
	dec, err := NewReader(strings.NewReader(input), FormatRDFXML)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var preds []string
	for {
		triple, err := dec.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("unexpected error: %v", err)
		}
		if triple.P.Value == rdfXMLNS+"type" {
			continue
		}
		preds = append(preds, triple.P.Value)
	}
	if len(preds) < 3 {
		t.Fatalf("expected at least 3 membership triples, got %d", len(preds))
	}
	if preds[0] != rdfXMLNS+"_1" {
		t.Fatalf("expected first rdf:li to map to rdf:_1, got %s", preds[0])
	}
	if preds[1] != rdfXMLNS+"_3" {
		t.Fatalf("expected rdf:_3, got %s", preds[1])
	}
	if preds[2] != rdfXMLNS+"_4" {
		t.Fatalf("expected subsequent rdf:li to map to rdf:_4, got %s", preds[2])
	}
}
