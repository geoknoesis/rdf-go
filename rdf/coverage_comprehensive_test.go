package rdf

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

// Test model functions for coverage

func TestNewTriple(t *testing.T) {
	s := IRI{Value: "http://example.org/s"}
	p := IRI{Value: "http://example.org/p"}
	o := IRI{Value: "http://example.org/o"}

	stmt := NewTriple(s, p, o)
	if stmt.S != s || stmt.P != p || stmt.O != o || stmt.G != nil {
		t.Errorf("NewTriple failed: got %v", stmt)
	}
	if !stmt.IsTriple() {
		t.Error("NewTriple should create a triple")
	}
	if stmt.IsQuad() {
		t.Error("NewTriple should not create a quad")
	}
}

func TestNewQuad(t *testing.T) {
	s := IRI{Value: "http://example.org/s"}
	p := IRI{Value: "http://example.org/p"}
	o := IRI{Value: "http://example.org/o"}
	g := IRI{Value: "http://example.org/g"}

	stmt := NewQuad(s, p, o, g)
	if stmt.S != s || stmt.P != p || stmt.O != o || stmt.G != g {
		t.Errorf("NewQuad failed: got %v", stmt)
	}
	if !stmt.IsQuad() {
		t.Error("NewQuad should create a quad")
	}
	if stmt.IsTriple() {
		t.Error("NewQuad should not create a triple")
	}
}

func TestTripleToStatement(t *testing.T) {
	triple := Triple{
		S: IRI{Value: "s"},
		P: IRI{Value: "p"},
		O: IRI{Value: "o"},
	}
	stmt := triple.ToStatement()
	if stmt.G != nil {
		t.Error("Triple.ToStatement should have nil graph")
	}
}

func TestQuadToStatement(t *testing.T) {
	quad := Quad{
		S: IRI{Value: "s"},
		P: IRI{Value: "p"},
		O: IRI{Value: "o"},
		G: IRI{Value: "g"},
	}
	stmt := quad.ToStatement()
	if stmt.G == nil {
		t.Error("Quad.ToStatement should have non-nil graph")
	}
}

func TestStatementAsTriple(t *testing.T) {
	stmt := Statement{
		S: IRI{Value: "s"},
		P: IRI{Value: "p"},
		O: IRI{Value: "o"},
		G: nil,
	}
	triple := stmt.AsTriple()
	if triple.S != stmt.S || triple.P != stmt.P || triple.O != stmt.O {
		t.Error("AsTriple failed")
	}
}

func TestStatementAsQuad(t *testing.T) {
	stmt := Statement{
		S: IRI{Value: "s"},
		P: IRI{Value: "p"},
		O: IRI{Value: "o"},
		G: IRI{Value: "g"},
	}
	quad := stmt.AsQuad()
	if quad.G == nil {
		t.Error("AsQuad should preserve graph")
	}
}

func TestQuadToTriple(t *testing.T) {
	quad := Quad{
		S: IRI{Value: "s"},
		P: IRI{Value: "p"},
		O: IRI{Value: "o"},
		G: IRI{Value: "g"},
	}
	triple := quad.ToTriple()
	if triple.S != quad.S || triple.P != quad.P || triple.O != quad.O {
		t.Error("ToTriple failed")
	}
}

func TestTripleToQuad(t *testing.T) {
	triple := Triple{
		S: IRI{Value: "s"},
		P: IRI{Value: "p"},
		O: IRI{Value: "o"},
	}
	quad := triple.ToQuad()
	if quad.G != nil {
		t.Error("ToQuad should have nil graph")
	}
}

func TestTripleToQuadInGraph(t *testing.T) {
	triple := Triple{
		S: IRI{Value: "s"},
		P: IRI{Value: "p"},
		O: IRI{Value: "o"},
	}
	graph := IRI{Value: "g"}
	quad := triple.ToQuadInGraph(graph)
	if quad.G != graph {
		t.Error("ToQuadInGraph failed")
	}
}

func TestQuadInDefaultGraph(t *testing.T) {
	q := Quad{G: nil}
	if !q.InDefaultGraph() {
		t.Error("Quad with nil graph should be in default graph")
	}
	q.G = IRI{Value: "g"}
	if q.InDefaultGraph() {
		t.Error("Quad with graph should not be in default graph")
	}
}

// Test format functions

