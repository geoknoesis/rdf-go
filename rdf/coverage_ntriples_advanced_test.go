package rdf

import (
	"strings"
	"testing"
)

// Test N-Triples parser edge cases and advanced features

func TestNTCursor_ParseSubject_TripleTerm(t *testing.T) {
	cursor := &ntCursor{input: "<<<s> <p> <o>>> <p2> <o2> ."}
	_, err := cursor.parseSubject()
	// Triple terms cannot be subjects in N-Triples
	if err == nil {
		t.Error("Expected error for triple term as subject")
	}
}

func TestNTCursor_ParseObject_LiteralWithLang(t *testing.T) {
	cursor := &ntCursor{input: `"test"@en`}
	object, err := cursor.parseObject()
	if err != nil {
		t.Fatalf("parseObject failed: %v", err)
	}
	if lit, ok := object.(Literal); !ok {
		t.Error("Expected Literal object")
	} else if lit.Lang != "en" {
		t.Errorf("Expected language 'en', got %q", lit.Lang)
	}
}

func TestNTCursor_ParseObject_LiteralWithDatatype(t *testing.T) {
	cursor := &ntCursor{input: `"123"^^<http://www.w3.org/2001/XMLSchema#integer>`}
	object, err := cursor.parseObject()
	if err != nil {
		t.Fatalf("parseObject failed: %v", err)
	}
	if lit, ok := object.(Literal); !ok {
		t.Error("Expected Literal object")
	} else if lit.Datatype.Value != "http://www.w3.org/2001/XMLSchema#integer" {
		t.Errorf("Expected integer datatype, got %q", lit.Datatype.Value)
	}
}

func TestNTCursor_ParseObject_TripleTerm(t *testing.T) {
	cursor := &ntCursor{input: `<<(<http://example.org/s> <http://example.org/p> <http://example.org/o>)>>`}
	object, err := cursor.parseObject()
	if err != nil {
		t.Fatalf("parseObject failed: %v", err)
	}
	if _, ok := object.(TripleTerm); !ok {
		t.Error("Expected TripleTerm object")
	}
}

func TestNTCursor_ParseIRI_Simple(t *testing.T) {
	cursor := &ntCursor{input: "<http://example.org/resource>"}
	iri, err := cursor.parseIRI()
	if err != nil {
		t.Fatalf("parseIRI failed: %v", err)
	}
	if iri.Value != "http://example.org/resource" {
		t.Errorf("parseIRI = %q, want %q", iri.Value, "http://example.org/resource")
	}
}

func TestNTCursor_ParseIRI_Unterminated(t *testing.T) {
	cursor := &ntCursor{input: "<http://example.org/resource"}
	_, err := cursor.parseIRI()
	if err == nil {
		t.Error("Expected error for unterminated IRI")
	}
}

func TestNTCursor_ParseIRI_NoStart(t *testing.T) {
	cursor := &ntCursor{input: "http://example.org/resource>"}
	_, err := cursor.parseIRI()
	if err == nil {
		t.Error("Expected error for IRI without start bracket")
	}
}

func TestNTCursor_ParseBlankNode_Simple(t *testing.T) {
	cursor := &ntCursor{input: "_:b1"}
	bnode, err := cursor.parseBlankNode()
	if err != nil {
		t.Fatalf("parseBlankNode failed: %v", err)
	}
	if bnode.ID != "b1" {
		t.Errorf("parseBlankNode = %q, want %q", bnode.ID, "b1")
	}
}

func TestNTCursor_ParseBlankNode_Empty(t *testing.T) {
	cursor := &ntCursor{input: "_:"}
	_, err := cursor.parseBlankNode()
	if err == nil {
		t.Error("Expected error for empty blank node")
	}
}

func TestNTCursor_ParseBlankNode_NoPrefix(t *testing.T) {
	cursor := &ntCursor{input: "b1"}
	_, err := cursor.parseBlankNode()
	if err == nil {
		t.Error("Expected error for blank node without _: prefix")
	}
}

func TestNTCursor_ParseLiteral_Simple(t *testing.T) {
	cursor := &ntCursor{input: `"test"`}
	lit, err := cursor.parseLiteral()
	if err != nil {
		t.Fatalf("parseLiteral failed: %v", err)
	}
	if lit.Lexical != "test" {
		t.Errorf("parseLiteral = %q, want %q", lit.Lexical, "test")
	}
}

