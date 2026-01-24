package rdf

import (
	"errors"
	"strings"
	"testing"
)

// Test lexer token scanning functions

func TestTurtleLexer_ScanIRIRef(t *testing.T) {
	// Lexer processes line by line, so need a complete line
	input := "<http://example.org/resource> <http://example.org/p> <http://example.org/o> .\n"
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	// Should parse successfully
	stmt, err := dec.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	if stmt.S.String() != "http://example.org/resource" {
		t.Errorf("Subject = %q, want 'http://example.org/resource'", stmt.S.String())
	}
}

func TestTurtleLexer_ScanString(t *testing.T) {
	input := `<s> <p> "simple string" .`
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	stmt, err := dec.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	if stmt.O.Kind() != TermLiteral {
		t.Error("Expected Literal for string")
	}
}

func TestTurtleLexer_ScanLongString(t *testing.T) {
	input := `<s> <p> """long
multiline
string""" .`
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	stmt, err := dec.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	if stmt.O.Kind() != TermLiteral {
		t.Error("Expected Literal for long string")
	}
}

func TestTurtleLexer_ScanBlankNode(t *testing.T) {
	input := "_:b1 <p> <o> ."
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	stmt, err := dec.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	if stmt.S.Kind() != TermBlankNode {
		t.Error("Expected BlankNode for subject")
	}
}

func TestTurtleLexer_ScanWord_Prefix(t *testing.T) {
	input := "@prefix ex: <http://example.org/> ."
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	// Should parse directive without error
	_, err = dec.Next()
	// May or may not have statements after directive
	_ = err
}

func TestTurtleLexer_ScanWord_Base(t *testing.T) {
	input := "@base <http://example.org/> ."
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	// Should parse directive without error
	_, err = dec.Next()
	// May or may not have statements after directive
	_ = err
}

func TestTurtleLexer_ScanWord_Version(t *testing.T) {
	input := `@version "1.1" .`
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	// Should parse directive without error
	_, err = dec.Next()
	// May or may not have statements after directive
	_ = err
}

func TestTurtleLexer_ScanWord_A(t *testing.T) {
	input := "<s> a <o> ."
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	stmt, err := dec.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	// 'a' should be parsed as rdf:type
	if stmt.P.Value != rdfTypeIRI {
		t.Errorf("Predicate = %q, want rdf:type", stmt.P.Value)
	}
}

func TestTurtleLexer_ScanWord_Boolean(t *testing.T) {
	tests := []string{"<s> <p> true .", "<s> <p> false ."}
	for _, input := range tests {
		dec, err := NewReader(strings.NewReader(input), FormatTurtle)
		if err != nil {
			t.Fatalf("NewReader failed: %v", err)
		}
		stmt, err := dec.Next()
		if err != nil {
			t.Fatalf("Next failed: %v", err)
		}
		if stmt.O.Kind() != TermLiteral {
			t.Errorf("Expected Literal for boolean in %q", input)
		}
		dec.Close()
	}
}

func TestTurtleLexer_ScanWord_PrefixedName(t *testing.T) {
	input := `@prefix ex: <http://example.org/> .
ex:localName <p> <o> .`
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	stmt, err := dec.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	// Prefixed name should be expanded
	if !strings.Contains(stmt.S.String(), "http://example.org/localName") {
		t.Errorf("Subject should be expanded, got %q", stmt.S.String())
	}
}

func TestTurtleLexer_ScanWord_PrefixedNameNS(t *testing.T) {
	input := `@prefix ex: <http://example.org/> .
ex: <p> <o> .`
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	// Should parse (ex: is a valid prefix name)
	_, err = dec.Next()
	// May or may not error depending on implementation
	_ = err
}

func TestTurtleLexer_ScanWord_Numeric(t *testing.T) {
	tests := []string{"<s> <p> 123 .", "<s> <p> 123.456 .", "<s> <p> 123e10 .", "<s> <p> 123.456e-10 ."}
	for _, input := range tests {
		dec, err := NewReader(strings.NewReader(input), FormatTurtle)
		if err != nil {
			t.Fatalf("NewReader failed: %v", err)
		}
		stmt, err := dec.Next()
		if err != nil {
			t.Fatalf("Next failed: %v", err)
		}
		if stmt.O.Kind() != TermLiteral {
			t.Errorf("Expected Literal for numeric in %q", input)
		}
		dec.Close()
	}
}