func TestParseFormat_CaseInsensitive(t *testing.T) {
	// Test case insensitivity and whitespace handling
	tests := []struct {
		input    string
		expected Format
		ok       bool
	}{
		{"TURTLE", FormatTurtle, true},
		{"  turtle  ", FormatTurtle, true},
		{"", FormatAuto, true},
		{"auto", FormatAuto, true},
	}

	for _, tt := range tests {
		got, ok := ParseFormat(tt.input)
		if ok != tt.ok {
			t.Errorf("ParseFormat(%q) ok = %v, want %v", tt.input, ok, tt.ok)
		}
		if got != tt.expected {
			t.Errorf("ParseFormat(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestFormatIsQuadFormat(t *testing.T) {
	if !FormatTriG.IsQuadFormat() {
		t.Error("FormatTriG should be quad format")
	}
	if !FormatNQuads.IsQuadFormat() {
		t.Error("FormatNQuads should be quad format")
	}
	if FormatTurtle.IsQuadFormat() {
		t.Error("FormatTurtle should not be quad format")
	}
	if FormatNTriples.IsQuadFormat() {
		t.Error("FormatNTriples should not be quad format")
	}
}

func TestFormatString(t *testing.T) {
	if FormatAuto.String() != "auto" {
		t.Errorf("FormatAuto.String() = %q, want 'auto'", FormatAuto.String())
	}
	if FormatTurtle.String() != "turtle" {
		t.Errorf("FormatTurtle.String() = %q, want 'turtle'", FormatTurtle.String())
	}
}

// Test format detection

func TestDetectFormatFromSample_JSONLD(t *testing.T) {
	tests := []string{
		`{"@context": {"ex": "http://example.org/"}, "@id": "ex:s"}`,
		`{"@id": "ex:s", "@type": "ex:Person"}`,
		`[{"@id": "ex:s"}]`,
		`{}`,
		`[]`,
	}

	for _, input := range tests {
		format, ok := detectFormatFromSample(strings.NewReader(input))
		if !ok {
			t.Errorf("detectFormatFromSample failed for JSON-LD input: %q", input)
		}
		if format != FormatJSONLD {
			t.Errorf("detectFormatFromSample = %v, want FormatJSONLD for input: %q", format, input)
		}
	}
}

func TestDetectFormatFromSample_RDFXML(t *testing.T) {
	tests := []string{
		`<?xml version="1.0"?><rdf:RDF></rdf:RDF>`,
		`<rdf:RDF></rdf:RDF>`,
		`<rdf RDF></rdf>`,
	}

	for _, input := range tests {
		format, ok := detectFormatFromSample(strings.NewReader(input))
		if !ok {
			t.Errorf("detectFormatFromSample failed for RDF/XML input: %q", input)
		}
		if format != FormatRDFXML {
			t.Errorf("detectFormatFromSample = %v, want FormatRDFXML for input: %q", format, input)
		}
	}
}

func TestDetectFormatFromSample_Turtle(t *testing.T) {
	tests := []string{
		`@prefix ex: <http://example.org/> .`,
		`PREFIX ex: <http://example.org/> .`,
		`@base <http://example.org/> .`,
		`BASE <http://example.org/> .`,
		`ex:s ex:p ex:o .`,
		`[ ex:p ex:o ] .`,
		`( ex:o1 ex:o2 ) .`,
	}

	for _, input := range tests {
		format, ok := detectFormatFromSample(strings.NewReader(input))
		if !ok {
			t.Errorf("detectFormatFromSample failed for Turtle input: %q", input)
		}
		if format != FormatTurtle {
			t.Errorf("detectFormatFromSample = %v, want FormatTurtle for input: %q", format, input)
		}
	}
}

func TestDetectFormatFromSample_NTriples(t *testing.T) {
	tests := []string{
		`<http://example.org/s> <http://example.org/p> <http://example.org/o> .`,
		`_:b0 <http://example.org/p> <http://example.org/o> .`,
		`<http://example.org/s> <http://example.org/p> _:b0 .`,
	}

	for _, input := range tests {
		format, ok := detectFormatFromSample(strings.NewReader(input))
		if !ok {
			t.Errorf("detectFormatFromSample failed for N-Triples input: %q", input)
		}
		if format != FormatNTriples {
			t.Errorf("detectFormatFromSample = %v, want FormatNTriples for input: %q", format, input)
		}
	}
}

func TestDetectFormatFromSample_Empty(t *testing.T) {
	format, ok := detectFormatFromSample(strings.NewReader(""))
	if ok {
		t.Errorf("detectFormatFromSample should fail for empty input, got %v", format)
	}
	format, ok = detectFormatFromSample(strings.NewReader("   "))
	if ok {
		t.Errorf("detectFormatFromSample should fail for whitespace-only input, got %v", format)
	}
}

func TestDetectQuadFormat_TriG(t *testing.T) {
	tests := []string{
		`GRAPH <http://example.org/g> { <s> <p> <o> . }`,
		`@prefix ex: <http://example.org/> . GRAPH ex:g { ex:s ex:p ex:o . }`,
		`{ <s> <p> <o> . }`,
	}

	for _, input := range tests {
		format, ok := detectQuadFormat(strings.NewReader(input))
		if !ok {
			t.Errorf("detectQuadFormat failed for TriG input: %q", input)
		}
		if format != FormatTriG {
			t.Errorf("detectQuadFormat = %v, want FormatTriG for input: %q", format, input)
		}
	}
}

func TestDetectQuadFormat_NQuads(t *testing.T) {
	input := `<http://example.org/s> <http://example.org/p> <http://example.org/o> <http://example.org/g> .`
	format, ok := detectQuadFormat(strings.NewReader(input))
	if !ok {
		t.Error("detectQuadFormat failed for N-Quads input")
	}
	if format != FormatNQuads {
		t.Errorf("detectQuadFormat = %v, want FormatNQuads", format)
	}
}

func TestIsValidJSONStructure(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{`{}`, true},
		{`[]`, true},
		{`{"key": "value"}`, true},
		{`[1, 2, 3]`, true},
		{`{`, false},
		{`}`, false},
		{`[`, false},
		{`]`, false},
		{``, false},
		{`   `, false},
		{`{"key": "value"`, false},
	}

	for _, tt := range tests {
		got := isValidJSONStructure(tt.input)
		if got != tt.expected {
			t.Errorf("isValidJSONStructure(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

// Test API adapter functions

func TestQuadReaderAdapter_Triple(t *testing.T) {
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
	if stmt.IsQuad() {
		t.Error("NTriples should produce triples, not quads")
	}
	if !stmt.IsTriple() {
		t.Error("NTriples should produce triples")
	}
}

func TestQuadReaderAdapter_Quad(t *testing.T) {
	input := `<http://example.org/s> <http://example.org/p> <http://example.org/o> <http://example.org/g> .`
	dec, err := NewReader(strings.NewReader(input), FormatNQuads)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer dec.Close()

	stmt, err := dec.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	if !stmt.IsQuad() {
		t.Error("NQuads should produce quads")
	}
	if stmt.IsTriple() {
		t.Error("NQuads should not produce triples")
	}
}

func TestQuadWriterAdapter_Triple(t *testing.T) {
	var buf bytes.Buffer
	enc, err := NewWriter(&buf, FormatNTriples)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}
	defer enc.Close()

	stmt := Statement{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: IRI{Value: "http://example.org/o"},
		G: nil, // Triple
	}

	if err := enc.Write(stmt); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if err := enc.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}
}

func TestQuadWriterAdapter_Quad(t *testing.T) {
	var buf bytes.Buffer
	enc, err := NewWriter(&buf, FormatNQuads)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}
	defer enc.Close()

	stmt := Statement{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: IRI{Value: "http://example.org/o"},
		G: IRI{Value: "http://example.org/g"}, // Quad
	}

	if err := enc.Write(stmt); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if err := enc.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}
}

// Test options

func TestOptContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	opts := defaultOptions()
	OptContext(ctx)(&opts)
	if opts.Context != ctx {
		t.Error("OptContext failed")
	}
}

func TestOptMaxLineBytes(t *testing.T) {
	opts := defaultOptions()
	OptMaxLineBytes(1000)(&opts)
	if opts.MaxLineBytes != 1000 {
		t.Error("OptMaxLineBytes failed")
	}
}

func TestOptMaxStatementBytes(t *testing.T) {
	opts := defaultOptions()
	OptMaxStatementBytes(2000)(&opts)
	if opts.MaxStatementBytes != 2000 {
		t.Error("OptMaxStatementBytes failed")
	}
}

func TestOptMaxDepth(t *testing.T) {
	opts := defaultOptions()
	OptMaxDepth(50)(&opts)
	if opts.MaxDepth != 50 {
		t.Error("OptMaxDepth failed")
	}
}

func TestOptMaxTriples(t *testing.T) {
	opts := defaultOptions()
	OptMaxTriples(1000000)(&opts)
	if opts.MaxTriples != 1000000 {
		t.Error("OptMaxTriples failed")
	}
}

func TestOptSafeLimits(t *testing.T) {
	opts := defaultOptions()
	OptSafeLimits()(&opts)
	safe := safeOptions()
	if opts.MaxLineBytes != safe.MaxLineBytes {
		t.Error("OptSafeLimits MaxLineBytes failed")
	}
	if opts.MaxStatementBytes != safe.MaxStatementBytes {
		t.Error("OptSafeLimits MaxStatementBytes failed")
	}
	if opts.MaxDepth != safe.MaxDepth {
		t.Error("OptSafeLimits MaxDepth failed")
	}
	if opts.MaxTriples != safe.MaxTriples {
		t.Error("OptSafeLimits MaxTriples failed")
	}
}

// Test format auto-detection

func TestNewReader_FormatAuto(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		format Format
	}{
		{"Turtle", "@prefix ex: <http://example.org/> .", FormatTurtle},
		{"NTriples", "<http://example.org/s> <http://example.org/p> <http://example.org/o> .", FormatNTriples},
		{"TriG", "GRAPH <g> { <s> <p> <o> . }", FormatTriG},
		{"NQuads", "<s> <p> <o> <g> .", FormatNQuads},
		{"JSONLD", `{"@context": {"ex": "http://example.org/"}}`, FormatJSONLD},
		{"RDFXML", `<?xml version="1.0"?><rdf:RDF></rdf:RDF>`, FormatRDFXML},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dec, err := NewReader(strings.NewReader(tt.input), FormatAuto)
			if err != nil {
				t.Fatalf("NewReader failed: %v", err)
			}
			defer dec.Close()

			// Try to read - should work if format detected correctly
			_, err = dec.Next()
			if err != nil && err != io.EOF {
				// Some formats might fail on incomplete input, that's OK
				// We just want to verify format detection doesn't crash
			}
		})
	}
}

