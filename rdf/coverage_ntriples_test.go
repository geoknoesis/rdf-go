package rdf

import (
	"bufio"
	"bytes"
	"io"
	"strings"
	"testing"
)

type customTerm struct{}

func (c customTerm) Kind() TermKind { return TermIRI }
func (c customTerm) String() string { return "custom" }

func TestNTriplesDecoderErrClose(t *testing.T) {
	dec := &ntDecoder{err: io.ErrUnexpectedEOF}
	if dec.Err() != io.ErrUnexpectedEOF {
		t.Fatalf("expected Err to return error")
	}
	if err := dec.Close(); err != nil {
		t.Fatalf("expected Close nil, got %v", err)
	}
}

func TestNTriplesReadLineEOFWithPartial(t *testing.T) {
	dec := &ntDecoder{reader: bufio.NewReader(strings.NewReader("<http://example.org/s> <http://example.org/p> <http://example.org/o> ."))}
	line, err := dec.readLine()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(line, "http://example.org/s") {
		t.Fatalf("unexpected line: %s", line)
	}
}

func TestNTriplesParseOptionalTermDot(t *testing.T) {
	cursor := &ntCursor{input: "."}
	if term := cursor.parseOptionalTerm(); term != nil {
		t.Fatal("expected nil term")
	}
}

func TestNTriplesParseSubjectLiteralError(t *testing.T) {
	line := "\"v\" <http://example.org/p> <http://example.org/o> ."
	_, err := parseNTLine(line, FormatNTriples)
	if err == nil {
		t.Fatal("expected subject error")
	}
}

func TestNTriplesParseSubjectBlankNode(t *testing.T) {
	line := "_:b1 <http://example.org/p> <http://example.org/o> ."
	quad, err := parseNTLine(line, FormatNTriples)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := quad.S.(BlankNode); !ok {
		t.Fatal("expected blank node subject")
	}
}

func TestNTriplesParseTermUnexpected(t *testing.T) {
	cursor := &ntCursor{input: "$"}
	if _, err := cursor.parseTerm(true); err == nil {
		t.Fatal("expected unexpected token error")
	}
}

func TestNTriplesParseIRIError(t *testing.T) {
	cursor := &ntCursor{input: "nope"}
	if _, err := cursor.parseIRI(); err == nil {
		t.Fatal("expected IRI error")
	}
}

func TestNTriplesParseOptionalTermValue(t *testing.T) {
	cursor := &ntCursor{input: "<http://example.org/g> ."}
	term := cursor.parseOptionalTerm()
	if term == nil {
		t.Fatal("expected term")
	}
}

func TestNTriplesParseOptionalTermEmpty(t *testing.T) {
	cursor := &ntCursor{input: ""}
	if term := cursor.parseOptionalTerm(); term != nil {
		t.Fatal("expected nil for empty input")
	}
}

func TestRenderTermBranches(t *testing.T) {
	if renderTerm(BlankNode{ID: "b1"}) != "_:b1" {
		t.Fatal("expected blank node render")
	}
	if !strings.Contains(renderTerm(Literal{Lexical: "v"}), "\"v\"") {
		t.Fatal("expected literal render")
	}
	if !strings.Contains(renderTerm(Literal{Lexical: "v", Lang: "en"}), "@en") {
		t.Fatal("expected lang literal render")
	}
	if !strings.Contains(renderTerm(Literal{Lexical: "v", Datatype: IRI{Value: "http://example.org/dt"}}), "^^") {
		t.Fatal("expected datatype literal render")
	}
	if !strings.Contains(renderTerm(TripleTerm{S: IRI{Value: "s"}, P: IRI{Value: "p"}, O: IRI{Value: "o"}}), "<<") {
		t.Fatal("expected triple term render")
	}
}

func TestNTriplesParseLiteralEscapes(t *testing.T) {
	line := "<http://example.org/s> <http://example.org/p> \"a\\n\\t\\r\\\"\\\\\" ."
	quad, err := parseNTLine(line, FormatNTriples)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lit, ok := quad.O.(Literal); !ok || !strings.Contains(lit.Lexical, "\n") {
		t.Fatalf("expected escaped literal")
	}
}

func TestNTriplesParseLiteralLang(t *testing.T) {
	cursor := &ntCursor{input: "\"v\"@en"}
	term, err := cursor.parseLiteral()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if term.Lang != "en" {
		t.Fatalf("expected lang literal")
	}
}

func TestNTriplesParseLiteralDatatype(t *testing.T) {
	cursor := &ntCursor{input: "\"v\"^^<http://example.org/dt>"}
	term, err := cursor.parseLiteral()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if term.Datatype.Value != "http://example.org/dt" {
		t.Fatalf("expected datatype literal")
	}
}

func TestNTriplesLiteralDatatypeError(t *testing.T) {
	line := "<http://example.org/s> <http://example.org/p> \"v\"^^\"dt\" ."
	dec := newNTriplesDecoder(strings.NewReader(line))
	if _, err := dec.Next(); err == nil {
		t.Fatal("expected datatype error")
	}
}