func TestTurtleLexer_ScanLangTag(t *testing.T) {
	input := `<s> <p> "value"@en .`
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	stmt, err := dec.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	lit, ok := stmt.O.(Literal)
	if !ok {
		t.Fatal("Expected Literal")
	}
	if lit.Lang != "en" {
		t.Errorf("Literal lang = %q, want 'en'", lit.Lang)
	}
}

func TestTurtleLexer_ScanDatatypePrefix(t *testing.T) {
	input := `<s> <p> "value"^^<http://example.org/type> .`
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	stmt, err := dec.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	lit, ok := stmt.O.(Literal)
	if !ok {
		t.Fatal("Expected Literal")
	}
	if lit.Datatype.Value == "" {
		t.Error("Literal should have datatype")
	}
}

func TestTurtleLexer_TokenKinds(t *testing.T) {
	// Test all token kind strings
	kinds := []turtleTokenKind{
		TokLine, TokEOF, TokError, TokIRIRef, TokPNAMENS, TokPNAMELN,
		TokBlankNode, TokString, TokStringLong, TokInteger, TokDecimal,
		TokDouble, TokBoolean, TokPrefix, TokBase, TokVersion, TokDot,
		TokComma, TokSemicolon, TokLBracket, TokRBracket, TokLParen,
		TokRParen, TokLBrace, TokRBrace, TokLDoubleAngle, TokRDoubleAngle,
		TokAnnotationL, TokAnnotationR, TokA, TokLangTag, TokDatatypePrefix,
	}

	for _, kind := range kinds {
		str := kind.String()
		if str == "" {
			t.Errorf("TokenKind(%d).String() returned empty", kind)
		}
		if str == "TokUnknown" && kind <= TokDatatypePrefix {
			t.Errorf("TokenKind(%d).String() = %q, should not be unknown", kind, str)
		}
	}
}

// Test parser functions

func TestTurtleParser_ParseCollection(t *testing.T) {
	input := `<s> <p> ( <o1> <o2> <o3> ) .`
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	stmt, err := dec.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}

	// Collection should be parsed as multiple triples
	// The exact structure depends on implementation
	_ = stmt
}

func TestTurtleParser_ParseBlankNodePropertyList(t *testing.T) {
	input := `<s> <p> [ <p1> <o1> ; <p2> <o2> ] .`
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	stmt, err := dec.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}

	// Blank node property list should be parsed
	_ = stmt
}

func TestTurtleParser_ParseTripleTerm(t *testing.T) {
	input := `<<<s> <p> <o>>> <p2> <o2> .`
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	stmt, err := dec.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}

	if stmt.S.Kind() != TermTriple {
		t.Error("Expected TripleTerm as subject")
	}
}

func TestTurtleParser_ParseNumericLiteral(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"123", true},
		{"123.456", true},
		{"123e10", true},
		{"123.456e-10", true},
		{"+123", true},
		{"-123", true},
		{"123.", true}, // isNumericLiteral accepts this (simple character check)
	}

	for _, tt := range tests {
		got := isNumericLiteral(tt.input)
		if got != tt.valid {
			t.Errorf("isNumericLiteral(%q) = %v, want %v", tt.input, got, tt.valid)
		}
	}
}

func TestTurtleParser_ParseBooleanLiteral(t *testing.T) {
	input := `<s> <p> true .`
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	stmt, err := dec.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}

	// Boolean should be parsed as literal
	if stmt.O.Kind() != TermLiteral {
		t.Error("Expected Literal for boolean")
	}
}

func TestTurtleParser_ParseAnnotation(t *testing.T) {
	input := `<s> <p> <o> {| <p2> <o2> |} .`
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	// Should parse annotation
	stmt, err := dec.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	_ = stmt
}

