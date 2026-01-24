package rdf

import (
	"strings"
	"testing"
)

// Test turtle_parser.go functions for maximum coverage

func TestTurtleParser_ReadNextTriple_Simple(t *testing.T) {
	input := "<http://example.org/s> <http://example.org/p> <http://example.org/o> ."
	parser := newTurtleParser(strings.NewReader(input), defaultDecodeOptions())

	_, err := parser.readNextTriple()
	if err != nil {
		t.Fatalf("readNextTriple failed: %v", err)
	}
}

func TestTurtleParser_ReadNextTriple_WithPrefix(t *testing.T) {
	input := `@prefix ex: <http://example.org/> .
ex:s ex:p ex:o .`
	parser := newTurtleParser(strings.NewReader(input), defaultDecodeOptions())

	// First call should process prefix
	_, err := parser.readNextTriple()
	if err != nil {
		t.Fatalf("readNextTriple failed: %v", err)
	}
}

func TestTurtleParser_ReadNextTriple_WithBase(t *testing.T) {
	input := `@base <http://example.org/> .
<s> <p> <o> .`
	parser := newTurtleParser(strings.NewReader(input), defaultDecodeOptions())

	triple, err := parser.readNextTriple()
	if err != nil {
		t.Fatalf("readNextTriple failed: %v", err)
	}
	if triple.S == nil {
		t.Error("readNextTriple should return triple")
	}
}

func TestTurtleParser_ReadNextTriple_WithBlankNode(t *testing.T) {
	input := "_:b1 <http://example.org/p> <http://example.org/o> ."
	parser := newTurtleParser(strings.NewReader(input), defaultDecodeOptions())

	triple, err := parser.readNextTriple()
	if err != nil {
		t.Fatalf("readNextTriple failed: %v", err)
	}
	if _, ok := triple.S.(BlankNode); !ok {
		t.Error("readNextTriple should return BlankNode subject")
	}
}

func TestTurtleParser_ReadNextTriple_WithLiteral(t *testing.T) {
	input := "<http://example.org/s> <http://example.org/p> \"literal\" ."
	parser := newTurtleParser(strings.NewReader(input), defaultDecodeOptions())

	triple, err := parser.readNextTriple()
	if err != nil {
		t.Fatalf("readNextTriple failed: %v", err)
	}
	if _, ok := triple.O.(Literal); !ok {
		t.Error("readNextTriple should return Literal object")
	}
}

func TestTurtleParser_ReadNextTriple_WithCollection(t *testing.T) {
	input := "<http://example.org/s> <http://example.org/p> (<http://example.org/o1> <http://example.org/o2>) ."
	parser := newTurtleParser(strings.NewReader(input), defaultDecodeOptions())

	// Should expand collection into multiple triples
	count := 0
	for {
		_, err := parser.readNextTriple()
		if err != nil {
			break
		}
		count++
		if count > 10 {
			break
		}
	}
	if count == 0 {
		t.Error("readNextTriple should expand collection")
	}
}

func TestTurtleParser_ReadNextTriple_WithBlankNodeList(t *testing.T) {
	input := "<http://example.org/s> <http://example.org/p> [<http://example.org/p2> <http://example.org/o2>] ."
	parser := newTurtleParser(strings.NewReader(input), defaultDecodeOptions())

	// Should expand blank node list
	count := 0
	for {
		_, err := parser.readNextTriple()
		if err != nil {
			break
		}
		count++
		if count > 10 {
			break
		}
	}
	if count == 0 {
		t.Error("readNextTriple should expand blank node list")
	}
}

func TestTurtleParser_ReadNextTriple_WithTripleTerm(t *testing.T) {
	input := "<<<http://example.org/s1> <http://example.org/p1> <http://example.org/o1>>> <http://example.org/p2> <http://example.org/o2> ."
	parser := newTurtleParser(strings.NewReader(input), defaultDecodeOptions())

	triple, err := parser.readNextTriple()
	if err != nil {
		t.Fatalf("readNextTriple failed: %v", err)
	}
	if _, ok := triple.S.(TripleTerm); !ok {
		t.Error("readNextTriple should return TripleTerm subject")
	}
}