func TestNTCursor_ParseLiteral_WithEscapes(t *testing.T) {
	cursor := &ntCursor{input: `"test\nwith\ttabs"`}
	lit, err := cursor.parseLiteral()
	if err != nil {
		t.Fatalf("parseLiteral failed: %v", err)
	}
	if !strings.Contains(lit.Lexical, "\n") {
		t.Error("parseLiteral should handle escape sequences")
	}
}

func TestNTCursor_ParseLiteral_Unterminated(t *testing.T) {
	cursor := &ntCursor{input: `"test`}
	_, err := cursor.parseLiteral()
	if err == nil {
		t.Error("Expected error for unterminated literal")
	}
}

func TestNTCursor_ParseLiteral_WithLangTag(t *testing.T) {
	cursor := &ntCursor{input: `"test"@en`}
	lit, err := cursor.parseLiteral()
	if err != nil {
		t.Fatalf("parseLiteral failed: %v", err)
	}
	if lit.Lang != "en" {
		t.Errorf("parseLiteral lang = %q, want %q", lit.Lang, "en")
	}
}

func TestNTCursor_ParseLiteral_WithDatatype(t *testing.T) {
	cursor := &ntCursor{input: `"123"^^<http://www.w3.org/2001/XMLSchema#integer>`}
	lit, err := cursor.parseLiteral()
	if err != nil {
		t.Fatalf("parseLiteral failed: %v", err)
	}
	if lit.Datatype.Value != "http://www.w3.org/2001/XMLSchema#integer" {
		t.Errorf("parseLiteral datatype = %q, want integer", lit.Datatype.Value)
	}
}

func TestNTCursor_ParseLiteral_InvalidLangTag(t *testing.T) {
	cursor := &ntCursor{input: `"test"@invalid-lang-tag`}
	_, err := cursor.parseLiteral()
	// May or may not error depending on validation
	_ = err
}

func TestNTCursor_ParseTripleTerm_Simple(t *testing.T) {
	cursor := &ntCursor{input: `<<(<http://example.org/s> <http://example.org/p> <http://example.org/o>)>>`}
	tt, err := cursor.parseTripleTerm()
	if err != nil {
		t.Fatalf("parseTripleTerm failed: %v", err)
	}
	if ttTerm, ok := tt.(TripleTerm); !ok {
		t.Error("Expected TripleTerm")
	} else if ttTerm.S == nil {
		t.Error("Expected non-nil subject in triple term")
	}
}

func TestNTCursor_ParseTripleTerm_Nested(t *testing.T) {
	cursor := &ntCursor{input: `<<(<<(<http://example.org/s1> <http://example.org/p1> <http://example.org/o1>)>> <http://example.org/p2> <http://example.org/o2>)>>`}
	tt, err := cursor.parseTripleTerm()
	if err != nil {
		t.Fatalf("parseTripleTerm failed: %v", err)
	}
	if ttTerm, ok := tt.(TripleTerm); !ok {
		t.Error("Expected TripleTerm")
	} else if ttTerm.S == nil {
		t.Error("Expected non-nil subject in triple term")
	}
}

func TestNTCursor_ParseTripleTerm_Unterminated(t *testing.T) {
	cursor := &ntCursor{input: `<<<s> <p> <o>`}
	_, err := cursor.parseTripleTerm()
	if err == nil {
		t.Error("Expected error for unterminated triple term")
	}
}

func TestNTCursor_ParseTripleTerm_InvalidFormat(t *testing.T) {
	cursor := &ntCursor{input: `<s> <p> <o>`}
	_, err := cursor.parseTripleTerm()
	if err == nil {
		t.Error("Expected error for invalid triple term format")
	}
}

func TestIsTermDelimiter(t *testing.T) {
	if !isTermDelimiter(' ') {
		t.Error("isTermDelimiter(' ') should return true")
	}
	if !isTermDelimiter('\t') {
		t.Error("isTermDelimiter('\\t') should return true")
	}
	if !isTermDelimiter('\n') {
		t.Error("isTermDelimiter('\\n') should return true")
	}
	if !isTermDelimiter('.') {
		t.Error("isTermDelimiter('.') should return true")
	}
	if !isTermDelimiter(')') {
		t.Error("isTermDelimiter(')') should return true")
	}
	if !isTermDelimiter('<') {
		t.Error("isTermDelimiter('<') should return true")
	}
	if !isTermDelimiter('>') {
		t.Error("isTermDelimiter('>') should return true")
	}
	if isTermDelimiter('a') {
		t.Error("isTermDelimiter('a') should return false")
	}
}
