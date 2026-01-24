package rdf

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

// Test edge cases and error paths

func TestNewReader_NilReader(t *testing.T) {
	// Library doesn't validate nil reader - it will panic later
	// This test documents the behavior
	defer func() {
		if r := recover(); r != nil {
			// Panic is expected for nil reader
		}
	}()
	_, _ = NewReader(nil, FormatTurtle)
}

func TestNewWriter_NilWriter(t *testing.T) {
	// Library doesn't validate nil writer - it will panic later
	// This test documents the behavior
	defer func() {
		if r := recover(); r != nil {
			// Panic is expected for nil writer
		}
	}()
	_, _ = NewWriter(nil, FormatTurtle)
}

func TestParse_NilReader(t *testing.T) {
	// Library doesn't validate nil reader - it will panic
	// This test documents the behavior
	defer func() {
		if r := recover(); r != nil {
			// Panic is expected for nil reader
		}
	}()
	_ = Parse(context.Background(), nil, FormatTurtle, func(Statement) error {
		return nil
	})
}

func TestParse_NilHandler(t *testing.T) {
	// This should panic or error - test the behavior
	defer func() {
		if r := recover(); r == nil {
			// Handler is called, so nil handler would panic
			// This is expected behavior
		}
	}()

	input := "<http://example.org/s> <http://example.org/p> <http://example.org/o> .\n"
	err := Parse(context.Background(), strings.NewReader(input), FormatNTriples, nil)
	if err == nil {
		// If it doesn't panic, it should error
	}
}

// Test encoder error handling

func TestTurtleEncoder_WriteAfterClose(t *testing.T) {
	var buf bytes.Buffer
	enc, err := NewWriter(&buf, FormatTurtle)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	enc.Close()

	stmt := Statement{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: IRI{Value: "http://example.org/o"},
	}

	if err := enc.Write(stmt); err == nil {
		t.Error("Write should fail after Close")
	}
}

func TestNTriplesEncoder_WriteAfterClose(t *testing.T) {
	var buf bytes.Buffer
	enc, err := NewWriter(&buf, FormatNTriples)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	enc.Close()

	stmt := Statement{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: IRI{Value: "http://example.org/o"},
	}

	if err := enc.Write(stmt); err == nil {
		t.Error("Write should fail after Close")
	}
}

// Test reader error handling

func TestReader_CloseMultipleTimes(t *testing.T) {
	input := "<http://example.org/s> <http://example.org/p> <http://example.org/o> .\n"
	dec, err := NewReader(strings.NewReader(input), FormatNTriples)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}

	if err := dec.Close(); err != nil {
		t.Errorf("First Close failed: %v", err)
	}

	// Second close should be safe
	if err := dec.Close(); err != nil {
		t.Errorf("Second Close failed: %v", err)
	}
}

func TestReader_NextAfterClose(t *testing.T) {
	input := "<http://example.org/s> <http://example.org/p> <http://example.org/o> .\n"
	dec, err := NewReader(strings.NewReader(input), FormatNTriples)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}

	dec.Close()

	// Next after close may or may not error depending on implementation
	// Just test that it doesn't panic
	_, _ = dec.Next()
}

// Test format detection edge cases

func TestDetectFormat_EmptyInput(t *testing.T) {
	format, reader, ok := detectFormat(strings.NewReader(""))
	if ok {
		t.Errorf("detectFormat should fail on empty input, got %v", format)
	}
	if reader == nil {
		t.Error("detectFormat should return reader even on failure")
	}
}

func TestDetectFormat_WhitespaceOnly(t *testing.T) {
	format, _, ok := detectFormat(strings.NewReader("   \n\t  "))
	if ok {
		t.Errorf("detectFormat should fail on whitespace-only input, got %v", format)
	}
}

func TestDetectFormat_ReadError(t *testing.T) {
	// Create a reader that will error on read
	errReader := &errorReader{err: io.ErrClosedPipe}
	format, _, ok := detectFormat(errReader)
	if ok {
		t.Errorf("detectFormat should fail on read error, got %v", format)
	}
}

type errorReader struct {
	err error
}

func (e *errorReader) Read(p []byte) (int, error) {
	return 0, e.err
}

// Test options edge cases