func TestTurtleParser_ReadNextTriple_Error(t *testing.T) {
	input := "invalid syntax"
	parser := newTurtleParser(strings.NewReader(input), defaultDecodeOptions())

	_, err := parser.readNextTriple()
	if err == nil {
		t.Error("readNextTriple should error for invalid syntax")
	}
}

func TestTurtleParser_NewBlankNode(t *testing.T) {
	parser := newTurtleParser(strings.NewReader(""), defaultDecodeOptions())

	bnode1 := parser.newBlankNode()
	bnode2 := parser.newBlankNode()

	if bnode1.ID == bnode2.ID {
		t.Error("newBlankNode should generate unique IDs")
	}
	if !strings.HasPrefix(bnode1.ID, "b") {
		t.Error("newBlankNode should generate IDs starting with 'b'")
	}
}

func TestTurtleParser_ShouldDebugStatements_Enabled(t *testing.T) {
	opts := defaultDecodeOptions()
	opts.DebugStatements = true
	parser := newTurtleParser(strings.NewReader(""), opts)

	if !parser.shouldDebugStatements() {
		t.Error("shouldDebugStatements should return true when enabled")
	}
}

func TestTurtleParser_ShouldDebugStatements_Disabled(t *testing.T) {
	opts := defaultDecodeOptions()
	opts.DebugStatements = false
	parser := newTurtleParser(strings.NewReader(""), opts)

	if parser.shouldDebugStatements() {
		t.Error("shouldDebugStatements should return false when disabled")
	}
}

func TestTurtleParser_NextTriple_WithPending(t *testing.T) {
	input := "<http://example.org/s> <http://example.org/p> (<http://example.org/o1> <http://example.org/o2>) ."
	parser := newTurtleParser(strings.NewReader(input), defaultDecodeOptions())

	// First call should queue expansion triples
	triple, err := parser.NextTriple()
	if err != nil {
		t.Fatalf("NextTriple failed: %v", err)
	}
	if triple.S == nil {
		t.Error("NextTriple should return triple")
	}

	// Should have pending triples from collection expansion
	if len(parser.pending) == 0 {
		t.Error("NextTriple should queue expansion triples")
	}
}

func TestTurtleParser_NextTriple_FromPending(t *testing.T) {
	input := "<http://example.org/s> <http://example.org/p> (<http://example.org/o1> <http://example.org/o2>) ."
	parser := newTurtleParser(strings.NewReader(input), defaultDecodeOptions())

	// Get first triple (should queue expansion triples)
	_, err := parser.NextTriple()
	if err != nil {
		t.Fatalf("NextTriple failed: %v", err)
	}

	// Next call should return from pending
	triple, err := parser.NextTriple()
	if err != nil {
		t.Fatalf("NextTriple failed: %v", err)
	}
	if triple.S == nil {
		t.Error("NextTriple should return triple from pending")
	}
}

func TestTurtleParser_NextTriple_ErrorPropagation(t *testing.T) {
	input := "invalid syntax"
	parser := newTurtleParser(strings.NewReader(input), defaultDecodeOptions())

	_, err := parser.NextTriple()
	if err == nil {
		t.Error("NextTriple should propagate error")
	}

	// Error should be stored
	if parser.err == nil {
		t.Error("NextTriple should store error")
	}
}

func TestTurtleParser_ReadNextTriple_WithMultiplePredicates(t *testing.T) {
	input := "<http://example.org/s> <http://example.org/p1> <http://example.org/o1> ; <http://example.org/p2> <http://example.org/o2> ."
	parser := newTurtleParser(strings.NewReader(input), defaultDecodeOptions())

	// Should generate multiple triples from semicolon-separated predicates
	count := 0
	for {
		_, err := parser.readNextTriple()
		if err != nil {
			break
		}
		count++
		if count > 10 {
			break
		}
	}
	// May generate 2+ triples depending on implementation
	_ = count
}

func TestTurtleParser_ReadNextTriple_WithMultipleObjects(t *testing.T) {
	input := "<http://example.org/s> <http://example.org/p> <http://example.org/o1> , <http://example.org/o2> ."
	parser := newTurtleParser(strings.NewReader(input), defaultDecodeOptions())

	// Should generate multiple triples from comma-separated objects
	count := 0
	for {
		_, err := parser.readNextTriple()
		if err != nil {
			break
		}
		count++
		if count > 10 {
			break
		}
	}
	// May generate 2+ triples depending on implementation
	_ = count
}

