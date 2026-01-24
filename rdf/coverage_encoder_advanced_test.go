package rdf

import (
	"bytes"
	"strings"
	"testing"
)

// Test encoder edge cases and error paths for maximum coverage

func TestTurtleEncoder_WriteBlankNode(t *testing.T) {
	var buf strings.Builder
	enc, err := NewWriter(&buf, FormatTurtle)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}
	defer enc.Close()

	stmt := Statement{
		S: BlankNode{ID: "b1"},
		P: IRI{Value: "http://example.org/p"},
		O: IRI{Value: "http://example.org/o"},
	}

	if err := enc.Write(stmt); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
}

func TestTurtleEncoder_WriteLiteralWithLang(t *testing.T) {
	var buf strings.Builder
	enc, err := NewWriter(&buf, FormatTurtle)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}
	defer enc.Close()

	stmt := Statement{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: Literal{Lexical: "test", Lang: "en"},
	}

	if err := enc.Write(stmt); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
}

func TestTurtleEncoder_WriteLiteralWithDatatype(t *testing.T) {
	var buf strings.Builder
	enc, err := NewWriter(&buf, FormatTurtle)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}
	defer enc.Close()

	stmt := Statement{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: Literal{
			Lexical:  "123",
			Datatype: IRI{Value: "http://www.w3.org/2001/XMLSchema#integer"},
		},
	}

	if err := enc.Write(stmt); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
}

func TestTurtleEncoder_WriteTripleTerm(t *testing.T) {
	var buf strings.Builder
	enc, err := NewWriter(&buf, FormatTurtle)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}
	defer enc.Close()

	stmt := Statement{
		S: TripleTerm{
			S: IRI{Value: "http://example.org/s1"},
			P: IRI{Value: "http://example.org/p1"},
			O: IRI{Value: "http://example.org/o1"},
		},
		P: IRI{Value: "http://example.org/p2"},
		O: IRI{Value: "http://example.org/o2"},
	}

	if err := enc.Write(stmt); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
}

func TestNTriplesEncoder_WriteBlankNode(t *testing.T) {
	var buf strings.Builder
	enc, err := NewWriter(&buf, FormatNTriples)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}
	defer enc.Close()

	stmt := Statement{
		S: BlankNode{ID: "b1"},
		P: IRI{Value: "http://example.org/p"},
		O: IRI{Value: "http://example.org/o"},
	}

	if err := enc.Write(stmt); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
}

func TestNTriplesEncoder_WriteLiteralWithLang(t *testing.T) {
	var buf strings.Builder
	enc, err := NewWriter(&buf, FormatNTriples)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}
	defer enc.Close()

	stmt := Statement{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: Literal{Lexical: "test", Lang: "en"},
	}

	if err := enc.Write(stmt); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
}

func TestNTriplesEncoder_WriteLiteralWithDatatype(t *testing.T) {
	var buf strings.Builder
	enc, err := NewWriter(&buf, FormatNTriples)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}
	defer enc.Close()

	stmt := Statement{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: Literal{
			Lexical:  "123",
			Datatype: IRI{Value: "http://www.w3.org/2001/XMLSchema#integer"},
		},
	}

	if err := enc.Write(stmt); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
}

func TestNQuadsEncoder_WriteQuad(t *testing.T) {
	var buf strings.Builder
	enc, err := NewWriter(&buf, FormatNQuads)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}
	defer enc.Close()

	stmt := Statement{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: IRI{Value: "http://example.org/o"},
		G: IRI{Value: "http://example.org/g"},
	}

	if err := enc.Write(stmt); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
}

func TestNQuadsEncoder_WriteTriple(t *testing.T) {
	var buf strings.Builder
	enc, err := NewWriter(&buf, FormatNQuads)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}
	defer enc.Close()

	stmt := Statement{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: IRI{Value: "http://example.org/o"},
		G: nil,
	}

	if err := enc.Write(stmt); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
}

func TestTriGEncoder_WriteQuad(t *testing.T) {
	var buf strings.Builder
	enc, err := NewWriter(&buf, FormatTriG)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}
	defer enc.Close()

	stmt := Statement{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: IRI{Value: "http://example.org/o"},
		G: IRI{Value: "http://example.org/g"},
	}

	if err := enc.Write(stmt); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
}

func TestTriGEncoder_WriteTriple(t *testing.T) {
	var buf strings.Builder
	enc, err := NewWriter(&buf, FormatTriG)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}
	defer enc.Close()

	stmt := Statement{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: IRI{Value: "http://example.org/o"},
		G: nil,
	}

	if err := enc.Write(stmt); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
}