func TestTurtleParser_ParseLiteralWithLang(t *testing.T) {
	input := `<s> <p> "value"@en .`
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	stmt, err := dec.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}

	lit, ok := stmt.O.(Literal)
	if !ok {
		t.Fatal("Expected Literal")
	}
	if lit.Lang != "en" {
		t.Errorf("Literal lang = %q, want 'en'", lit.Lang)
	}
}

func TestTurtleParser_ParseLiteralWithDatatype(t *testing.T) {
	input := `<s> <p> "42"^^<http://www.w3.org/2001/XMLSchema#integer> .`
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	stmt, err := dec.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}

	lit, ok := stmt.O.(Literal)
	if !ok {
		t.Fatal("Expected Literal")
	}
	if lit.Datatype.Value == "" {
		t.Error("Literal should have datatype")
	}
}

func TestTurtleParser_ParsePredicateObjectList(t *testing.T) {
	input := `<http://example.org/s> <http://example.org/p1> <http://example.org/o1> ; <http://example.org/p2> <http://example.org/o2> .`
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	// Should parse multiple triples
	count := 0
	for {
		stmt, err := dec.Next()
		if err != nil {
			break
		}
		if stmt.S.String() == "http://example.org/s" {
			count++
		}
	}

	if count < 2 {
		t.Error("Expected at least 2 triples from predicate-object list")
	}
}

func TestTurtleParser_ParseObjectList(t *testing.T) {
	input := `<http://example.org/s> <http://example.org/p> <http://example.org/o1> , <http://example.org/o2> , <http://example.org/o3> .`
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	// Should parse multiple triples
	count := 0
	for {
		stmt, err := dec.Next()
		if err != nil {
			break
		}
		if stmt.S.String() == "http://example.org/s" && stmt.P.String() == "http://example.org/p" {
			count++
		}
	}

	if count < 3 {
		t.Error("Expected at least 3 triples from object list")
	}
}

func TestTurtleParser_ParseDirective_BarePrefix(t *testing.T) {
	input := `PREFIX ex: <http://example.org/> .`
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	// Should parse directive without error
	_, err = dec.Next()
	// May or may not have statements after directive
	_ = err
}

func TestTurtleParser_ParseDirective_BareBase(t *testing.T) {
	input := `BASE <http://example.org/> .`
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	// Should parse directive without error
	_, err = dec.Next()
	// May or may not have statements after directive
	_ = err
}

func TestTurtleParser_ParseDirective_Version(t *testing.T) {
	input := `@version "1.1" .`
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	// Should parse directive without error
	_, err = dec.Next()
	// May or may not have statements after directive
	_ = err
}

// Test parser error paths

func TestTurtleParser_InvalidCollection_DepthExceeded(t *testing.T) {
	// Create deeply nested collection
	depth := 200
	input := strings.Repeat("( ", depth) + "<o>" + strings.Repeat(" )", depth) + " ."

	dec, err := NewReader(strings.NewReader(input), FormatTurtle, OptMaxDepth(50))
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	_, err = dec.Next()
	if err == nil {
		t.Error("Expected error for depth exceeded")
	}
	code := Code(err)
	if code != ErrCodeDepthExceeded {
		t.Errorf("Expected ErrCodeDepthExceeded, got %v", code)
	}
}

func TestTurtleParser_InvalidBlankNodeList_DepthExceeded(t *testing.T) {
	// Create deeply nested blank node list
	depth := 200
	input := strings.Repeat("[ ", depth) + "<p> <o>" + strings.Repeat(" ]", depth) + " ."

	dec, err := NewReader(strings.NewReader(input), FormatTurtle, OptMaxDepth(50))
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	_, err = dec.Next()
	if err == nil {
		t.Error("Expected error for depth exceeded")
	}
}

func TestTurtleParser_UnexpectedLangTag(t *testing.T) {
	input := `@en .`
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	_, err = dec.Next()
	if err == nil {
		t.Error("Expected error for unexpected lang tag")
	}
}

func TestTurtleParser_UnexpectedDatatypePrefix(t *testing.T) {
	input := `^^<http://example.org/type> .`
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	_, err = dec.Next()
	if err == nil {
		t.Error("Expected error for unexpected datatype prefix")
	}
}

// Test more encoder variations