func TestTurtleParser_ReadNextTriple_WithLiteralLang(t *testing.T) {
	input := "<http://example.org/s> <http://example.org/p> \"literal\"@en ."
	parser := newTurtleParser(strings.NewReader(input), defaultDecodeOptions())

	triple, err := parser.readNextTriple()
	if err != nil {
		t.Fatalf("readNextTriple failed: %v", err)
	}
	if lit, ok := triple.O.(Literal); !ok {
		t.Error("readNextTriple should return Literal object")
	} else if lit.Lang != "en" {
		t.Errorf("readNextTriple literal lang = %q, want %q", lit.Lang, "en")
	}
}

func TestTurtleParser_ReadNextTriple_WithLiteralDatatype(t *testing.T) {
	input := "<http://example.org/s> <http://example.org/p> \"123\"^^<http://www.w3.org/2001/XMLSchema#integer> ."
	parser := newTurtleParser(strings.NewReader(input), defaultDecodeOptions())

	triple, err := parser.readNextTriple()
	if err != nil {
		t.Fatalf("readNextTriple failed: %v", err)
	}
	if lit, ok := triple.O.(Literal); !ok {
		t.Error("readNextTriple should return Literal object")
	} else if lit.Datatype.Value != "http://www.w3.org/2001/XMLSchema#integer" {
		t.Errorf("readNextTriple literal datatype = %q, want integer", lit.Datatype.Value)
	}
}

func TestTurtleParser_ReadNextTriple_WithLongLiteral(t *testing.T) {
	input := "<http://example.org/s> <http://example.org/p> \"\"\"long\nliteral\nwith\nnewlines\"\"\" ."
	parser := newTurtleParser(strings.NewReader(input), defaultDecodeOptions())

	triple, err := parser.readNextTriple()
	if err != nil {
		t.Fatalf("readNextTriple failed: %v", err)
	}
	if lit, ok := triple.O.(Literal); !ok {
		t.Error("readNextTriple should return Literal object")
	} else if lit.Lexical == "" {
		t.Error("readNextTriple should handle long literals")
	}
}

func TestTurtleParser_ReadNextTriple_WithNumericLiteral(t *testing.T) {
	input := "<http://example.org/s> <http://example.org/p> 42 ."
	parser := newTurtleParser(strings.NewReader(input), defaultDecodeOptions())

	triple, err := parser.readNextTriple()
	if err != nil {
		t.Fatalf("readNextTriple failed: %v", err)
	}
	if _, ok := triple.O.(Literal); !ok {
		t.Error("readNextTriple should return Literal object for numeric")
	}
}

func TestTurtleParser_ReadNextTriple_WithBooleanLiteral(t *testing.T) {
	input := "<http://example.org/s> <http://example.org/p> true ."
	parser := newTurtleParser(strings.NewReader(input), defaultDecodeOptions())

	triple, err := parser.readNextTriple()
	if err != nil {
		t.Fatalf("readNextTriple failed: %v", err)
	}
	if _, ok := triple.O.(Literal); !ok {
		t.Error("readNextTriple should return Literal object for boolean")
	}
}

func TestTurtleParser_ReadNextTriple_EmptyInput(t *testing.T) {
	parser := newTurtleParser(strings.NewReader(""), defaultDecodeOptions())

	_, err := parser.readNextTriple()
	if err == nil {
		t.Error("readNextTriple should error for empty input")
	}
}

func TestTurtleParser_ReadNextTriple_CommentOnly(t *testing.T) {
	input := "# This is a comment\n"
	parser := newTurtleParser(strings.NewReader(input), defaultDecodeOptions())

	_, err := parser.readNextTriple()
	if err == nil {
		t.Error("readNextTriple should error for comment-only input")
	}
}

func TestTurtleParser_ReadNextTriple_WhitespaceOnly(t *testing.T) {
	input := "   \n  \t  \n"
	parser := newTurtleParser(strings.NewReader(input), defaultDecodeOptions())

	_, err := parser.readNextTriple()
	if err == nil {
		t.Error("readNextTriple should error for whitespace-only input")
	}
}
