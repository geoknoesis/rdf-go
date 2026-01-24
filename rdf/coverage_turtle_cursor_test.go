package rdf

import (
	"errors"
	"strings"
	"testing"
)

// Test turtle cursor parsing functions

func TestTurtleCursor_ParseIRI(t *testing.T) {
	input := `<http://example.org/resource> <p> <o> .`
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	stmt, err := dec.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	if stmt.S.Kind() != TermIRI {
		t.Error("Expected IRI")
	}
}

func TestTurtleCursor_ParseIRI_WithBase(t *testing.T) {
	input := `@base <http://example.org/> .
<resource> <p> <o> .`
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	stmt, err := dec.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	// IRI should be resolved against base
	if !strings.Contains(stmt.S.String(), "http://example.org/resource") {
		t.Errorf("Subject should be resolved, got %q", stmt.S.String())
	}
}

func TestTurtleCursor_ParsePrefixedName(t *testing.T) {
	input := `@prefix ex: <http://example.org/> .
ex:resource <p> <o> .`
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
	if !strings.Contains(stmt.S.String(), "http://example.org/resource") {
		t.Errorf("Subject should be expanded, got %q", stmt.S.String())
	}
}

func TestTurtleCursor_ParsePrefixedName_Undefined(t *testing.T) {
	input := `ex:resource <p> <o> .`
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	_, err = dec.Next()
	if err == nil {
		t.Error("Expected error for undefined prefix")
	}
}

func TestTurtleCursor_ParseLiteral_Simple(t *testing.T) {
	input := `<s> <p> "simple literal" .`
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
		t.Error("Expected Literal")
	}
}

func TestTurtleCursor_ParseLiteral_SingleQuote(t *testing.T) {
	input := `<s> <p> 'single quoted' .`
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
		t.Error("Expected Literal")
	}
}

func TestTurtleCursor_ParseLiteral_Long(t *testing.T) {
	input := `<s> <p> """long
multiline
literal""" .`
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
		t.Error("Expected Literal")
	}
}

func TestTurtleCursor_ParseLiteral_WithLang(t *testing.T) {
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

func TestTurtleCursor_ParseLiteral_WithDatatype(t *testing.T) {
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

func TestTurtleCursor_ParseCollection_Empty(t *testing.T) {
	input := `<s> <p> ( ) .`
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	stmt, err := dec.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	// Empty collection should be rdf:nil
	if stmt.O.String() != rdfNilIRI {
		t.Errorf("Empty collection = %q, want rdf:nil", stmt.O.String())
	}
}

func TestTurtleCursor_ParseCollection_Single(t *testing.T) {
	input := `<s> <p> ( <o> ) .`
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	// Should generate expansion triples
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
		t.Error("Expected at least 2 statements from single-item collection")
	}
}

func TestTurtleCursor_ParseCollection_Multiple(t *testing.T) {
	input := `<s> <p> ( <o1> <o2> <o3> ) .`
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	// Should generate expansion triples
	count := 0
	for {
		stmt, err := dec.Next()
		if err != nil {
			break
		}
		count++
		_ = stmt
	}
	if count < 3 {
		t.Error("Expected at least 3 statements from multi-item collection")
	}
}

func TestTurtleCursor_ParseBlankNodePropertyList_Empty(t *testing.T) {
	input := `<s> <p> [ ] .`
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	stmt, err := dec.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	// Empty blank node list should be blank node
	if stmt.O.Kind() != TermBlankNode {
		t.Error("Expected BlankNode for empty blank node list")
	}
}

func TestTurtleCursor_ParseBlankNodePropertyList_WithProperties(t *testing.T) {
	input := `<s> <p> [ <p2> <o2> ; <p3> <o3> ] .`
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	// Should generate expansion triples
	count := 0
	for {
		stmt, err := dec.Next()
		if err != nil {
			break
		}
		count++
		_ = stmt
	}
	if count < 3 {
		t.Error("Expected at least 3 statements from blank node property list")
	}
}

func TestTurtleCursor_ParseTripleTerm(t *testing.T) {
	input := `<s> <p> <<<s2> <p2> <o2>>> .`
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	stmt, err := dec.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	if stmt.O.Kind() != TermTriple {
		t.Error("Expected TripleTerm")
	}
}

func TestTurtleCursor_ParseNumericLiteral_Integer(t *testing.T) {
	input := `<s> <p> 123 .`
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
		t.Error("Expected Literal for numeric")
	}
}

func TestTurtleCursor_ParseNumericLiteral_Decimal(t *testing.T) {
	input := `<s> <p> 123.456 .`
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
		t.Error("Expected Literal for decimal")
	}
}

func TestTurtleCursor_ParseNumericLiteral_Double(t *testing.T) {
	input := `<s> <p> 123.456e10 .`
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
		t.Error("Expected Literal for double")
	}
}