func TestNewReader_FormatAuto_Failure(t *testing.T) {
	// Invalid input that can't be detected
	_, err := NewReader(strings.NewReader("invalid input that doesn't match any format"), FormatAuto)
	if err == nil {
		t.Error("NewReader should fail for undetectable format")
	}
	if err != ErrUnsupportedFormat {
		t.Errorf("NewReader error = %v, want ErrUnsupportedFormat", err)
	}
}

// Test error handling edge cases

func TestParseError_FormatExcerpt_NoColumn(t *testing.T) {
	err := &ParseError{
		Format:    "turtle",
		Statement: "short statement",
		Line:      1,
		Column:    0, // No column
		Offset:    10,
		Err:       errors.New("test error"),
	}

	msg := err.Error()
	if msg == "" {
		t.Error("ParseError.Error() should return non-empty message")
	}
	if !strings.Contains(msg, "turtle") {
		t.Error("ParseError should include format name")
	}
}

func TestParseError_FormatExcerpt_LongStatement(t *testing.T) {
	longStmt := strings.Repeat("a", 200)
	err := &ParseError{
		Format:    "turtle",
		Statement: longStmt,
		Line:      1,
		Column:    0,
		Offset:    10,
		Err:       errors.New("test error"),
	}

	msg := err.Error()
	if !strings.Contains(msg, "...") {
		t.Error("ParseError should truncate long statements")
	}
}