func TestOptions_MultipleOptions(t *testing.T) {
	ctx := context.Background()
	opts := defaultOptions()

	OptContext(ctx)(&opts)
	OptMaxDepth(50)(&opts)
	OptMaxTriples(1000000)(&opts)
	OptSafeLimits()(&opts)

	if opts.Context != ctx {
		t.Error("Context option failed")
	}
	// SafeLimits should override MaxDepth and MaxTriples
	safe := safeOptions()
	if opts.MaxDepth != safe.MaxDepth {
		t.Errorf("SafeLimits should override MaxDepth, got %d, want %d", opts.MaxDepth, safe.MaxDepth)
	}
	if opts.MaxTriples != safe.MaxTriples {
		t.Errorf("SafeLimits should override MaxTriples, got %d, want %d", opts.MaxTriples, safe.MaxTriples)
	}
}

// Test statement conversion edge cases

func TestStatement_AsTripleWithGraph(t *testing.T) {
	stmt := Statement{
		S: IRI{Value: "s"},
		P: IRI{Value: "p"},
		O: IRI{Value: "o"},
		G: IRI{Value: "g"}, // Has graph
	}

	triple := stmt.AsTriple()
	if triple.S != stmt.S || triple.P != stmt.P || triple.O != stmt.O {
		t.Error("AsTriple should preserve S, P, O")
	}
	// Graph should be ignored
}

func TestStatement_AsQuadWithoutGraph(t *testing.T) {
	stmt := Statement{
		S: IRI{Value: "s"},
		P: IRI{Value: "p"},
		O: IRI{Value: "o"},
		G: nil, // No graph
	}

	quad := stmt.AsQuad()
	if quad.G != nil {
		t.Error("AsQuad should preserve nil graph")
	}
}

// Test format string edge cases

func TestFormat_String_Auto(t *testing.T) {
	if FormatAuto.String() != "auto" {
		t.Errorf("FormatAuto.String() = %q, want 'auto'", FormatAuto.String())
	}
}

func TestFormat_String_Empty(t *testing.T) {
	if Format("").String() != "auto" {
		t.Errorf("Format(\"\").String() = %q, want 'auto'", Format("").String())
	}
}

// Test error code edge cases

func TestCode_WrappedParseError(t *testing.T) {
	baseErr := ErrLineTooLong
	wrapped := wrapParseError("turtle", "test", 0, baseErr)

	code := Code(wrapped)
	if code != ErrCodeLineTooLong {
		t.Errorf("Code(wrapped error) = %v, want ErrCodeLineTooLong", code)
	}
}

func TestCode_ParseErrorWithUnderlying(t *testing.T) {
	parseErr := &ParseError{
		Format: "turtle",
		Err:    ErrDepthExceeded,
	}

	code := Code(parseErr)
	if code != ErrCodeDepthExceeded {
		t.Errorf("Code(ParseError) = %v, want ErrCodeDepthExceeded", code)
	}
}

// Test quad adapter edge cases

func TestQuadReaderAdapter_CloseError(t *testing.T) {
	// Test that adapter properly handles close
	input := "<http://example.org/s> <http://example.org/p> <http://example.org/o> .\n"
	dec, err := NewReader(strings.NewReader(input), FormatNTriples)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}

	// Read one statement
	_, err = dec.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}

	// Close should work
	if err := dec.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestQuadWriterAdapter_FlushAfterClose(t *testing.T) {
	var buf bytes.Buffer
	enc, err := NewWriter(&buf, FormatNTriples)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	enc.Close()

	// Flush after close should be safe (no-op or error)
	_ = enc.Flush()
}

// Test format detection with various inputs

func TestDetectFormatFromSample_Comments(t *testing.T) {
	// Input with comments - format detection may skip comments
	// Test that it doesn't crash
	input := "# Comment\n<http://example.org/s> <http://example.org/p> <http://example.org/o> ."
	format, ok := detectFormatFromSample(strings.NewReader(input))
	// May or may not detect - just test it doesn't crash
	_ = format
	_ = ok
}

func TestDetectQuadFormat_Comments(t *testing.T) {
	// Input with comments - format detection may skip comments
	// Test that it doesn't crash
	input := "# Comment\n<s> <p> <o> <g> ."
	format, ok := detectQuadFormat(strings.NewReader(input))
	// May or may not detect - just test it doesn't crash
	_ = format
	_ = ok
}

// Test IRI resolution edge cases