func TestRenderTermDefault(t *testing.T) {
	if got := renderTerm(customTerm{}); got != "" {
		t.Fatalf("expected empty render for unknown term, got %q", got)
	}
}

func TestNTriplesNextSkipsComments(t *testing.T) {
	input := "# comment\n<http://example.org/s> <http://example.org/p> <http://example.org/o> .\n"
	dec := newNTriplesDecoder(strings.NewReader(input))
	if _, err := dec.Next(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

type errReader struct{}

func (e errReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func TestNTriplesNextReadError(t *testing.T) {
	dec := newNTriplesDecoder(errReader{})
	if _, err := dec.Next(); err == nil {
		t.Fatal("expected read error")
	}
}

func TestNTriplesLiteralUnterminatedEscape(t *testing.T) {
	line := "<http://example.org/s> <http://example.org/p> \"a\\\" .\n"
	dec := newNTriplesDecoder(strings.NewReader(line))
	if _, err := dec.Next(); err == nil {
		t.Fatal("expected unterminated escape error")
	}
}

func TestNTriplesLiteralNotAllowed(t *testing.T) {
	line := "<http://example.org/s> \"lit\" <http://example.org/o> .\n"
	dec := newNTriplesDecoder(strings.NewReader(line))
	if _, err := dec.Next(); err == nil {
		t.Fatal("expected literal not allowed error")
	}
}

func TestNTriplesEncoderGraphIgnored(t *testing.T) {
	var buf bytes.Buffer
	enc := newNTriplesEncoder(&buf)
	_ = enc.Write(Quad{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: IRI{Value: "http://example.org/o"},
		G: IRI{Value: "http://example.org/g"},
	})
	_ = enc.Flush()
	if strings.Contains(buf.String(), "http://example.org/g") {
		t.Fatalf("expected graph to be ignored in N-Triples")
	}
}

func TestNTriplesEncoderFlushError(t *testing.T) {
	enc := newNTriplesEncoder(failingWriter{})
	_ = enc.Write(Quad{S: IRI{Value: "s"}, P: IRI{Value: "p"}, O: IRI{Value: "o"}})
	if err := enc.Flush(); err == nil {
		t.Fatal("expected flush error")
	}
}

func TestNTriplesEncoderFlushCachedError(t *testing.T) {
	enc := newNTriplesEncoder(&bytes.Buffer{}).(*ntEncoder)
	enc.err = io.ErrClosedPipe
	if err := enc.Flush(); err == nil {
		t.Fatal("expected cached flush error")
	}
}

func TestNTriplesParseTripleTermMissingStart(t *testing.T) {
	cursor := &ntCursor{input: "<http://example.org/s>"}
	if _, err := cursor.parseTripleTerm(); err == nil {
		t.Fatal("expected triple term start error")
	}
}

func TestNTriplesParseTripleTermSubjectError(t *testing.T) {
	line := "<< \"lit\" <http://example.org/p> <http://example.org/o> >> <http://example.org/p2> <http://example.org/o2> ."
	dec := newNTriplesDecoder(strings.NewReader(line))
	if _, err := dec.Next(); err == nil {
		t.Fatal("expected triple term subject error")
	}
}

func TestNQuadsMissingGraphAllowed(t *testing.T) {
	line := "<http://example.org/s> <http://example.org/p> <http://example.org/o> ."
	if _, err := parseNTLine(line, FormatNQuads); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNTriplesParseSubjectInvalid(t *testing.T) {
	cursor := &ntCursor{input: "\"v\""}
	if _, err := cursor.parseSubject(); err == nil {
		t.Fatal("expected subject invalid error")
	}
}

func TestNTriplesParseIRI(t *testing.T) {
	cursor := &ntCursor{input: "<http://example.org/s>"}
	iri, err := cursor.parseIRI()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if iri.Value != "http://example.org/s" {
		t.Fatalf("unexpected IRI: %s", iri.Value)
	}
}

func TestNTriplesParseIRIUnterminated(t *testing.T) {
	cursor := &ntCursor{input: "<http://example.org/s"}
	if _, err := cursor.parseIRI(); err == nil {
		t.Fatal("expected unterminated IRI error")
	}
}

func TestNTriplesParseTermBranches(t *testing.T) {
	cursor := &ntCursor{input: "<http://example.org/s>"}
	if _, err := cursor.parseTerm(false); err != nil {
		t.Fatalf("expected IRI term")
	}
	cursor = &ntCursor{input: "_:b1"}
	if _, err := cursor.parseTerm(false); err != nil {
		t.Fatalf("expected blank node term")
	}
	cursor = &ntCursor{input: "\"v\""}
	if _, err := cursor.parseTerm(true); err != nil {
		t.Fatalf("expected literal term")
	}
	cursor = &ntCursor{input: "<< <http://example.org/s> <http://example.org/p> <http://example.org/o> >>"}
	if _, err := cursor.parseTerm(false); err != nil {
		t.Fatalf("expected triple term")
	}
}

func TestNTriplesParseBlankNodeError(t *testing.T) {
	cursor := &ntCursor{input: "bad"}
	if _, err := cursor.parseBlankNode(); err == nil {
		t.Fatal("expected blank node error")
	}
}

func TestNTriplesParseTermUnexpectedEnd(t *testing.T) {
	cursor := &ntCursor{input: ""}
	if _, err := cursor.parseTerm(false); err == nil {
		t.Fatal("expected unexpected end error")
	}
}

func TestNTriplesParseLiteralExpectedError(t *testing.T) {
	cursor := &ntCursor{input: "nope"}
	if _, err := cursor.parseLiteral(); err == nil {
		t.Fatal("expected literal error")
	}
}

func TestNTriplesParseLiteralUnknownEscape(t *testing.T) {
	cursor := &ntCursor{input: "\"a\\x\""}
	term, err := cursor.parseLiteral()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if term.Lexical != "ax" {
		t.Fatalf("unexpected escape handling: %s", term.Lexical)
	}
}

func TestNTriplesParseNTLineGraphNotAllowed(t *testing.T) {
	line := "<http://example.org/s> <http://example.org/p> <http://example.org/o> <http://example.org/g> ."
	if _, err := parseNTLine(line, FormatNTriples); err == nil {
		t.Fatal("expected graph not allowed error")
	}
}

func TestNTriplesParseTripleTermMissingEnd(t *testing.T) {
	cursor := &ntCursor{input: "<< <http://example.org/s> <http://example.org/p> <http://example.org/o> >"}
	if _, err := cursor.parseTripleTerm(); err == nil {
		t.Fatal("expected missing >> error")
	}
}

func TestNTriplesParseTripleTermPredicateError(t *testing.T) {
	cursor := &ntCursor{input: "<< <http://example.org/s> _:b1 <http://example.org/o> >>"}
	if _, err := cursor.parseTripleTerm(); err == nil {
		t.Fatal("expected predicate error")
	}
}

func TestNTriplesWriteErrors(t *testing.T) {
	enc := newNTriplesEncoder(&bytes.Buffer{})
	if err := enc.Write(Quad{}); err == nil {
		t.Fatal("expected empty statement error")
	}
	if err := enc.Write(Quad{S: IRI{Value: "s"}}); err == nil {
		t.Fatal("expected missing fields error")
	}
	enc = newNQuadsEncoder(&bytes.Buffer{})
	if err := enc.Write(Quad{S: IRI{Value: "s"}, P: IRI{Value: "p"}, O: IRI{Value: "o"}, G: IRI{Value: "g"}}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNTriplesWriteCachedError(t *testing.T) {
	enc := newNTriplesEncoder(&bytes.Buffer{}).(*ntEncoder)
	enc.err = io.ErrClosedPipe
	if err := enc.Write(Quad{S: IRI{Value: "s"}, P: IRI{Value: "p"}, O: IRI{Value: "o"}}); err == nil {
		t.Fatal("expected cached error")
	}
}

func TestNTriplesParseNTLineObjectError(t *testing.T) {
	line := "<http://example.org/s> <http://example.org/p> \"bad"
	if _, err := parseNTLine(line, FormatNTriples); err == nil {
		t.Fatal("expected object parse error")
	}
}

func TestNTriplesParseNTLinePredicateError(t *testing.T) {
	line := "<http://example.org/s> \"bad\" <http://example.org/o> ."
	if _, err := parseNTLine(line, FormatNTriples); err == nil {
		t.Fatal("expected predicate error")
	}
}

func TestNTriplesParseLiteralSimple(t *testing.T) {
	cursor := &ntCursor{input: "\"v\""}
	term, err := cursor.parseLiteral()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if term.Lexical != "v" {
		t.Fatalf("unexpected lexical: %s", term.Lexical)
	}
}

func TestNTriplesParseTripleTermObjectError(t *testing.T) {
	cursor := &ntCursor{input: "<< <http://example.org/s> <http://example.org/p> \"bad >>"}
	if _, err := cursor.parseTripleTerm(); err == nil {
		t.Fatal("expected object error")
	}
}

func TestNQuadsWriteNoGraph(t *testing.T) {
	enc := newNQuadsEncoder(&bytes.Buffer{})
	if err := enc.Write(Quad{S: IRI{Value: "s"}, P: IRI{Value: "p"}, O: IRI{Value: "o"}}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNTriplesParseSubjectTripleTerm(t *testing.T) {
	cursor := &ntCursor{input: "<< <http://example.org/s> <http://example.org/p> <http://example.org/o> >>"}
	if _, err := cursor.parseSubject(); err != nil {
		t.Fatalf("expected triple term subject")
	}
}