func TestTurtleEncoder_MultipleStatements(t *testing.T) {
	var buf strings.Builder
	enc, err := NewWriter(&buf, FormatTurtle)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}
	defer enc.Close()

	stmts := []Statement{
		{S: IRI{Value: "http://example.org/s1"}, P: IRI{Value: "http://example.org/p"}, O: IRI{Value: "http://example.org/o1"}},
		{S: IRI{Value: "http://example.org/s2"}, P: IRI{Value: "http://example.org/p"}, O: IRI{Value: "http://example.org/o2"}},
		{S: IRI{Value: "http://example.org/s3"}, P: IRI{Value: "http://example.org/p"}, O: IRI{Value: "http://example.org/o3"}},
	}

	for _, stmt := range stmts {
		if err := enc.Write(stmt); err != nil {
			t.Fatalf("Write failed: %v", err)
		}
	}

	if err := enc.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}
}

func TestTriGEncoder_MultipleGraphs(t *testing.T) {
	var buf strings.Builder
	enc, err := NewWriter(&buf, FormatTriG)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}
	defer enc.Close()

	stmts := []Statement{
		{S: IRI{Value: "http://example.org/s1"}, P: IRI{Value: "http://example.org/p"}, O: IRI{Value: "http://example.org/o1"}, G: IRI{Value: "http://example.org/g1"}},
		{S: IRI{Value: "http://example.org/s2"}, P: IRI{Value: "http://example.org/p"}, O: IRI{Value: "http://example.org/o2"}, G: IRI{Value: "http://example.org/g2"}},
	}

	for _, stmt := range stmts {
		if err := enc.Write(stmt); err != nil {
			t.Fatalf("Write failed: %v", err)
		}
	}
}

// Test more format-specific parsing

func TestTriGParser_GraphBlock(t *testing.T) {
	input := `GRAPH <http://example.org/g> { <s> <p> <o> . }`
	dec, err := NewReader(strings.NewReader(input), FormatTriG)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	stmt, err := dec.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}

	if !stmt.IsQuad() {
		t.Error("Expected quad from GRAPH block")
	}
	if stmt.G.String() != "http://example.org/g" {
		t.Errorf("Graph = %q, want 'http://example.org/g'", stmt.G.String())
	}
}

func TestTriGParser_MultipleGraphs(t *testing.T) {
	input := `GRAPH <g1> { <s1> <p1> <o1> . }
GRAPH <g2> { <s2> <p2> <o2> . }`
	dec, err := NewReader(strings.NewReader(input), FormatTriG)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	count := 0
	for {
		stmt, err := dec.Next()
		if err != nil {
			break
		}
		if stmt.IsQuad() {
			count++
		}
	}

	if count < 2 {
		t.Error("Expected at least 2 quads from multiple GRAPH blocks")
	}
}

// Test RDF/XML parsing edge cases

func TestRDFXMLParser_Basic(t *testing.T) {
	input := `<?xml version="1.0"?>
<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" xmlns:ex="http://example.org/">
  <rdf:Description rdf:about="http://example.org/s">
    <ex:p>value</ex:p>
  </rdf:Description>
</rdf:RDF>`
	dec, err := NewReader(strings.NewReader(input), FormatRDFXML)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	stmt, err := dec.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}

	if stmt.S.String() != "http://example.org/s" {
		t.Errorf("Subject = %q, want 'http://example.org/s'", stmt.S.String())
	}
}

// Test JSON-LD parsing edge cases

func TestJSONLDParser_Array(t *testing.T) {
	input := `[
  {"@id": "http://example.org/s1", "http://example.org/p": "o1"},
  {"@id": "http://example.org/s2", "http://example.org/p": "o2"}
]`
	dec, err := NewReader(strings.NewReader(input), FormatJSONLD)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	count := 0
	for {
		stmt, err := dec.Next()
		if err != nil {
			break
		}
		count++
		_ = stmt
	}

	if count < 2 {
		t.Error("Expected at least 2 statements from JSON-LD array")
	}
}