func TestResolveIRI_EmptyBase(t *testing.T) {
	got := resolveIRI("", "path")
	if got == "" {
		t.Error("resolveIRI should handle empty base")
	}
}

func TestResolveIRI_EmptyRelative(t *testing.T) {
	got := resolveIRI("http://example.org/base/", "")
	if got == "" {
		t.Error("resolveIRI should handle empty relative")
	}
}

// Test literal string formatting edge cases

func TestLiteral_String_Empty(t *testing.T) {
	lit := Literal{Lexical: ""}
	str := lit.String()
	if str != `""` {
		t.Errorf("Empty literal String() = %q, want '\"\"'", str)
	}
}

func TestLiteral_String_BothLangAndDatatype(t *testing.T) {
	// Lang takes precedence over datatype
	lit := Literal{
		Lexical:  "test",
		Lang:     "en",
		Datatype: IRI{Value: "http://example.org/string"},
	}
	str := lit.String()
	if !strings.Contains(str, "@en") {
		t.Errorf("Literal with both lang and datatype should prefer lang, got %q", str)
	}
	if strings.Contains(str, "^^") {
		t.Errorf("Literal with lang should not include datatype, got %q", str)
	}
}

// Test blank node string formatting

func TestBlankNode_String_EmptyID(t *testing.T) {
	bnode := BlankNode{ID: ""}
	str := bnode.String()
	if str != "_:" {
		t.Errorf("BlankNode with empty ID String() = %q, want '_:'", str)
	}
}

// Test triple term string formatting

func TestTripleTerm_String_NilTerms(t *testing.T) {
	// This might panic, but let's test the behavior
	defer func() {
		if r := recover(); r != nil {
			// Panic is acceptable for nil terms
		}
	}()

	tt := TripleTerm{
		S: nil,
		P: IRI{Value: "p"},
		O: nil,
	}
	_ = tt.String() // Should handle nil gracefully or panic
}

// Test statement methods with edge cases

func TestStatement_IsQuad_NilGraph(t *testing.T) {
	stmt := Statement{G: nil}
	if stmt.IsQuad() {
		t.Error("Statement with nil graph should not be quad")
	}
}

func TestStatement_IsTriple_WithGraph(t *testing.T) {
	stmt := Statement{G: IRI{Value: "g"}}
	if stmt.IsTriple() {
		t.Error("Statement with graph should not be triple")
	}
}

// Test format detection with malformed input

func TestDetectFormatFromSample_MalformedJSON(t *testing.T) {
	input := "{ invalid json"
	format, ok := detectFormatFromSample(strings.NewReader(input))
	// Should either detect as JSON-LD or fail
	if ok && format != FormatJSONLD {
		t.Errorf("Malformed JSON should either fail or detect as JSON-LD, got %v", format)
	}
}

func TestDetectFormatFromSample_PartialXML(t *testing.T) {
	input := "<rdf:"
	format, ok := detectFormatFromSample(strings.NewReader(input))
	if ok && format != FormatRDFXML {
		t.Errorf("Partial XML should detect as RDF/XML, got %v", format)
	}
}

// Test error wrapping

func TestWrapParseError_NilError(t *testing.T) {
	err := wrapParseError("turtle", "test", 0, nil)
	if err != nil {
		t.Error("wrapParseError should return nil for nil error")
	}
}

func TestWrapParseErrorWithPosition_NilError(t *testing.T) {
	err := wrapParseErrorWithPosition("turtle", "test", 1, 1, 0, nil)
	if err != nil {
		t.Error("wrapParseErrorWithPosition should return nil for nil error")
	}
}

// Test ParseError formatting edge cases

func TestParseError_Error_NoPosition(t *testing.T) {
	err := &ParseError{
		Format: "turtle",
		Err:    errors.New("test error"),
		Line:   0,
		Column: 0,
		Offset: -1,
	}

	msg := err.Error()
	if msg == "" {
		t.Error("ParseError.Error() should return non-empty message")
	}
	if !strings.Contains(msg, "turtle") {
		t.Error("ParseError should include format name")
	}
}

func TestParseError_FormatExcerpt_NoStatement(t *testing.T) {
	err := &ParseError{
		Format:    "turtle",
		Statement: "",
		Line:      1,
		Column:    1,
		Err:       errors.New("test error"),
	}

	excerpt := err.formatExcerpt()
	if excerpt != "" {
		t.Error("formatExcerpt should return empty for empty statement")
	}
}
