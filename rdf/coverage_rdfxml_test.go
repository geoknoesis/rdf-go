package rdf

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"io"
	"strings"
	"testing"
)

func TestRDFXMLDecoderErrClose(t *testing.T) {
	dec := &rdfxmlDecoder{err: io.ErrUnexpectedEOF}
	if dec.Err() != io.ErrUnexpectedEOF {
		t.Fatalf("expected Err to return error")
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
	dec := newRDFXMLDecoder(strings.NewReader(input))
	for i := 0; i < 4; i++ {
		if _, err := dec.Next(); err != nil {
			t.Fatalf("unexpected error at %d: %v", i, err)
		}
	}
}

func TestRDFXMLConsumeElementEOF(t *testing.T) {
	dec := xml.NewDecoder(strings.NewReader("<rdf:RDF"))
	if err := consumeElement(dec); err == nil {
		t.Fatal("expected consume error")
	}
}

func TestRDFXMLLiteralObject(t *testing.T) {
	input := `<?xml version="1.0"?><rdf:RDF xmlns:rdf="` + rdfXMLNS + `"><rdf:Description rdf:about="http://example.org/s"><ex:p xmlns:ex="http://example.org/">literal</ex:p></rdf:Description></rdf:RDF>`
	dec := newRDFXMLDecoder(strings.NewReader(input))
	quad, err := dec.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := quad.O.(Literal); !ok {
		t.Fatalf("expected literal object")
	}
}

func TestRDFXMLNextNonRDFRoot(t *testing.T) {
	input := `<?xml version="1.0"?><ex:Thing xmlns:ex="http://example.org/" xmlns:rdf="` + rdfXMLNS + `" rdf:about="http://example.org/s"><ex:p>v</ex:p></ex:Thing>`
	dec := newRDFXMLDecoder(strings.NewReader(input))
	if _, err := dec.Next(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRDFXMLReadPredicateElementsError(t *testing.T) {
	input := `<?xml version="1.0"?><rdf:RDF xmlns:rdf="` + rdfXMLNS + `"><rdf:Description rdf:about="http://example.org/s"><ex:p xmlns:ex="http://example.org/">`
	dec := newRDFXMLDecoder(strings.NewReader(input))
	if _, err := dec.Next(); err == nil {
		t.Fatal("expected error for truncated XML")
	}
}

func TestRDFXMLEmptyDescription(t *testing.T) {
	input := `<?xml version="1.0"?><rdf:RDF xmlns:rdf="` + rdfXMLNS + `"><rdf:Description rdf:about="http://example.org/s"></rdf:Description></rdf:RDF>`
	dec := newRDFXMLDecoder(strings.NewReader(input))
	if _, err := dec.Next(); err != io.EOF {
		t.Fatalf("expected EOF, got %v", err)
	}
}

func TestRDFXMLTypeQueue(t *testing.T) {
	input := `<?xml version="1.0"?><rdf:RDF xmlns:rdf="` + rdfXMLNS + `"><ex:Thing xmlns:ex="http://example.org/" rdf:about="http://example.org/s"><ex:p>v</ex:p></ex:Thing></rdf:RDF>`
	dec := newRDFXMLDecoder(strings.NewReader(input))
	if _, err := dec.Next(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := dec.Next(); err != nil {
		t.Fatalf("unexpected error for queued type: %v", err)
	}
}

func TestRDFXMLNextSkipsNonNodeElements(t *testing.T) {
	input := `<?xml version="1.0"?><rdf:RDF xmlns:rdf="` + rdfXMLNS + `"><rdf:Bag></rdf:Bag></rdf:RDF>`
	dec := newRDFXMLDecoder(strings.NewReader(input))
	if _, err := dec.Next(); err != io.EOF {
		t.Fatalf("expected EOF, got %v", err)
	}
}

func TestRDFXMLNestedPredicateError(t *testing.T) {
	input := `<?xml version="1.0"?><rdf:RDF xmlns:rdf="` + rdfXMLNS + `"><rdf:Description rdf:about="http://example.org/s"><ex:p xmlns:ex="http://example.org/"><ex:inner>v</ex:inner></ex:p></rdf:Description></rdf:RDF>`
	dec := newRDFXMLDecoder(strings.NewReader(input))
	if _, err := dec.Next(); err == nil {
		t.Fatal("expected nested predicate error")
	}
}

func TestRDFXMLEncoderCloseWithoutWrite(t *testing.T) {
	enc := newRDFXMLEncoder(&bytes.Buffer{})
	if err := enc.Close(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRDFXMLEncoderWriteErrors(t *testing.T) {
	enc := newRDFXMLEncoder(failingWriter{})
	err := enc.Write(Quad{
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

	enc = newRDFXMLEncoder(failingWriter{})
	_ = enc.Write(Quad{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: Literal{Lexical: "v"},
	})
	if err := enc.Close(); err == nil {
		t.Fatal("expected close error")
	}
}

func TestRDFXMLEncoderUnsupportedObjectExtra(t *testing.T) {
	enc := newRDFXMLEncoder(&bytes.Buffer{}).(*rdfxmlEncoder)
	err := enc.Write(Quad{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: customTerm{},
	})
	if err == nil {
		t.Fatal("expected unsupported object error")
	}
}

func TestRDFXMLEncoderClosedError(t *testing.T) {
	enc := newRDFXMLEncoder(&bytes.Buffer{}).(*rdfxmlEncoder)
	_ = enc.Close()
	if err := enc.Write(Quad{S: IRI{Value: "s"}, P: IRI{Value: "p"}, O: IRI{Value: "o"}}); err == nil {
		t.Fatal("expected closed error")
	}
	enc.err = io.ErrClosedPipe
	if err := enc.Flush(); err == nil {
		t.Fatal("expected cached flush error")
	}
	if err := enc.Close(); err == nil {
		t.Fatal("expected cached close error")
	}
}

func TestRDFXMLEncoderHeaderWriteError(t *testing.T) {
	enc := newRDFXMLEncoder(&bytes.Buffer{}).(*rdfxmlEncoder)
	enc.writer = bufio.NewWriterSize(errWriter{}, 1)
	if err := enc.Write(Quad{S: IRI{Value: "s"}, P: IRI{Value: "p"}, O: IRI{Value: "o"}}); err == nil {
		t.Fatal("expected header write error")
	}
}

func TestRDFXMLEncoderCloseClosed(t *testing.T) {
	enc := newRDFXMLEncoder(&bytes.Buffer{}).(*rdfxmlEncoder)
	enc.err = io.ErrClosedPipe
	enc.closed = true
	if err := enc.Close(); err == nil {
		t.Fatal("expected error on closed encoder")
	}
}

func TestRDFXMLEncoderCloseTwiceWithError(t *testing.T) {
	enc := newRDFXMLEncoder(&bytes.Buffer{}).(*rdfxmlEncoder)
	_ = enc.Close()
	enc.err = io.ErrClosedPipe
	if err := enc.Close(); err == nil {
		t.Fatal("expected error on second close")
	}
}