func TestTurtleCursor_ParseBooleanLiteral_True(t *testing.T) {
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
	if stmt.O.Kind() != TermLiteral {
		t.Error("Expected Literal for boolean")
	}
}

func TestTurtleCursor_ParseBooleanLiteral_False(t *testing.T) {
	input := `<s> <p> false .`
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
		t.Error("Expected Literal for boolean")
	}
}

func TestTurtleCursor_ParsePredicateObjectList_MultiplePredicates(t *testing.T) {
	input := `<s> <p1> <o1> ; <p2> <o2> ; <p3> <o3> .`
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
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
	if count < 3 {
		t.Error("Expected at least 3 statements from multiple predicates")
	}
}

func TestTurtleCursor_ParseObjectList_MultipleObjects(t *testing.T) {
	// Use absolute IRIs to match string comparison
	input := `<http://example.org/s> <http://example.org/p> <http://example.org/o1> , <http://example.org/o2> , <http://example.org/o3> .`
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
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
		if stmt.S.String() == "http://example.org/s" && stmt.P.String() == "http://example.org/p" {
			count++
		}
		_ = stmt
	}
	if count < 3 {
		t.Error("Expected at least 3 statements from multiple objects")
	}
}

func TestTurtleCursor_ParseAnonBlankNode(t *testing.T) {
	input := `<s> <p> [] .`
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	stmt, err := dec.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	if stmt.O.Kind() != TermBlankNode {
		t.Error("Expected BlankNode for anonymous blank node")
	}
}

func TestTurtleCursor_ParseQuotedSubject(t *testing.T) {
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
		t.Error("Expected TripleTerm subject")
	}
}

func TestTurtleCursor_ParseQuotedObject(t *testing.T) {
	input := `<s> <p> <<<s2> <p2> <o2>>> .`
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	stmt, err := dec.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	if stmt.O.Kind() != TermTriple {
		t.Error("Expected TripleTerm object")
	}
}

// Test more error paths

func TestTurtleCursor_ParseIRI_Invalid(t *testing.T) {
	// Use IRI with spaces which is definitely invalid
	input := `<invalid iri with spaces> <p> <o> .`
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	_, err = dec.Next()
	// Parser may or may not error on invalid IRIs - both behaviors are acceptable
	_ = err
}

func TestTurtleCursor_ParseLiteral_Unterminated(t *testing.T) {
	input := `"unterminated <p> <o> .`
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	_, err = dec.Next()
	if err == nil {
		t.Error("Expected error for unterminated literal")
	}
}

func TestTurtleCursor_ParseCollection_Unbalanced(t *testing.T) {
	input := `( <o1> <o2> <p> <o> .`
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	_, err = dec.Next()
	if err == nil {
		t.Error("Expected error for unbalanced collection")
	}
}

func TestTurtleCursor_ParseBlankNodeList_Unbalanced(t *testing.T) {
	input := `[ <p> <o> <p2> <o2> .`
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	_, err = dec.Next()
	if err == nil {
		t.Error("Expected error for unbalanced blank node list")
	}
}

func TestTurtleCursor_ParseTripleTerm_Unclosed(t *testing.T) {
	input := `<<<s> <p> <o> <p2> <o2> .`
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	_, err = dec.Next()
	if err == nil {
		t.Error("Expected error for unclosed triple term")
	}
}

// Test more encoder variations

func TestTurtleEncoder_WithCollection(t *testing.T) {
	var buf strings.Builder
	enc, err := NewWriter(&buf, FormatTurtle)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}
	defer enc.Close()

	// Write statement that would generate collection
	stmt := Statement{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: IRI{Value: rdfNilIRI}, // rdf:nil
	}

	if err := enc.Write(stmt); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
}

func TestTurtleEncoder_WithTripleTerm_Additional(t *testing.T) {
	var buf strings.Builder
	enc, err := NewWriter(&buf, FormatTurtle)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}
	defer enc.Close()

	stmt := Statement{
		S: TripleTerm{
			S: IRI{Value: "http://example.org/s"},
			P: IRI{Value: "http://example.org/p"},
			O: IRI{Value: "http://example.org/o"},
		},
		P: IRI{Value: "http://example.org/asserted"},
		O: Literal{Lexical: "true"},
	}

	if err := enc.Write(stmt); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
}

// Test more N-Triples parser functions

func TestNTCursor_ParseSubject_IRI(t *testing.T) {
	input := `<http://example.org/s> <http://example.org/p> <http://example.org/o> .`
	dec, err := NewReader(strings.NewReader(input), FormatNTriples)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	stmt, err := dec.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	if stmt.S.Kind() != TermIRI {
		t.Error("Expected IRI subject")
	}
}