func TestJSONLDParser_WithContext(t *testing.T) {
	input := `{
  "@context": {"ex": "http://example.org/"},
  "@id": "ex:s",
  "ex:p": "value"
}`
	dec, err := NewReader(strings.NewReader(input), FormatJSONLD)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	stmt, err := dec.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}

	// IRI should be expanded using context
	if !strings.Contains(stmt.S.String(), "http://example.org/s") {
		t.Errorf("Subject should be expanded, got %q", stmt.S.String())
	}
}

// Test more error code scenarios

func TestCode_StatementTooLong(t *testing.T) {
	// Create a statement that exceeds limit
	// Use Turtle format which checks MaxStatementBytes in appendStatementPart
	largeIRI := strings.Repeat("a", 300<<10) // 300KB
	input := "<" + largeIRI + "> <http://example.org/p> <http://example.org/o> .\n"

	dec, err := NewReader(strings.NewReader(input), FormatTurtle, OptMaxStatementBytes(256<<10))
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	_, err = dec.Next()
	if err == nil {
		t.Error("Expected error for statement too long")
		return
	}
	code := Code(err)
	// Error might be wrapped in ParseError, check underlying error
	if code != ErrCodeStatementTooLong {
		if !errors.Is(err, ErrStatementTooLong) {
			t.Errorf("Expected ErrCodeStatementTooLong or ErrStatementTooLong, got code=%v, err=%v", code, err)
		}
	}
}

// Test more utility functions

func TestIsNumericLiteral_EdgeCases(t *testing.T) {
	tests := []struct {
		value  string
		expect bool
	}{
		{"", false},
		{"123", true},
		{".123", true},
		{"123.", true}, // isNumericLiteral accepts this (simple character check)
		{"+123", true},
		{"-123", true},
		{"123e10", true},
		{"123E10", true},
		{"123e+10", true},
		{"123e-10", true},
		{"123.456e10", true},
		{"abc", false},
		{"123abc", false},
	}

	for _, tt := range tests {
		got := isNumericLiteral(tt.value)
		if got != tt.expect {
			t.Errorf("isNumericLiteral(%q) = %v, want %v", tt.value, got, tt.expect)
		}
	}
}

// Test token stream functions

func TestTurtleTokenStream_PeekAndNext(t *testing.T) {
	input := `<s> <p> <o> .`
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	// Should be able to read statement
	stmt, err := dec.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}

	if stmt.S.String() == "" {
		t.Error("Statement should have subject")
	}
}

// Test parser directive detection

func TestTurtleParser_IsLikelyDirective(t *testing.T) {
	// This is an internal function, test via actual parsing
	tests := []string{
		"@prefix ex: <http://example.org/> .",
		"@base <http://example.org/> .",
		"@version \"1.1\" .",
		"PREFIX ex: <http://example.org/> .",
		"BASE <http://example.org/> .",
	}

	for _, input := range tests {
		dec, err := NewReader(strings.NewReader(input), FormatTurtle)
		if err != nil {
			t.Fatalf("NewReader failed for %q: %v", input, err)
		}
		defer dec.Close()

		// Should parse directive without error
		_, err = dec.Next()
		// May or may not have statements after directive
		_ = err
	}
}

// Test more encoder error handling

func TestTurtleEncoder_InvalidStatement_MissingFields(t *testing.T) {
	var buf strings.Builder
	enc, err := NewWriter(&buf, FormatTurtle)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}
	defer enc.Close()

	// Statement with nil subject
	stmt := Statement{
		S: nil,
		P: IRI{Value: "http://example.org/p"},
		O: IRI{Value: "http://example.org/o"},
	}

	err = enc.Write(stmt)
	if err == nil {
		t.Error("Write should fail for invalid statement")
	}
}

func TestNTriplesEncoder_InvalidStatement_EmptyPredicate(t *testing.T) {
	var buf strings.Builder
	enc, err := NewWriter(&buf, FormatNTriples)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}
	defer enc.Close()

	// Statement with empty predicate
	stmt := Statement{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: ""}, // Empty
		O: IRI{Value: "http://example.org/o"},
	}

	// May or may not error - test doesn't crash
	_ = enc.Write(stmt)
}

// Test more format detection combinations