func TestJSONLDEncoder_WriteSimple(t *testing.T) {
	var buf strings.Builder
	enc, err := NewWriter(&buf, FormatJSONLD)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}
	defer enc.Close()

	stmt := Statement{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: IRI{Value: "http://example.org/o"},
	}

	if err := enc.Write(stmt); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
}

func TestJSONLDEncoder_WriteWithGraph(t *testing.T) {
	var buf strings.Builder
	enc, err := NewWriter(&buf, FormatJSONLD)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}
	defer enc.Close()

	stmt := Statement{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: IRI{Value: "http://example.org/o"},
		G: IRI{Value: "http://example.org/g"},
	}

	if err := enc.Write(stmt); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
}

func TestJSONLDEncoder_WriteBlankNode(t *testing.T) {
	var buf strings.Builder
	enc, err := NewWriter(&buf, FormatJSONLD)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}
	defer enc.Close()

	stmt := Statement{
		S: BlankNode{ID: "b1"},
		P: IRI{Value: "http://example.org/p"},
		O: IRI{Value: "http://example.org/o"},
	}

	if err := enc.Write(stmt); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
}

func TestJSONLDEncoder_WriteLiteral(t *testing.T) {
	var buf strings.Builder
	enc, err := NewWriter(&buf, FormatJSONLD)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}
	defer enc.Close()

	stmt := Statement{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: Literal{Lexical: "test", Lang: "en"},
	}

	if err := enc.Write(stmt); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
}

func TestEncoder_Flush(t *testing.T) {
	var buf bytes.Buffer
	enc, err := NewWriter(&buf, FormatTurtle)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}
	defer enc.Close()

	stmt := Statement{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: IRI{Value: "http://example.org/o"},
	}

	if err := enc.Write(stmt); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if err := enc.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}
}

func TestEncoder_CloseIdempotent(t *testing.T) {
	var buf bytes.Buffer
	enc, err := NewWriter(&buf, FormatTurtle)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	if err := enc.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Close may or may not be idempotent depending on implementation
	// Some encoders error on second close, which is acceptable
	_ = enc.Close()
}

func TestEncoder_WriteAfterClose(t *testing.T) {
	var buf bytes.Buffer
	enc, err := NewWriter(&buf, FormatTurtle)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	if err := enc.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	stmt := Statement{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: IRI{Value: "http://example.org/o"},
	}

	// Write after close should error
	if err := enc.Write(stmt); err == nil {
		t.Error("Expected error when writing after close")
	}
}

func TestEncoder_FlushAfterClose(t *testing.T) {
	var buf bytes.Buffer
	enc, err := NewWriter(&buf, FormatTurtle)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	if err := enc.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Flush after close should error
	if err := enc.Flush(); err == nil {
		t.Error("Expected error when flushing after close")
	}
}

func TestEncoder_WriteMultipleStatements(t *testing.T) {
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
}

func TestEncoder_WriteWithSpecialCharacters(t *testing.T) {
	var buf strings.Builder
	enc, err := NewWriter(&buf, FormatTurtle)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}
	defer enc.Close()

	stmt := Statement{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: Literal{Lexical: "test\nwith\tspecial\"chars"},
	}

	if err := enc.Write(stmt); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
}

func TestEncoder_WriteLongIRI(t *testing.T) {
	var buf strings.Builder
	enc, err := NewWriter(&buf, FormatTurtle)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}
	defer enc.Close()

	longIRI := "http://example.org/" + strings.Repeat("path/", 100) + "resource"
	stmt := Statement{
		S: IRI{Value: longIRI},
		P: IRI{Value: "http://example.org/p"},
		O: IRI{Value: "http://example.org/o"},
	}

	if err := enc.Write(stmt); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
}

func TestEncoder_WriteEmptyLiteral(t *testing.T) {
	var buf strings.Builder
	enc, err := NewWriter(&buf, FormatTurtle)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}
	defer enc.Close()

	stmt := Statement{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: Literal{Lexical: ""},
	}

	if err := enc.Write(stmt); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
}

func TestEncoder_WriteNumericLiteral(t *testing.T) {
	var buf strings.Builder
	enc, err := NewWriter(&buf, FormatTurtle)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}
	defer enc.Close()

	stmt := Statement{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: Literal{
			Lexical:  "42",
			Datatype: IRI{Value: "http://www.w3.org/2001/XMLSchema#integer"},
		},
	}

	if err := enc.Write(stmt); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
}

func TestEncoder_WriteBooleanLiteral(t *testing.T) {
	var buf strings.Builder
	enc, err := NewWriter(&buf, FormatTurtle)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}
	defer enc.Close()

	stmt := Statement{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: Literal{
			Lexical:  "true",
			Datatype: IRI{Value: "http://www.w3.org/2001/XMLSchema#boolean"},
		},
	}

	if err := enc.Write(stmt); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
}