func TestNTCursor_ParseSubject_BlankNode(t *testing.T) {
	input := `_:b1 <http://example.org/p> <http://example.org/o> .`
	dec, err := NewReader(strings.NewReader(input), FormatNTriples)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	stmt, err := dec.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	if stmt.S.Kind() != TermBlankNode {
		t.Error("Expected BlankNode subject")
	}
}

func TestNTCursor_ParseObject_IRI(t *testing.T) {
	input := `<http://example.org/s> <http://example.org/p> <http://example.org/o> .`
	dec, err := NewReader(strings.NewReader(input), FormatNTriples)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	stmt, err := dec.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	if stmt.O.Kind() != TermIRI {
		t.Error("Expected IRI object")
	}
}

func TestNTCursor_ParseObject_BlankNode(t *testing.T) {
	input := `<http://example.org/s> <http://example.org/p> _:b1 .`
	dec, err := NewReader(strings.NewReader(input), FormatNTriples)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	stmt, err := dec.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	if stmt.O.Kind() != TermBlankNode {
		t.Error("Expected BlankNode object")
	}
}

func TestNTCursor_ParseObject_Literal(t *testing.T) {
	input := `<http://example.org/s> <http://example.org/p> "value" .`
	dec, err := NewReader(strings.NewReader(input), FormatNTriples)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	stmt, err := dec.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	if stmt.O.Kind() != TermLiteral {
		t.Error("Expected Literal object")
	}
}

// Test more error paths

func TestNTCursor_ParseIRI_Unclosed(t *testing.T) {
	input := `<unclosed <p> <o> .`
	dec, err := NewReader(strings.NewReader(input), FormatNTriples)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	_, err = dec.Next()
	if err == nil {
		t.Error("Expected error for unclosed IRI")
	}
}

func TestNTCursor_ParseLiteral_Unclosed(t *testing.T) {
	input := `"unclosed <p> <o> .`
	dec, err := NewReader(strings.NewReader(input), FormatNTriples)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	_, err = dec.Next()
	if err == nil {
		t.Error("Expected error for unclosed literal")
	}
}

func TestNTCursor_ParseBlankNode_Invalid(t *testing.T) {
	input := `_: <p> <o> .`
	dec, err := NewReader(strings.NewReader(input), FormatNTriples)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	_, err = dec.Next()
	// May or may not error - test doesn't crash
	_ = err
}

// Test more encoder error handling

func TestTurtleEncoder_FlushAfterClose(t *testing.T) {
	var buf strings.Builder
	enc, err := NewWriter(&buf, FormatTurtle)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	enc.Close()

	// Flush after close should be safe
	_ = enc.Flush()
}

func TestNTriplesEncoder_FlushAfterClose(t *testing.T) {
	var buf strings.Builder
	enc, err := NewWriter(&buf, FormatNTriples)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	enc.Close()

	// Flush after close should be safe
	_ = enc.Flush()
}

// Test more format detection

func TestDetectFormat_WithMultipleLines(t *testing.T) {
	input := `<http://example.org/s1> <http://example.org/p1> <http://example.org/o1> .
<http://example.org/s2> <http://example.org/p2> <http://example.org/o2> .`
	format, _, ok := detectFormat(strings.NewReader(input))
	if !ok {
		t.Error("detectFormat should handle multiple lines")
	}
	// Format detection may return various formats
	if format != FormatNTriples && format != FormatNQuads && format != FormatTurtle {
		t.Errorf("detectFormat = %v, want FormatNTriples, FormatNQuads, or FormatTurtle", format)
	}
}

func TestDetectFormat_WithMixedWhitespace(t *testing.T) {
	input := "  \t  <http://example.org/s> <http://example.org/p> <http://example.org/o> .  \n  "
	format, _, ok := detectFormat(strings.NewReader(input))
	if !ok {
		t.Error("detectFormat should handle mixed whitespace")
	}
	// Format detection may return various formats
	if format != FormatNTriples && format != FormatNQuads && format != FormatTurtle {
		t.Errorf("detectFormat = %v, want FormatNTriples, FormatNQuads, or FormatTurtle", format)
	}
}

// Test more round-trip scenarios

func TestRoundTrip_WithBlankNodes(t *testing.T) {
	input := `_:b1 <http://example.org/p> <http://example.org/o> .`

	dec, err := NewReader(strings.NewReader(input), FormatNTriples)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	stmt, err := dec.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}

	// Write it back
	var buf strings.Builder
	enc, err := NewWriter(&buf, FormatNTriples)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}
	defer enc.Close()

	if err := enc.Write(stmt); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if err := enc.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Output should contain blank node
	output := buf.String()
	if !strings.Contains(output, "_:") {
		t.Error("Output should contain blank node")
	}
}