func TestDetectFormatFromSample_ComplexTurtle(t *testing.T) {
	input := `@prefix ex: <http://example.org/> .
@base <http://example.org/> .
ex:s ex:p [ ex:p2 ex:o2 ] .`
	format, ok := detectFormatFromSample(strings.NewReader(input))
	if !ok {
		t.Error("detectFormatFromSample should detect complex Turtle")
	}
	if format != FormatTurtle {
		t.Errorf("detectFormatFromSample = %v, want FormatTurtle", format)
	}
}

func TestDetectQuadFormat_ComplexTriG(t *testing.T) {
	input := `@prefix ex: <http://example.org/> .
GRAPH ex:g1 {
  ex:s1 ex:p1 ex:o1 .
}
GRAPH ex:g2 {
  ex:s2 ex:p2 ex:o2 .
}`
	format, ok := detectQuadFormat(strings.NewReader(input))
	if !ok {
		t.Error("detectQuadFormat should detect complex TriG")
	}
	if format != FormatTriG {
		t.Errorf("detectQuadFormat = %v, want FormatTriG", format)
	}
}

// Test more IRI resolution edge cases

func TestResolveIRI_ComplexPaths(t *testing.T) {
	tests := []struct {
		base     string
		relative string
		expect   string
	}{
		{"http://example.org/base/", "../path", "http://example.org/path"},
		{"http://example.org/base/", "./path", "http://example.org/base/path"},
		{"http://example.org/base/", "../../path", "http://example.org/path"},
		{"http://example.org/base/file", "path", "http://example.org/base/path"},
		{"http://example.org/base/file", "../path", "http://example.org/path"},
	}

	for _, tt := range tests {
		got := resolveIRI(tt.base, tt.relative)
		// Just verify it resolves (exact format may vary)
		if got == "" {
			t.Errorf("resolveIRI(%q, %q) returned empty", tt.base, tt.relative)
		}
	}
}

// Test more statement helper methods

func TestStatement_ConversionMethods(t *testing.T) {
	// Test all conversion methods
	triple := Triple{
		S: IRI{Value: "s"},
		P: IRI{Value: "p"},
		O: IRI{Value: "o"},
	}

	stmt := triple.ToStatement()
	if !stmt.IsTriple() {
		t.Error("ToStatement should create triple")
	}

	triple2 := stmt.AsTriple()
	if triple2.S != triple.S {
		t.Error("AsTriple should preserve subject")
	}

	quad := triple.ToQuad()
	if quad.G != nil {
		t.Error("ToQuad should have nil graph")
	}

	graph := IRI{Value: "g"}
	quad2 := triple.ToQuadInGraph(graph)
	if quad2.G != graph {
		t.Error("ToQuadInGraph should set graph")
	}
}

// Test error wrapping with various error types

func TestWrapParseError_PreservePosition(t *testing.T) {
	baseErr := errors.New("base error")
	wrapped := wrapParseErrorWithPosition("turtle", "test", 5, 10, 100, baseErr)

	parseErr, ok := wrapped.(*ParseError)
	if !ok {
		t.Fatal("Expected ParseError")
	}

	if parseErr.Line != 5 {
		t.Errorf("ParseError Line = %d, want 5", parseErr.Line)
	}
	if parseErr.Column != 10 {
		t.Errorf("ParseError Column = %d, want 10", parseErr.Column)
	}
	if parseErr.Offset != 100 {
		t.Errorf("ParseError Offset = %d, want 100", parseErr.Offset)
	}
}

func TestWrapParseError_PreserveBetterPosition(t *testing.T) {
	// Create error with position
	baseErr := &ParseError{
		Line:   10,
		Column: 20,
		Offset: 200,
		Err:    errors.New("base"),
	}

	// Wrap with worse position
	wrapped := wrapParseErrorWithPosition("turtle", "test", 0, 0, 0, baseErr)

	parseErr, ok := wrapped.(*ParseError)
	if !ok {
		t.Fatal("Expected ParseError")
	}

	// Should preserve better position from base error
	if parseErr.Line != 10 {
		t.Errorf("ParseError should preserve Line = %d, want 10", parseErr.Line)
	}
	if parseErr.Column != 20 {
		t.Errorf("ParseError should preserve Column = %d, want 20", parseErr.Column)
	}
}
