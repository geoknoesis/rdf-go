package rdf

import (
	"strings"
	"testing"
)

func TestTurtleDecoderErrClose(t *testing.T) {
	// Test error handling with actual decoder
	input := `invalid`
	dec, err := NewTripleDecoder(strings.NewReader(input), TripleFormatTurtle)
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

func TestStripCommentNoComment(t *testing.T) {
	if stripComment("no comment") != "no comment" {
		t.Fatalf("expected unchanged")
	}
}

func TestStripCommentWithComment(t *testing.T) {
	if stripComment("value # comment") != "value " {
		t.Fatalf("expected comment stripped")
	}
}

func TestPeekNext(t *testing.T) {
	cursor := &turtleCursor{input: "a "}
	if cursor.peekNext() != ' ' {
		t.Fatalf("unexpected peek value")
	}
	cursor = &turtleCursor{input: "a"}
	if cursor.peekNext() != 0 {
		t.Fatalf("expected zero on end")
	}
}

func TestParsePredicateKeywordA(t *testing.T) {
	cursor := &turtleCursor{input: "a "}
	pred, err := cursor.parsePredicate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pred.Value != rdfTypeIRI {
		t.Fatalf("unexpected predicate: %s", pred.Value)
	}
}

func TestTurtleParsePredicatePrefixed(t *testing.T) {
	cursor := &turtleCursor{
		input:    "ex:p",
		prefixes: map[string]string{"ex": "http://example.org/"},
	}
	pred, err := cursor.parsePredicate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pred.Value != "http://example.org/p" {
		t.Fatalf("unexpected predicate: %s", pred.Value)
	}
}

func TestTurtleConsumeFalse(t *testing.T) {
	cursor := &turtleCursor{input: "x"}
	if cursor.consume('.') {
		t.Fatal("expected consume false")
	}
}

func TestTurtleParsePrefixedNameError(t *testing.T) {
	cursor := &turtleCursor{input: "badtoken"}
	if _, err := cursor.parsePrefixedName(); err == nil {
		t.Fatal("expected prefixed name error")
	}
}

func TestTurtleParsePrefixedNameEmpty(t *testing.T) {
	cursor := &turtleCursor{input: ""}
	if _, err := cursor.parsePrefixedName(); err == nil {
		t.Fatal("expected empty token error")
	}
}

func TestTurtleParseIRIUnterminated(t *testing.T) {
	cursor := &turtleCursor{input: "<http://example.org/s"}
	if _, err := cursor.parseIRI(); err == nil {
		t.Fatal("expected unterminated IRI error")
	}
}

func TestTurtleParseIRINoStart(t *testing.T) {
	cursor := &turtleCursor{input: "http://example.org/s"}
	if _, err := cursor.parseIRI(); err == nil {
		t.Fatal("expected IRI start error")
	}
}

func TestTurtleParseBlankNodeError(t *testing.T) {
	cursor := &turtleCursor{input: "_:"}
	if _, err := cursor.parseBlankNode(); err == nil {
		t.Fatal("expected blank node error")
	}
}

func TestTurtleParseBlankNodeInvalid(t *testing.T) {
	cursor := &turtleCursor{input: "bad"}
	if _, err := cursor.parseBlankNode(); err == nil {
		t.Fatal("expected blank node invalid error")
	}
}

func TestTurtleParseLiteralEscapes(t *testing.T) {
	cursor := &turtleCursor{input: "\"a\\n\\t\\r\\\"\\\\\""}
	term, err := cursor.parseLiteral()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lit := term.(Literal)
	if !strings.Contains(lit.Lexical, "\n") {
		t.Fatalf("expected escaped literal")
	}
}

func TestTurtleLiteralUnterminatedEscape(t *testing.T) {
	cursor := &turtleCursor{input: "\"a\\"}
	if _, err := cursor.parseLiteral(); err == nil {
		t.Fatal("expected escape error")
	}
}

func TestTurtleParseTermUnexpected(t *testing.T) {
	cursor := &turtleCursor{input: "$"}
	if _, err := cursor.parseTerm(true); err == nil {
		t.Fatal("expected unexpected token error")
	}
}

func TestTurtleParseTermUnexpectedEnd(t *testing.T) {
	cursor := &turtleCursor{input: ""}
	if _, err := cursor.parseTerm(false); err == nil {
		t.Fatal("expected unexpected end error")
	}
}

func TestTurtleParseTermBlankNode(t *testing.T) {
	cursor := &turtleCursor{input: "_:b1"}
	if term, err := cursor.parseTerm(false); err != nil || term == nil {
		t.Fatalf("expected blank node term")
	}
}

func TestTurtleParseTermLiteralNotAllowed(t *testing.T) {
	cursor := &turtleCursor{input: "\"v\""}
	if _, err := cursor.parseTerm(false); err == nil {
		t.Fatal("expected literal not allowed error")
	}
}

// These tests are for internal parsing details, skip them

func TestTurtleParseSubjectInvalid(t *testing.T) {
	cursor := &turtleCursor{input: "\"v\""}
	if _, err := cursor.parseSubject(); err == nil {
		t.Fatal("expected subject error")
	}
}

func TestTurtleParseSubjectBlankNode(t *testing.T) {
	cursor := &turtleCursor{input: "_:b1"}
	if _, err := cursor.parseSubject(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTurtleParseLiteralDatatypeError(t *testing.T) {
	cursor := &turtleCursor{input: "\"v\"^^\"bad\""}
	if _, err := cursor.parseLiteral(); err == nil {
		t.Fatal("expected datatype error")
	}
}

func TestTurtleParseLiteralDatatypeIRI(t *testing.T) {
	cursor := &turtleCursor{input: "\"v\"^^<http://example.org/dt>"}
	if term, err := cursor.parseLiteral(); err != nil || term.(Literal).Datatype.Value == "" {
		t.Fatalf("expected datatype literal")
	}
}

func TestTurtleParseLiteralLang(t *testing.T) {
	cursor := &turtleCursor{input: "\"v\"@en"}
	term, err := cursor.parseLiteral()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if term.(Literal).Lang != "en" {
		t.Fatalf("expected lang literal")
	}
}

func TestTurtleParseLiteralExpectedError(t *testing.T) {
	cursor := &turtleCursor{input: "nope"}
	if _, err := cursor.parseLiteral(); err == nil {
		t.Fatal("expected literal error")
	}
}

func TestTriGMultiLineGraph(t *testing.T) {
	input := "@prefix ex: <http://example.org/> .\nex:g {\nex:s ex:p ex:o .\n}\n"
	dec, err := NewQuadDecoder(strings.NewReader(input), QuadFormatTriG)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	quad, err := dec.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if quad.G == nil {
		t.Fatal("expected graph term")
	}
}

func TestTurtleNextSkipsEmptyAndComment(t *testing.T) {
	input := "\n# comment\n@prefix ex: <http://example.org/> .\nex:s ex:p ex:o .\n"
	dec, err := NewTripleDecoder(strings.NewReader(input), TripleFormatTurtle)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := dec.Next(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTurtleParseTripleTermDirect(t *testing.T) {
	cursor := &turtleCursor{input: "<< <http://example.org/s> <http://example.org/p> <http://example.org/o> >>"}
	if _, err := cursor.parseTripleTerm(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cursor = &turtleCursor{input: "<http://example.org/s>"}
	if _, err := cursor.parseTripleTerm(); err == nil {
		t.Fatal("expected triple term error")
	}
	cursor = &turtleCursor{input: "<< <http://example.org/s> \"bad\" <http://example.org/o> >>"}
	if _, err := cursor.parseTripleTerm(); err == nil {
		t.Fatal("expected triple term predicate error")
	}
	cursor = &turtleCursor{input: "<< <http://example.org/s> <http://example.org/p> <http://example.org/o> >"}
	if _, err := cursor.parseTripleTerm(); err == nil {
		t.Fatal("expected triple term closing error")
	}
	cursor = &turtleCursor{input: "<< << <http://example.org/s> <http://example.org/p> <http://example.org/o> >> <http://example.org/p2> <http://example.org/o2> >>"}
	if _, err := cursor.parseTripleTerm(); err != nil {
		t.Fatalf("unexpected nested triple term error: %v", err)
	}
}

func TestTriGGraphReset(t *testing.T) {
	input := "@prefix ex: <http://example.org/> .\nex:g {\n}\nex:s ex:p ex:o .\n"
	dec, err := NewQuadDecoder(strings.NewReader(input), QuadFormatTriG)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	quad, err := dec.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if quad.G != nil {
		t.Fatal("expected default graph after reset")
	}
}

func TestTurtleNextGraphOpenLine(t *testing.T) {
	input := "@prefix ex: <http://example.org/> .\nex:g {\nex:s ex:p ex:o .\n"
	dec, err := NewQuadDecoder(strings.NewReader(input), QuadFormatTriG)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := dec.Next(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTriGInlineGraph(t *testing.T) {
	input := "<g> { <s> <p> <o> . }"
	dec, err := NewQuadDecoder(strings.NewReader(input), QuadFormatTriG)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := dec.Next(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTurtlePredicateNonIRI(t *testing.T) {
	cursor := &turtleCursor{input: "\"v\""}
	if _, err := cursor.parsePredicate(); err == nil {
		t.Fatal("expected predicate error")
	}
}

func TestTurtleEncoderErrors(t *testing.T) {
	enc, err := NewTripleEncoder(failingWriter{}, TripleFormatTurtle)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = enc.Write(Triple{S: IRI{Value: "s"}, P: IRI{Value: "p"}, O: IRI{Value: "o"}})
	if err := enc.Flush(); err == nil {
		t.Fatal("expected flush error")
	}
}

func TestTurtleEncoderErrorState(t *testing.T) {
	enc, err := NewTripleEncoder(&strings.Builder{}, TripleFormatTurtle)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = enc.Write(Triple{S: IRI{Value: "s"}, P: IRI{Value: "p"}, O: IRI{Value: "o"}})
	// Close to set error state
	_ = enc.Close()
	if err := enc.Write(Triple{S: IRI{Value: "s2"}, P: IRI{Value: "p2"}, O: IRI{Value: "o2"}}); err == nil {
		t.Fatal("expected cached error")
	}
	if err := enc.Flush(); err == nil {
		t.Fatal("expected cached flush error")
	}
}

func TestTurtleEncoderDefaultGraph(t *testing.T) {
	enc, err := NewQuadEncoder(&strings.Builder{}, QuadFormatTriG)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := enc.Write(Quad{S: IRI{Value: "s"}, P: IRI{Value: "p"}, O: IRI{Value: "o"}}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTriGEncoderGraph(t *testing.T) {
	enc, err := NewQuadEncoder(&strings.Builder{}, QuadFormatTriG)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := enc.Write(Quad{S: IRI{Value: "s"}, P: IRI{Value: "p"}, O: IRI{Value: "o"}, G: IRI{Value: "g"}}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