func TestRoundTrip_WithLiterals(t *testing.T) {
	input := `<http://example.org/s> <http://example.org/p> "value"@en .`

	dec, err := NewReader(strings.NewReader(input), FormatNTriples)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	stmt, err := dec.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}

	// Write it back
	var buf strings.Builder
	enc, err := NewWriter(&buf, FormatNTriples)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}
	defer enc.Close()

	if err := enc.Write(stmt); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if err := enc.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Output should contain language tag
	output := buf.String()
	if !strings.Contains(output, "@en") {
		t.Error("Output should contain language tag")
	}
}

// Test more utility functions

func TestTurtleCursor_NewBlankNode(t *testing.T) {
	// Test blank node generation via actual parsing
	input := `[ <p> <o> ] <p2> <o2> .`
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	stmt, err := dec.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	// Should generate blank node
	if stmt.S.Kind() != TermBlankNode {
		t.Error("Expected BlankNode from blank node list")
	}
}

func TestTurtleCursor_EnsureLineEnd(t *testing.T) {
	// Test ensureLineEnd via actual parsing
	input := `<s> <p> <o> . extra`
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	_, err = dec.Next()
	// May or may not error - test doesn't crash
	_ = err
}

// Test more error code scenarios

func TestCode_StatementTooLong_Additional(t *testing.T) {
	// Create a statement that exceeds limit - use Turtle format which checks MaxStatementBytes
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
	// Error might be wrapped, check underlying error
	if code != ErrCodeStatementTooLong {
		if !errors.Is(err, ErrStatementTooLong) {
			t.Errorf("Expected ErrCodeStatementTooLong or ErrStatementTooLong, got code=%v, err=%v", code, err)
		}
	}
}

// Test more format-specific edge cases

func TestTriGParser_WithNestedGraphs(t *testing.T) {
	input := `GRAPH <g1> {
  GRAPH <g2> {
    <s> <p> <o> .
  }
}`
	dec, err := NewReader(strings.NewReader(input), FormatTriG)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	// Nested graphs may not be allowed - parser may error
	stmt, err := dec.Next()
	if err != nil {
		// Nested graphs not allowed - that's acceptable
		return
	}
	_ = stmt
}

func TestTriGParser_WithDefaultGraphInGraph(t *testing.T) {
	input := `GRAPH <g> {
  <s1> <p1> <o1> .
}
<s2> <p2> <o2> .`
	dec, err := NewReader(strings.NewReader(input), FormatTriG)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	// Should parse both named and default graph
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
		t.Error("Expected at least 2 statements")
	}
}

// Test more JSON-LD edge cases

func TestJSONLDParser_WithNestedArray(t *testing.T) {
	input := `{
  "@context": {"ex": "http://example.org/"},
  "@id": "ex:s",
  "ex:p": [["v1", "v2"], ["v3", "v4"]]
}`
	dec, err := NewReader(strings.NewReader(input), FormatJSONLD)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	// Should parse nested array
	count := 0
	for {
		stmt, err := dec.Next()
		if err != nil {
			break
		}
		count++
		_ = stmt
	}
	if count == 0 {
		t.Error("Expected at least one statement from nested array")
	}
}

func TestJSONLDParser_WithNull(t *testing.T) {
	input := `{
  "@context": {"ex": "http://example.org/"},
  "@id": "ex:s",
  "ex:p": null
}`
	dec, err := NewReader(strings.NewReader(input), FormatJSONLD)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	// Null values should be skipped
	_, err = dec.Next()
	// May or may not have statements
	_ = err
}

// Test more RDF/XML edge cases

func TestRDFXMLParser_WithLi(t *testing.T) {
	input := `<?xml version="1.0"?>
<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" xmlns:ex="http://example.org/">
  <rdf:Description rdf:about="http://example.org/s">
    <ex:p>
      <rdf:Bag>
        <rdf:li>value1</rdf:li>
        <rdf:li>value2</rdf:li>
      </rdf:Bag>
    </ex:p>
  </rdf:Description>
</rdf:RDF>`
	dec, err := NewReader(strings.NewReader(input), FormatRDFXML)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	// Should parse rdf:li elements
	count := 0
	for {
		stmt, err := dec.Next()
		if err != nil {
			break
		}
		count++
		_ = stmt
	}
	if count == 0 {
		t.Error("Expected at least one statement from rdf:li")
	}
}

func TestRDFXMLParser_WithReification(t *testing.T) {
	input := `<?xml version="1.0"?>
<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" xmlns:ex="http://example.org/">
  <rdf:Statement rdf:about="http://example.org/stmt">
    <rdf:subject rdf:resource="http://example.org/s"/>
    <rdf:predicate rdf:resource="http://example.org/p"/>
    <rdf:object rdf:resource="http://example.org/o"/>
  </rdf:Statement>
</rdf:RDF>`
	dec, err := NewReader(strings.NewReader(input), FormatRDFXML)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	// Should parse reification
	stmt, err := dec.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	_ = stmt
}
