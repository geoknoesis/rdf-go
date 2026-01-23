package rdf

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestRDFXMLDecoderErrClose(t *testing.T) {
	// Test error handling with actual decoder
	input := `invalid xml`
	dec, err := NewTripleDecoder(strings.NewReader(input), TripleFormatRDFXML)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := dec.Next(); err == nil {
		t.Fatal("expected error")
	}
	if dec.Err() == nil {
		t.Fatal("expected Err() to return error")
	}
	if err := dec.Close(); err != nil {
		t.Fatalf("expected Close nil, got %v", err)
	}
}

func TestRDFXMLSubjectBranches(t *testing.T) {
	input := `<?xml version="1.0"?><rdf:RDF xmlns:rdf="` + rdfXMLNS + `">
<rdf:Description rdf:about="http://example.org/s1"><ex:p xmlns:ex="http://example.org/">v</ex:p></rdf:Description>
<rdf:Description rdf:ID="id2"><ex:p xmlns:ex="http://example.org/">v</ex:p></rdf:Description>
<rdf:Description rdf:nodeID="b3"><ex:p xmlns:ex="http://example.org/">v</ex:p></rdf:Description>
<rdf:Description><ex:p xmlns:ex="http://example.org/">v</ex:p></rdf:Description>
</rdf:RDF>`
	dec, _ := NewTripleDecoder(strings.NewReader(input), TripleFormatRDFXML)
	for i := 0; i < 4; i++ {
		if _, err := dec.Next(); err != nil {
			t.Fatalf("unexpected error at %d: %v", i, err)
		}
	}
}

func TestRDFXMLConsumeElementEOF(t *testing.T) {
	rdfDec := newRDFXMLTripleDecoder(strings.NewReader("<rdf:RDF"))
	dec := rdfDec.(*rdfxmlTripleDecoder)
	if err := dec.consumeElement(); err == nil {
		t.Fatal("expected consume error")
	}
}

func TestRDFXMLLiteralObject(t *testing.T) {
	input := `<?xml version="1.0"?><rdf:RDF xmlns:rdf="` + rdfXMLNS + `"><rdf:Description rdf:about="http://example.org/s"><ex:p xmlns:ex="http://example.org/">literal</ex:p></rdf:Description></rdf:RDF>`
	dec, _ := NewTripleDecoder(strings.NewReader(input), TripleFormatRDFXML)
	triple, err := dec.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := triple.O.(Literal); !ok {
		t.Fatalf("expected literal object")
	}
}

func TestRDFXMLNextNonRDFRoot(t *testing.T) {
	input := `<?xml version="1.0"?><ex:Thing xmlns:ex="http://example.org/" xmlns:rdf="` + rdfXMLNS + `" rdf:about="http://example.org/s"><ex:p>v</ex:p></ex:Thing>`
	dec, _ := NewTripleDecoder(strings.NewReader(input), TripleFormatRDFXML)
	if _, err := dec.Next(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRDFXMLReadPredicateElementsError(t *testing.T) {
	input := `<?xml version="1.0"?><rdf:RDF xmlns:rdf="` + rdfXMLNS + `"><rdf:Description rdf:about="http://example.org/s"><ex:p xmlns:ex="http://example.org/">`
	dec, _ := NewTripleDecoder(strings.NewReader(input), TripleFormatRDFXML)
	if _, err := dec.Next(); err == nil {
		t.Fatal("expected error for truncated XML")
	}
}

func TestRDFXMLEmptyDescription(t *testing.T) {
	input := `<?xml version="1.0"?><rdf:RDF xmlns:rdf="` + rdfXMLNS + `"><rdf:Description rdf:about="http://example.org/s"></rdf:Description></rdf:RDF>`
	dec, _ := NewTripleDecoder(strings.NewReader(input), TripleFormatRDFXML)
	if _, err := dec.Next(); err != io.EOF {
		t.Fatalf("expected EOF, got %v", err)
	}
}

func TestRDFXMLTypeQueue(t *testing.T) {
	input := `<?xml version="1.0"?><rdf:RDF xmlns:rdf="` + rdfXMLNS + `"><ex:Thing xmlns:ex="http://example.org/" rdf:about="http://example.org/s"><ex:p>v</ex:p></ex:Thing></rdf:RDF>`
	dec, _ := NewTripleDecoder(strings.NewReader(input), TripleFormatRDFXML)
	if _, err := dec.Next(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := dec.Next(); err != nil {
		t.Fatalf("unexpected error for queued type: %v", err)
	}
}

func TestRDFXMLNextSkipsNonNodeElements(t *testing.T) {
	input := `<?xml version="1.0"?><rdf:RDF xmlns:rdf="` + rdfXMLNS + `"><rdf:Bag></rdf:Bag></rdf:RDF>`
	dec, _ := NewTripleDecoder(strings.NewReader(input), TripleFormatRDFXML)
	if _, err := dec.Next(); err != nil {
		t.Fatalf("expected type triple, got %v", err)
	}
	if _, err := dec.Next(); err != io.EOF {
		t.Fatalf("expected EOF, got %v", err)
	}
}

func TestRDFXMLNestedPredicateError(t *testing.T) {
	input := `<?xml version="1.0"?><rdf:RDF xmlns:rdf="` + rdfXMLNS + `"><rdf:Description rdf:about="http://example.org/s"><ex:p xmlns:ex="http://example.org/"><ex:inner>v</ex:inner></ex:p></rdf:Description></rdf:RDF>`
	dec, _ := NewTripleDecoder(strings.NewReader(input), TripleFormatRDFXML)
	if _, err := dec.Next(); err == nil {
		t.Fatal("expected nested predicate error")
	}
}

func TestRDFXMLEncoderCloseWithoutWrite(t *testing.T) {
	enc, err := NewTripleEncoder(&bytes.Buffer{}, TripleFormatRDFXML)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := enc.Close(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRDFXMLEncoderWriteErrors(t *testing.T) {
	enc, err := NewTripleEncoder(failingWriter{}, TripleFormatRDFXML)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = enc.Write(Triple{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: IRI{Value: "http://example.org/o"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := enc.Flush(); err == nil {
		t.Fatal("expected flush error")
	}

	enc, err = NewTripleEncoder(failingWriter{}, TripleFormatRDFXML)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = enc.Write(Triple{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: Literal{Lexical: "v"},
	})
	if err := enc.Close(); err == nil {
		t.Fatal("expected close error")
	}
}

func TestRDFXMLEncoderUnsupportedObjectExtra(t *testing.T) {
	enc, err := NewTripleEncoder(&bytes.Buffer{}, TripleFormatRDFXML)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = enc.Write(Triple{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: customTerm{},
	})
	if err == nil {
		t.Fatal("expected unsupported object error")
	}
}

func TestRDFXMLEncoderClosedError(t *testing.T) {
	enc, err := NewTripleEncoder(&bytes.Buffer{}, TripleFormatRDFXML)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = enc.Close()
	if err := enc.Write(Triple{S: IRI{Value: "s"}, P: IRI{Value: "p"}, O: IRI{Value: "o"}}); err == nil {
		t.Fatal("expected closed error")
	}
	if err := enc.Flush(); err == nil {
		t.Fatal("expected flush error on closed encoder")
	}
	if err := enc.Close(); err != nil {
		t.Fatalf("expected close to be idempotent: %v", err)
	}
}