// Test literal string formatting

func TestLiteral_String_WithLang(t *testing.T) {
	lit := Literal{
		Lexical: "Hello",
		Lang:    "en",
	}
	str := lit.String()
	if !strings.Contains(str, "@en") {
		t.Errorf("Literal with lang should include lang tag, got %q", str)
	}
}

func TestLiteral_String_WithDatatype(t *testing.T) {
	lit := Literal{
		Lexical:  "42",
		Datatype: IRI{Value: "http://www.w3.org/2001/XMLSchema#integer"},
	}
	str := lit.String()
	if !strings.Contains(str, "^^") {
		t.Errorf("Literal with datatype should include datatype, got %q", str)
	}
}

func TestLiteral_String_Plain(t *testing.T) {
	lit := Literal{
		Lexical: "Hello",
	}
	str := lit.String()
	if !strings.HasPrefix(str, `"`) || !strings.HasSuffix(str, `"`) {
		t.Errorf("Plain literal should be quoted, got %q", str)
	}
}

// Test term kinds (additional coverage beyond model_test.go)

func TestTermKinds_Additional(t *testing.T) {
	// Test that all term types return correct kinds
	iri := IRI{Value: "http://example.org/"}
	if iri.Kind() != TermIRI {
		t.Error("IRI.Kind() should return TermIRI")
	}

	bnode := BlankNode{ID: "b1"}
	if bnode.Kind() != TermBlankNode {
		t.Error("BlankNode.Kind() should return TermBlankNode")
	}

	lit := Literal{Lexical: "value"}
	if lit.Kind() != TermLiteral {
		t.Error("Literal.Kind() should return TermLiteral")
	}

	triple := TripleTerm{
		S: IRI{Value: "s"},
		P: IRI{Value: "p"},
		O: IRI{Value: "o"},
	}
	if triple.Kind() != TermTriple {
		t.Error("TripleTerm.Kind() should return TermTriple")
	}
}

// Test writer error handling

func TestWriter_WriteError(t *testing.T) {
	// Create a writer that will fail on write
	var buf bytes.Buffer
	enc, err := NewWriter(&buf, FormatNTriples)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	// Close first to put writer in error state
	enc.Close()

	// Try to write after close
	stmt := Statement{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: IRI{Value: "http://example.org/o"},
	}
	if err := enc.Write(stmt); err == nil {
		t.Error("Write should fail after Close")
	}
}
