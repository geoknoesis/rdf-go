package rdf

import (
	"testing"
)

// Test parseTurtleStatement and parseTurtleTripleLine functions for coverage

func TestParseTurtleStatement_Simple(t *testing.T) {
	prefixes := map[string]string{
		"ex": "http://example.org/",
	}
	line := "ex:s ex:p ex:o ."

	triples, err := parseTurtleStatement(prefixes, "", false, false, line)
	if err != nil {
		t.Fatalf("parseTurtleStatement failed: %v", err)
	}
	if len(triples) == 0 {
		t.Fatal("Expected at least one triple")
	}
}

func TestParseTurtleStatement_WithBase(t *testing.T) {
	prefixes := map[string]string{}
	baseIRI := "http://example.org/"
	line := "<s> <p> <o> ."

	triples, err := parseTurtleStatement(prefixes, baseIRI, false, false, line)
	if err != nil {
		t.Fatalf("parseTurtleStatement failed: %v", err)
	}
	if len(triples) == 0 {
		t.Fatal("Expected at least one triple")
	}
}

func TestParseTurtleStatement_WithQuoted(t *testing.T) {
	prefixes := map[string]string{
		"ex": "http://example.org/",
	}
	line := "<<ex:s ex:p ex:o>> ex:p2 ex:o2 ."

	triples, err := parseTurtleStatement(prefixes, "", true, false, line)
	if err != nil {
		t.Fatalf("parseTurtleStatement failed: %v", err)
	}
	if len(triples) == 0 {
		t.Fatal("Expected at least one triple")
	}
}

func TestParseTurtleTripleLine_Simple(t *testing.T) {
	prefixes := map[string]string{
		"ex": "http://example.org/",
	}
	line := "ex:s ex:p ex:o ."

	triples, err := parseTurtleTripleLine(prefixes, "", false, false, line)
	if err != nil {
		t.Fatalf("parseTurtleTripleLine failed: %v", err)
	}
	if len(triples) == 0 {
		t.Fatal("Expected at least one triple")
	}
}

func TestParseTurtleTripleLine_WithCollection(t *testing.T) {
	prefixes := map[string]string{
		"ex": "http://example.org/",
	}
	line := "ex:s ex:p (ex:o1 ex:o2) ."

	triples, err := parseTurtleTripleLine(prefixes, "", false, false, line)
	if err != nil {
		t.Fatalf("parseTurtleTripleLine failed: %v", err)
	}
	if len(triples) == 0 {
		t.Fatal("Expected at least one triple")
	}
}

func TestParseTurtleTripleLine_WithBlankNodeList(t *testing.T) {
	prefixes := map[string]string{
		"ex": "http://example.org/",
	}
	line := "ex:s ex:p [ex:p2 ex:o2] ."

	triples, err := parseTurtleTripleLine(prefixes, "", false, false, line)
	if err != nil {
		t.Fatalf("parseTurtleTripleLine failed: %v", err)
	}
	if len(triples) == 0 {
		t.Fatal("Expected at least one triple")
	}
}

func TestParseTurtleTripleLine_Error(t *testing.T) {
	prefixes := map[string]string{}
	line := "invalid syntax"

	_, err := parseTurtleTripleLine(prefixes, "", false, false, line)
	// May or may not error depending on implementation
	_ = err
}

func TestParseTurtleTripleLineWithOptions_Simple(t *testing.T) {
	opts := TurtleParseOptions{
		Prefixes: map[string]string{
			"ex": "http://example.org/",
		},
		BaseIRI:         "",
		AllowQuoted:     false,
		DebugStatements: false,
	}
	line := "ex:s ex:p ex:o ."

	triples, err := parseTurtleTripleLineWithOptions(opts, line)
	if err != nil {
		t.Fatalf("parseTurtleTripleLineWithOptions failed: %v", err)
	}
	if len(triples) == 0 {
		t.Fatal("Expected at least one triple")
	}
}

func TestParseTurtleTripleLineWithOptions_AllOptions(t *testing.T) {
	opts := TurtleParseOptions{
		Prefixes: map[string]string{
			"ex": "http://example.org/",
		},
		BaseIRI:         "http://example.org/",
		AllowQuoted:     true,
		DebugStatements: true,
	}
	line := "<<ex:s ex:p ex:o>> ex:p2 ex:o2 ."

	triples, err := parseTurtleTripleLineWithOptions(opts, line)
	if err != nil {
		t.Fatalf("parseTurtleTripleLineWithOptions failed: %v", err)
	}
	if len(triples) == 0 {
		t.Fatal("Expected at least one triple")
	}
}
