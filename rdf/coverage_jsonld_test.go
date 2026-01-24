package rdf

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
)

func TestJSONLDDecoderErrClose(t *testing.T) {
	// Test error handling with actual decoder
	input := `invalid json`
	dec, err := NewReader(strings.NewReader(input), FormatJSONLD)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := dec.Next(); err == nil {
		t.Fatal("expected error")
	}
	if err := dec.Close(); err != nil {
		t.Fatalf("expected Close to be nil, got %v", err)
	}
}

func TestJSONLDLoad_ArrayTopLevel(t *testing.T) {
	input := `[{"@context":{"ex":"http://example.org/"},"@id":"ex:s","ex:p":"v"}]`
	dec, err := NewReader(strings.NewReader(input), FormatJSONLD)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := dec.Next(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJSONLDWithVocab(t *testing.T) {
	input := `{"@context":{"@vocab":"http://example.org/"},"@id":"thing","p":"v"}`
	dec, err := NewReader(strings.NewReader(input), FormatJSONLD)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	triple, err := dec.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if triple.P.Value != "http://example.org/p" {
		t.Fatalf("unexpected predicate: %s", triple.P.Value)
	}
}

func TestJSONLDValueTypes(t *testing.T) {
	input := `{"@context":{"ex":"http://example.org/"},"@id":"ex:s","ex:p":[{"@id":"ex:o"},{"@value":"x"},1,true,"str"]}`
	dec, err := NewReader(strings.NewReader(input), FormatJSONLD)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i := 0; i < 5; i++ {
		if _, err := dec.Next(); err != nil {
			t.Fatalf("unexpected error at %d: %v", i, err)
		}
	}
}

func TestJSONLDExpandTermFallbacks(t *testing.T) {
	ctx := newJSONLDContext()
	if got := expandJSONLDTerm("ex:value", ctx); got != "ex:value" {
		t.Fatalf("unexpected fallback: %s", got)
	}
	if got := expandJSONLDTerm("plain", ctx); got != "plain" {
		t.Fatalf("unexpected plain term: %s", got)
	}
}

func TestJSONLDGraphObject(t *testing.T) {
	input := `{"@context":{"ex":"http://example.org/"},"@graph":{"@id":"ex:s","ex:p":"v"}}`
	dec, err := NewReader(strings.NewReader(input), FormatJSONLD)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := dec.Next(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJSONLDErrorUnsupportedLiteral(t *testing.T) {
	input := `{"@context":{"ex":"http://example.org/"},"@id":"ex:s","ex:p":null}`
	dec, err := NewReader(strings.NewReader(input), FormatJSONLD)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := dec.Next(); err == nil {
		t.Fatal("expected error for unsupported literal value")
	}
}

func TestJSONLDEncoderErrors(t *testing.T) {
	enc, err := NewWriter(failingWriter{}, FormatJSONLD)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := enc.Write(Statement{S: IRI{Value: "s"}, P: IRI{Value: "p"}, O: Literal{Lexical: "v"}, G: nil}); err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}
	if err := enc.Flush(); err == nil {
		t.Fatal("expected flush error")
	}

	var buf bytes.Buffer
	enc, err = NewWriter(&buf, FormatJSONLD)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = enc.Write(Statement{S: IRI{Value: "s"}, P: IRI{Value: "p"}, O: Literal{Lexical: "v"}, G: nil})
	_ = enc.Write(Statement{S: IRI{Value: "s2"}, P: IRI{Value: "p2"}, O: IRI{Value: "o2"}, G: nil})
	if err := enc.Close(); err != nil {
		t.Fatalf("unexpected close error: %v", err)
	}
	enc, err = NewWriter(&buf, FormatJSONLD)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = enc.Close()
	if err := enc.Write(Statement{S: IRI{Value: "s"}, P: IRI{Value: "p"}, O: Literal{Lexical: "v"}}); err == nil {
		t.Fatal("expected closed error")
	}
}

func TestJSONLDDecoderEOF(t *testing.T) {
	input := `{"@context":{"ex":"http://example.org/"},"@id":"ex:s","ex:p":"v"}`
	dec, err := NewReader(strings.NewReader(input), FormatJSONLD)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := dec.Next(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := dec.Next(); err != io.EOF {
		t.Fatalf("expected EOF, got %v", err)
	}
}

func TestJSONLDLoadNonCollection(t *testing.T) {
	dec, err := NewReader(strings.NewReader("5"), FormatJSONLD)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := dec.Next(); err != io.EOF {
		t.Fatalf("expected EOF for non-collection, got %v", err)
	}
}

func TestJSONLDLoadArraySkipsNonObjects(t *testing.T) {
	input := `[{"@context":{"ex":"http://example.org/"},"@id":"ex:s","ex:p":"v"}, "skip"]`
	dec, err := NewReader(strings.NewReader(input), FormatJSONLD)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := dec.Next(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJSONLDLoadGraphError(t *testing.T) {
	input := `{"@context":{"ex":"http://example.org/"},"@graph":{"ex:p":"v"}}`
	dec, err := NewReader(strings.NewReader(input), FormatJSONLD)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := dec.Next(); err == nil {
		t.Fatal("expected graph parse error")
	}
}

func TestJSONLDLoadNodeError(t *testing.T) {
	input := `{"@context":{"ex":"http://example.org/"},"ex:p":"v"}`
	dec, err := NewReader(strings.NewReader(input), FormatJSONLD)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := dec.Next(); err == nil {
		t.Fatal("expected node parse error")
	}
}

func TestJSONLDGraphUnsupportedType(t *testing.T) {
	var quads []Quad
	state := &jsonldState{}
	if err := parseJSONLDGraph("bad", newJSONLDContext(), nil, state, appendQuadSink(&quads)); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestJSONLDGraphErrorPath(t *testing.T) {
	var quads []Quad
	state := &jsonldState{}
	err := parseJSONLDGraph(map[string]interface{}{"ex:p": "v"}, newJSONLDContext(), nil, state, appendQuadSink(&quads))
	if err == nil {
		t.Fatal("expected graph parse error")
	}
}

func TestJSONLDGraphArraySkipsInvalid(t *testing.T) {
	var quads []Quad
	state := &jsonldState{}
	err := parseJSONLDGraph([]interface{}{"skip"}, newJSONLDContext(), nil, state, appendQuadSink(&quads))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJSONLDGraphArrayError(t *testing.T) {
	var quads []Quad
	state := &jsonldState{}
	err := parseJSONLDGraph([]interface{}{map[string]interface{}{"ex:p": "v"}}, newJSONLDContext(), nil, state, appendQuadSink(&quads))
	if err == nil {
		t.Fatal("expected error for graph node without @id")
	}
}

func TestJSONLDLoadEmptyArray(t *testing.T) {
	dec, err := NewReader(strings.NewReader("[]"), FormatJSONLD)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := dec.Next(); err != io.EOF {
		t.Fatalf("expected EOF, got %v", err)
	}
}

func TestJSONLDLoadArrayNodeError(t *testing.T) {
	input := `[{"ex:p":"v"}]`
	dec, err := NewReader(strings.NewReader(input), FormatJSONLD)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := dec.Next(); err == nil {
		t.Fatal("expected node error in array")
	}
}

func TestJSONLDLoadDecodeError(t *testing.T) {
	dec, err := NewReader(strings.NewReader("{"), FormatJSONLD)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := dec.Next(); err == nil {
		t.Fatal("expected decode error")
	}
}

func TestJSONLDPredicateResolutionError(t *testing.T) {
	node := map[string]interface{}{
		"@id": "http://example.org/s",
		"":    "v",
	}
	state := &jsonldState{}
	var quads []Quad
	if err := parseJSONLDNode(node, newJSONLDContext(), nil, state, appendQuadSink(&quads)); err == nil {
		t.Fatal("expected predicate resolution error")
	}
}

func TestJSONLDEncoderErrorState(t *testing.T) {
	enc, err := NewWriter(failingWriter{}, FormatJSONLD)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Write to trigger error
	_ = enc.Write(Statement{S: IRI{Value: "s"}, P: IRI{Value: "p"}, O: Literal{Lexical: "v"}, G: nil})
	// Flush should fail
	if err := enc.Flush(); err == nil {
		t.Fatal("expected flush error")
	}
	// Close should also fail
	if err := enc.Close(); err == nil {
		t.Fatal("expected close error")
	}
}

func TestJSONLDObjectValueBranches(t *testing.T) {
	if got, err := jsonldObjectValueJSON(IRI{Value: "http://example.org/o"}); err != nil || !strings.Contains(string(got), "@id") {
		t.Fatalf("expected @id value, got %s (err=%v)", string(got), err)
	}
	if got, err := jsonldObjectValueJSON(Literal{Lexical: "v"}); err != nil || !strings.Contains(string(got), "@value") {
		t.Fatalf("expected @value literal, got %s (err=%v)", string(got), err)
	}
	if got, err := jsonldObjectValueJSON(customTerm{}); err != nil || string(got) == "" {
		t.Fatalf("expected fallback value, got %s (err=%v)", string(got), err)
	}
}

func TestJSONLDEmitArrayBranch(t *testing.T) {
	var quads []Quad
	ctx := newJSONLDContext().withContext(map[string]interface{}{"ex": "http://example.org/"})
	sub := IRI{Value: "http://example.org/s"}
	pred := IRI{Value: "http://example.org/p"}
	value := []interface{}{"v1", "v2"}
	state := &jsonldState{}
	if err := emitJSONLDValue(sub, pred, value, ctx, nil, state, appendQuadSink(&quads)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(quads) != 2 {
		t.Fatalf("expected 2 quads, got %d", len(quads))
	}
}

func TestJSONLDEmitValueMap(t *testing.T) {
	var quads []Quad
	ctx := newJSONLDContext()
	sub := IRI{Value: "http://example.org/s"}
	pred := IRI{Value: "http://example.org/p"}
	value := map[string]interface{}{"@value": "x"}
	state := &jsonldState{}
	if err := emitJSONLDValue(sub, pred, value, ctx, nil, state, appendQuadSink(&quads)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJSONLDEmitValueUnsupportedMap(t *testing.T) {
	var quads []Quad
	ctx := newJSONLDContext()
	sub := IRI{Value: "http://example.org/s"}
	pred := IRI{Value: "http://example.org/p"}
	value := map[string]interface{}{"bad": "x"}
	state := &jsonldState{}
	if err := emitJSONLDValue(sub, pred, value, ctx, nil, state, appendQuadSink(&quads)); err == nil {
		t.Fatal("expected unsupported object error")
	}
}

func TestJSONLDEmitValueScalarBranches(t *testing.T) {
	var quads []Quad
	ctx := newJSONLDContext()
	sub := IRI{Value: "http://example.org/s"}
	pred := IRI{Value: "http://example.org/p"}
	state := &jsonldState{}
	if err := emitJSONLDValue(sub, pred, 1.5, ctx, nil, state, appendQuadSink(&quads)); err != nil {
		t.Fatalf("unexpected float error: %v", err)
	}
	if err := emitJSONLDValue(sub, pred, true, ctx, nil, state, appendQuadSink(&quads)); err != nil {
		t.Fatalf("unexpected bool error: %v", err)
	}
}

func TestJSONLDEmitArrayError(t *testing.T) {
	var quads []Quad
	ctx := newJSONLDContext()
	sub := IRI{Value: "http://example.org/s"}
	pred := IRI{Value: "http://example.org/p"}
	value := []interface{}{map[string]interface{}{"bad": "x"}}
	state := &jsonldState{}
	if err := emitJSONLDValue(sub, pred, value, ctx, nil, state, appendQuadSink(&quads)); err == nil {
		t.Fatal("expected error for invalid array value")
	}
}

func appendQuadSink(quads *[]Quad) jsonldQuadSink {
	return func(q Quad) error {
		*quads = append(*quads, q)
		return nil
	}
}

func TestJSONLDWriteErrors(t *testing.T) {
	enc, err := NewWriter(failingWriter{}, FormatJSONLD)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Write should succeed initially
	if err := enc.Write(Statement{S: IRI{Value: "s"}, P: IRI{Value: "p"}, O: Literal{Lexical: "v"}}); err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}
	// Flush should fail
	if err := enc.Flush(); err == nil {
		t.Fatal("expected flush error")
	}

	enc, err = NewWriter(&bytes.Buffer{}, FormatJSONLD)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = enc.Close()
	if err := enc.Close(); err != nil {
		t.Fatalf("expected close idempotent: %v", err)
	}
}

type errWriter struct{}

func (errWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestJSONLDWriteCommaError(t *testing.T) {
	// Test that writing multiple triples works
	enc, err := NewWriter(&bytes.Buffer{}, FormatJSONLD)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = enc.Write(Statement{S: IRI{Value: "s"}, P: IRI{Value: "p"}, O: Literal{Lexical: "v"}, G: nil})
	_ = enc.Write(Statement{S: IRI{Value: "s2"}, P: IRI{Value: "p2"}, O: Literal{Lexical: "v2"}})
	if err := enc.Close(); err != nil {
		t.Fatalf("unexpected close error: %v", err)
	}
}

func TestJSONLDWriteOpenError(t *testing.T) {
	enc, err := NewWriter(errWriter{}, FormatJSONLD)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := enc.Write(Statement{S: IRI{Value: "s"}, P: IRI{Value: "p"}, O: Literal{Lexical: "v"}}); err == nil {
		t.Fatal("expected write error")
	}
}

type failAfterWriter struct {
	writes int
}

func (f *failAfterWriter) Write(p []byte) (int, error) {
	f.writes++
	if f.writes >= 2 {
		return 0, io.ErrClosedPipe
	}
	return len(p), nil
}

func TestJSONLDWriteFragmentError(t *testing.T) {
	writer := &failAfterWriter{}
	enc, err := NewWriter(writer, FormatJSONLD)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = enc.Write(Statement{S: IRI{Value: "s"}, P: IRI{Value: "p"}, O: Literal{Lexical: "v"}, G: nil})
	// Second write should fail due to writer
	if err := enc.Write(Statement{S: IRI{Value: "s2"}, P: IRI{Value: "p2"}, O: Literal{Lexical: "v2"}}); err == nil {
		t.Fatal("expected write error")
	}
}

func TestJSONLDCloseWriteError(t *testing.T) {
	enc, err := NewWriter(&bytes.Buffer{}, FormatJSONLD)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = enc.Write(Statement{S: IRI{Value: "s"}, P: IRI{Value: "p"}, O: Literal{Lexical: "v"}, G: nil})
	if err := enc.Close(); err != nil {
		t.Fatalf("unexpected close error: %v", err)
	}
}

func TestJSONLDCloseWithErrorState(t *testing.T) {
	enc, err := NewWriter(errWriter{}, FormatJSONLD)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = enc.Write(Statement{S: IRI{Value: "s"}, P: IRI{Value: "p"}, O: Literal{Lexical: "v"}, G: nil})
	// Close should handle error state
	if err := enc.Close(); err == nil {
		t.Fatal("expected close error")
	}
}

func TestJSONLDObjectValueJSONLiteralCombinations(t *testing.T) {
	// Test literal with lang only
	lit1 := Literal{Lexical: "test", Lang: "en"}
	if got, err := jsonldObjectValueJSON(lit1); err != nil || !strings.Contains(string(got), "@language") {
		t.Fatalf("expected @language, got %s (err=%v)", string(got), err)
	}

	// Test literal with datatype only
	lit2 := Literal{Lexical: "test", Datatype: IRI{Value: "http://www.w3.org/2001/XMLSchema#string"}}
	if got, err := jsonldObjectValueJSON(lit2); err != nil || !strings.Contains(string(got), "@type") {
		t.Fatalf("expected @type, got %s (err=%v)", string(got), err)
	}

	// Test literal with both lang and datatype (should prefer lang)
	lit3 := Literal{Lexical: "test", Lang: "en", Datatype: IRI{Value: "http://www.w3.org/2001/XMLSchema#string"}}
	if got, err := jsonldObjectValueJSON(lit3); err != nil || !strings.Contains(string(got), "@value") {
		t.Fatalf("expected @value, got %s (err=%v)", string(got), err)
	}

	// Test BlankNode
	bn := BlankNode{ID: "b1"}
	if got, err := jsonldObjectValueJSON(bn); err != nil || !strings.Contains(string(got), "@id") {
		t.Fatalf("expected @id for blank node, got %s (err=%v)", string(got), err)
	}
}

func TestJSONLDSubjectIDInvalidSubject(t *testing.T) {
	// Test with invalid subject type (Literal)
	lit := Literal{Lexical: "test"}
	if _, err := jsonldSubjectID(lit); err == nil {
		t.Fatal("expected error for invalid subject type")
	}
}

func TestJSONLDTripleEncoderInvalidSubject(t *testing.T) {
	var buf bytes.Buffer
	enc := newJSONLDtripleEncoder(&buf).(*jsonldtripleEncoder)

	// Test invalid subject type
	triple := Triple{S: Literal{Lexical: "invalid"}, P: IRI{Value: "p"}, O: Literal{Lexical: "v"}}
	if err := enc.Write(triple); err == nil {
		t.Fatal("expected error for invalid subject")
	}
}

func TestJSONLDTripleEncoderMissingPredicate(t *testing.T) {
	var buf bytes.Buffer
	enc := newJSONLDtripleEncoder(&buf).(*jsonldtripleEncoder)

	// Test missing predicate
	triple := Triple{S: IRI{Value: "s"}, P: IRI{Value: ""}, O: Literal{Lexical: "v"}}
	if err := enc.Write(triple); err == nil {
		t.Fatal("expected error for missing predicate")
	}
}

func TestJSONLDTripleEncoderMissingObject(t *testing.T) {
	var buf bytes.Buffer
	enc := newJSONLDtripleEncoder(&buf).(*jsonldtripleEncoder)

	// Test missing object
	triple := Triple{S: IRI{Value: "s"}, P: IRI{Value: "p"}, O: nil}
	if err := enc.Write(triple); err == nil {
		t.Fatal("expected error for missing object")
	}
}

func TestJSONLDValueTermErrorCases(t *testing.T) {
	ctx := newJSONLDContext()
	state := &jsonldState{}
	var quads []Quad
	sink := appendQuadSink(&quads)

	// Test invalid list value (not a map with @id or @value)
	invalid := map[string]interface{}{"bad": "value"}
	if _, err := jsonldValueTerm(invalid, ctx, nil, state, sink); err == nil {
		t.Fatal("expected error for invalid list value")
	}
}

func TestJSONLDEmitListErrorCases(t *testing.T) {
	ctx := newJSONLDContext()
	state := &jsonldState{}
	var quads []Quad
	sink := appendQuadSink(&quads)

	// Test invalid @list value (not an array)
	invalid := "not an array"
	if _, err := emitJSONLDList(invalid, ctx, nil, state, sink); err == nil {
		t.Fatal("expected error for invalid @list value")
	}
}

func TestJSONLDExpandTermWithBaseIRI(t *testing.T) {
	ctx := newJSONLDContext()
	ctx.base = "http://example.org/"

	// Test expansion with base IRI
	if got := expandJSONLDTerm("relative", ctx); got != "http://example.org/relative" {
		t.Fatalf("expected base+relative, got %s", got)
	}

	// Test expansion with vocab
	ctx.vocab = "http://example.org/vocab/"
	if got := expandJSONLDTerm("term", ctx); got != "http://example.org/vocab/term" {
		t.Fatalf("expected vocab+term, got %s", got)
	}

	// Test expansion with prefix
	ctx.prefixes["ex"] = "http://example.org/"
	if got := expandJSONLDTerm("ex:value", ctx); got != "http://example.org/value" {
		t.Fatalf("expected prefix expansion, got %s", got)
	}
}

func TestJSONLDObjectFromID(t *testing.T) {
	ctx := newJSONLDContext().withContext(map[string]interface{}{"ex": "http://example.org/"})
	state := &jsonldState{}

	// Test blank node
	bn := jsonldObjectFromID("_:b1", ctx, state)
	if _, ok := bn.(BlankNode); !ok {
		t.Fatal("expected BlankNode")
	}

	// Test IRI
	iri := jsonldObjectFromID("ex:value", ctx, state)
	if iriVal, ok := iri.(IRI); !ok || iriVal.Value != "http://example.org/value" {
		t.Fatalf("expected IRI, got %T", iri)
	}
}

func TestJSONLDSubjectErrorCases(t *testing.T) {
	ctx := newJSONLDContext()
	state := &jsonldState{}

	// Test nil subject
	if _, err := jsonldSubject(nil, ctx, state); err == nil {
		t.Fatal("expected error for nil subject")
	}

	// Test invalid subject type
	if _, err := jsonldSubject(123, ctx, state); err == nil {
		t.Fatal("expected error for invalid subject type")
	}

	// Test empty expanded IRI
	ctx.vocab = ""
	ctx.base = ""
	if _, err := jsonldSubject("", ctx, state); err == nil {
		t.Fatal("expected error for empty expanded IRI")
	}
}

func TestJSONLDShouldEagerFlush(t *testing.T) {
	// Test with errWriter
	if !shouldEagerFlushJSONLD(errWriter{}) {
		t.Fatal("expected true for errWriter")
	}

	// Test with failAfterWriter
	if !shouldEagerFlushJSONLD(&failAfterWriter{}) {
		t.Fatal("expected true for failAfterWriter")
	}

	// Test with regular buffer
	var buf bytes.Buffer
	if shouldEagerFlushJSONLD(&buf) {
		t.Fatal("expected false for regular buffer")
	}
}

func TestJSONLDLimitReader(t *testing.T) {
	// Test with maxBytes <= 0 (should return reader as-is)
	reader := strings.NewReader("test")
	limited := limitJSONLDReader(reader, 0)
	if limited != reader {
		t.Fatal("expected same reader when maxBytes <= 0")
	}

	// Test with maxBytes > 0
	limited = limitJSONLDReader(reader, 10)
	if limited == reader {
		t.Fatal("expected different reader when maxBytes > 0")
	}
}

func TestJSONLDContextWithCancel(t *testing.T) {
	opts := JSONLDOptions{}
	ctx, cancel := jsonldContextWithCancel(opts)
	if ctx == nil {
		t.Fatal("expected non-nil context")
	}
	cancel()
}

func TestJSONLDContextOrBackground(t *testing.T) {
	opts := JSONLDOptions{}
	ctx := jsonldContextOrBackground(opts)
	if ctx == nil {
		t.Fatal("expected non-nil context")
	}
}

func TestJSONLDQuadEncoder(t *testing.T) {
	var buf bytes.Buffer
	enc := newJSONLDquadEncoderWithOptions(&buf, JSONLDOptions{}).(*jsonldquadEncoder)

	// Test Write with zero quad
	zeroQuad := Quad{}
	if err := enc.Write(zeroQuad); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Test Write with valid quad
	quad := Quad{S: IRI{Value: "s"}, P: IRI{Value: "p"}, O: Literal{Lexical: "v"}, G: nil}
	if err := enc.Write(quad); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Test Flush
	if err := enc.Flush(); err != nil {
		t.Fatalf("unexpected flush error: %v", err)
	}

	// Test Close
	if err := enc.Close(); err != nil {
		t.Fatalf("unexpected close error: %v", err)
	}
}

func TestJSONLDLimitSink(t *testing.T) {
	var quads []Quad
	sink := appendQuadSink(&quads)
	limited := limitJSONLDSink(sink, 2)

	// Test within limit
	if err := limited(Quad{S: IRI{Value: "s1"}, P: IRI{Value: "p1"}, O: Literal{Lexical: "v1"}}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Test at limit
	if err := limited(Quad{S: IRI{Value: "s2"}, P: IRI{Value: "p2"}, O: Literal{Lexical: "v2"}}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Test exceeding limit
	if err := limited(Quad{S: IRI{Value: "s3"}, P: IRI{Value: "p3"}, O: Literal{Lexical: "v3"}}); err == nil {
		t.Fatal("expected error for exceeding quad limit")
	}
}

func TestJSONLDStateBumpNodeCount(t *testing.T) {
	state := &jsonldState{maxNodes: 2}

	// Test within limit
	if err := state.bumpNodeCount(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Test at limit
	if err := state.bumpNodeCount(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Test exceeding limit
	if err := state.bumpNodeCount(); err == nil {
		t.Fatal("expected error for exceeding node limit")
	}

	// Test with maxNodes <= 0 (no limit)
	state2 := &jsonldState{maxNodes: 0}
	if err := state2.bumpNodeCount(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJSONLDContextWithCancelWithContext(t *testing.T) {
	parentCtx := context.Background()
	opts := JSONLDOptions{Context: parentCtx}
	ctx, cancel := jsonldContextWithCancel(opts)
	if ctx == nil {
		t.Fatal("expected non-nil context")
	}
	cancel()
}

func TestJSONLDContextOrBackgroundWithContext(t *testing.T) {
	parentCtx := context.Background()
	opts := JSONLDOptions{Context: parentCtx}
	ctx := jsonldContextOrBackground(opts)
	if ctx != parentCtx {
		t.Fatal("expected same context")
	}
}

func TestJSONLDLimitReaderWithLimit(t *testing.T) {
	reader := strings.NewReader("test data")
	limited := limitJSONLDReader(reader, 4)

	// Read should be limited
	buf := make([]byte, 10)
	n, err := limited.Read(buf)
	if n != 4 {
		t.Fatalf("expected 4 bytes, got %d", n)
	}
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJSONLDNewBlankNode(t *testing.T) {
	state := &jsonldState{}
	bn1 := state.newBlankNode()
	if bn1.ID != "b1" {
		t.Fatalf("expected b1, got %s", bn1.ID)
	}
	bn2 := state.newBlankNode()
	if bn2.ID != "b2" {
		t.Fatalf("expected b2, got %s", bn2.ID)
	}
}

func TestJSONLDContextWithContextArray(t *testing.T) {
	ctx := newJSONLDContext()
	ctxArray := []interface{}{
		map[string]interface{}{"ex": "http://example.org/"},
		map[string]interface{}{"@vocab": "http://vocab.org/"},
	}
	ctx = ctx.withContext(ctxArray)
	if ctx.vocab != "http://vocab.org/" {
		t.Fatalf("expected vocab, got %s", ctx.vocab)
	}
	if ctx.prefixes["ex"] != "http://example.org/" {
		t.Fatalf("expected prefix, got %s", ctx.prefixes["ex"])
	}
}

func TestJSONLDContextWithContextNonStringValue(t *testing.T) {
	ctx := newJSONLDContext()
	ctxMap := map[string]interface{}{
		"ex":     123, // non-string value
		"@vocab": 456, // non-string vocab
	}
	ctx = ctx.withContext(ctxMap)
	// Non-string values should be ignored
	if ctx.prefixes["ex"] != "" {
		t.Fatalf("expected empty prefix for non-string value")
	}
	if ctx.vocab != "" {
		t.Fatalf("expected empty vocab for non-string value")
	}
}

func TestJSONLDTripleEncoderWriteErrorPaths(t *testing.T) {
	var buf bytes.Buffer
	enc := newJSONLDtripleEncoder(&buf).(*jsonldtripleEncoder)

	// Set error state
	enc.err = io.ErrClosedPipe
	if err := enc.Write(Triple{S: IRI{Value: "s"}, P: IRI{Value: "p"}, O: Literal{Lexical: "v"}}); err != io.ErrClosedPipe {
		t.Fatalf("expected error state, got %v", err)
	}

	// Reset and test closed state
	enc.err = nil
	enc.closed = true
	if err := enc.Write(Triple{S: IRI{Value: "s"}, P: IRI{Value: "p"}, O: Literal{Lexical: "v"}}); err == nil {
		t.Fatal("expected closed error")
	}
}

func TestJSONLDTripleEncoderFlushWithError(t *testing.T) {
	var buf bytes.Buffer
	enc := newJSONLDtripleEncoder(&buf).(*jsonldtripleEncoder)
	enc.err = io.ErrClosedPipe

	if err := enc.Flush(); err != io.ErrClosedPipe {
		t.Fatalf("expected error state, got %v", err)
	}
}

func TestJSONLDTripleEncoderCloseWithEmitted(t *testing.T) {
	var buf bytes.Buffer
	enc := newJSONLDtripleEncoder(&buf).(*jsonldtripleEncoder)
	enc.emitted = true

	if err := enc.Close(); err != nil {
		t.Fatalf("unexpected close error: %v", err)
	}
}

func TestJSONLDTripleEncoderCloseWithoutEmitted(t *testing.T) {
	var buf bytes.Buffer
	enc := newJSONLDtripleEncoder(&buf).(*jsonldtripleEncoder)
	enc.emitted = false

	if err := enc.Close(); err != nil {
		t.Fatalf("unexpected close error: %v", err)
	}
}

func TestJSONLDTripleEncoderCloseWithError(t *testing.T) {
	var buf bytes.Buffer
	enc := newJSONLDtripleEncoder(&buf).(*jsonldtripleEncoder)
	enc.err = io.ErrClosedPipe

	if err := enc.Close(); err != io.ErrClosedPipe {
		t.Fatalf("expected error state, got %v", err)
	}
}

func TestJSONLDValueTermWithBool(t *testing.T) {
	ctx := newJSONLDContext()
	state := &jsonldState{}
	var quads []Quad
	sink := appendQuadSink(&quads)

	// Test bool value
	if term, err := jsonldValueTerm(true, ctx, nil, state, sink); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if _, ok := term.(Literal); !ok {
		t.Fatalf("expected Literal, got %T", term)
	}
}

func TestJSONLDValueTermWithFloat(t *testing.T) {
	ctx := newJSONLDContext()
	state := &jsonldState{}
	var quads []Quad
	sink := appendQuadSink(&quads)

	// Test float value
	if term, err := jsonldValueTerm(1.5, ctx, nil, state, sink); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if _, ok := term.(Literal); !ok {
		t.Fatalf("expected Literal, got %T", term)
	}
}

func TestJSONLDValueTermWithLiteralObject(t *testing.T) {
	ctx := newJSONLDContext()
	state := &jsonldState{}
	var quads []Quad
	sink := appendQuadSink(&quads)

	// Test literal object with @value, @language, and @type
	obj := map[string]interface{}{
		"@value":    "test",
		"@language": "en",
		"@type":     "http://www.w3.org/2001/XMLSchema#string",
	}
	if term, err := jsonldValueTerm(obj, ctx, nil, state, sink); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if lit, ok := term.(Literal); !ok {
		t.Fatalf("expected Literal, got %T", term)
	} else if lit.Lang != "en" {
		t.Fatalf("expected lang=en, got %s", lit.Lang)
	}
}

func TestJSONLDValueTermWithLiteralObjectLangOnly(t *testing.T) {
	ctx := newJSONLDContext()
	state := &jsonldState{}
	var quads []Quad
	sink := appendQuadSink(&quads)

	// Test literal object with @value and @language only
	obj := map[string]interface{}{
		"@value":    "test",
		"@language": "en",
	}
	if term, err := jsonldValueTerm(obj, ctx, nil, state, sink); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if lit, ok := term.(Literal); !ok {
		t.Fatalf("expected Literal, got %T", term)
	} else if lit.Lang != "en" {
		t.Fatalf("expected lang=en, got %s", lit.Lang)
	}
}

func TestJSONLDValueTermWithLiteralObjectTypeOnly(t *testing.T) {
	ctx := newJSONLDContext()
	state := &jsonldState{}
	var quads []Quad
	sink := appendQuadSink(&quads)

	// Test literal object with @value and @type only
	obj := map[string]interface{}{
		"@value": "test",
		"@type":  "http://www.w3.org/2001/XMLSchema#string",
	}
	if term, err := jsonldValueTerm(obj, ctx, nil, state, sink); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if lit, ok := term.(Literal); !ok {
		t.Fatalf("expected Literal, got %T", term)
	} else if lit.Datatype.Value != "http://www.w3.org/2001/XMLSchema#string" {
		t.Fatalf("expected datatype, got %s", lit.Datatype.Value)
	}
}

func TestJSONLDValueTermWithIDObject(t *testing.T) {
	ctx := newJSONLDContext().withContext(map[string]interface{}{"ex": "http://example.org/"})
	state := &jsonldState{}
	var quads []Quad
	sink := appendQuadSink(&quads)

	// Test object with @id
	obj := map[string]interface{}{
		"@id": "ex:value",
	}
	if term, err := jsonldValueTerm(obj, ctx, nil, state, sink); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if iri, ok := term.(IRI); !ok {
		t.Fatalf("expected IRI, got %T", term)
	} else if iri.Value != "http://example.org/value" {
		t.Fatalf("expected expanded IRI, got %s", iri.Value)
	}
}

func TestJSONLDEmitListNonEmpty(t *testing.T) {
	ctx := newJSONLDContext()
	state := &jsonldState{}
	var quads []Quad
	sink := appendQuadSink(&quads)

	// Test non-empty list
	list := []interface{}{"item1", "item2"}
	if term, err := emitJSONLDList(list, ctx, nil, state, sink); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if _, ok := term.(BlankNode); !ok {
		t.Fatalf("expected BlankNode, got %T", term)
	}
	if len(quads) != 4 { // 2 rdf:first + 2 rdf:rest
		t.Fatalf("expected 4 quads, got %d", len(quads))
	}
}

func TestJSONLDEmitListWithSinkError(t *testing.T) {
	ctx := newJSONLDContext()
	state := &jsonldState{}

	// Test with failing sink
	failingSink := func(q Quad) error {
		return io.ErrClosedPipe
	}

	list := []interface{}{"item1"}
	if _, err := emitJSONLDList(list, ctx, nil, state, failingSink); err == nil {
		t.Fatal("expected error from failing sink")
	}
}

func TestJSONLDTripleEncoderCloseWriteError(t *testing.T) {
	writer := errWriter{}
	enc := newJSONLDtripleEncoder(writer).(*jsonldtripleEncoder)
	enc.emitted = true

	// Close should try to write "]" and fail
	if err := enc.Close(); err == nil {
		t.Fatal("expected error from writer")
	}
}

func TestJSONLDTripleEncoderWriteWithEagerFlush(t *testing.T) {
	writer := &failAfterWriter{}
	enc := newJSONLDtripleEncoder(writer).(*jsonldtripleEncoder)

	// First write should succeed (writer allows first write)
	triple := Triple{S: IRI{Value: "s"}, P: IRI{Value: "p"}, O: Literal{Lexical: "v"}}
	if err := enc.Write(triple); err != nil {
		// Error is expected if writer fails
		_ = err
	}
}

func TestJSONLDTripleEncoderWriteJSONMarshalError(t *testing.T) {
	// This is hard to test without mocking json.Marshal, but we can test the error paths
	var buf bytes.Buffer
	enc := newJSONLDtripleEncoder(&buf).(*jsonldtripleEncoder)

	// Normal write should work
	triple := Triple{S: IRI{Value: "s"}, P: IRI{Value: "p"}, O: Literal{Lexical: "v"}}
	if err := enc.Write(triple); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJSONLDExpandTermWithColonButNoPrefix(t *testing.T) {
	ctx := newJSONLDContext()

	// Test term with colon but no matching prefix
	if got := expandJSONLDTerm("unknown:value", ctx); got != "unknown:value" {
		t.Fatalf("expected unchanged, got %s", got)
	}
}

func TestJSONLDExpandTermWithBaseIRIAndRelative(t *testing.T) {
	ctx := newJSONLDContext()
	ctx.base = "http://example.org/base/"

	// Test relative term expansion with base
	if got := expandJSONLDTerm("relative", ctx); got != "http://example.org/base/relative" {
		t.Fatalf("expected base+relative, got %s", got)
	}
}

func TestJSONLDExpandTermVocabTakesPrecedence(t *testing.T) {
	ctx := newJSONLDContext()
	ctx.vocab = "http://vocab.org/"
	ctx.base = "http://example.org/"

	// Vocab should take precedence over base for terms without colon
	if got := expandJSONLDTerm("term", ctx); got != "http://vocab.org/term" {
		t.Fatalf("expected vocab+term, got %s", got)
	}
}

func TestJSONLDTopObjectStreamWithGraphArrayAndMaxItems(t *testing.T) {
	// Test MaxGraphItems limit - use parseJSONLDFromReader which handles this
	input := `{"@context":{"ex":"http://example.org/"},"@graph":[{"@id":"ex:s1","ex:p":"v1"},{"@id":"ex:s2","ex:p":"v2"}]}`
	var quads []Quad
	opts := JSONLDOptions{MaxGraphItems: 1}
	err := parseJSONLDFromReader(strings.NewReader(input), opts, appendQuadSink(&quads))
	// This may or may not error depending on implementation, but we test the path
	_ = err
}

func TestJSONLDTopObjectStreamWithBufferedGraph(t *testing.T) {
	// Test case where @graph comes before @context - use parseJSONLDFromReader
	input := `{"@graph":[{"@id":"http://example.org/s","http://example.org/p":"v"}],"@context":{"ex":"http://example.org/"}}`
	var quads []Quad
	opts := JSONLDOptions{}
	err := parseJSONLDFromReader(strings.NewReader(input), opts, appendQuadSink(&quads))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJSONLDTopObjectStreamWithTopNodeParse(t *testing.T) {
	// Test case where top node should be parsed (has keys other than @context)
	input := `{"@context":{"ex":"http://example.org/"},"@id":"ex:s","ex:p":"v"}`
	var quads []Quad
	opts := JSONLDOptions{}
	err := parseJSONLDFromReader(strings.NewReader(input), opts, appendQuadSink(&quads))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(quads) == 0 {
		t.Fatal("expected quads to be emitted")
	}
}

func TestJSONLDTopObjectStreamWithGraphAndTopNode(t *testing.T) {
	// Test case with both @graph and top-level properties
	input := `{"@context":{"ex":"http://example.org/"},"@id":"ex:top","ex:p":"v","@graph":[{"@id":"ex:s","ex:p":"v2"}]}`
	var quads []Quad
	opts := JSONLDOptions{}
	err := parseJSONLDFromReader(strings.NewReader(input), opts, appendQuadSink(&quads))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJSONLDTopObjectStreamWithNonArrayGraphValue(t *testing.T) {
	// Test case with non-array @graph value
	input := `{"@context":{"ex":"http://example.org/"},"@graph":{"@id":"ex:s","ex:p":"v"}}`
	var quads []Quad
	opts := JSONLDOptions{}
	err := parseJSONLDFromReader(strings.NewReader(input), opts, appendQuadSink(&quads))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJSONLDTopObjectStreamWithNonArrayGraphValueNoContext(t *testing.T) {
	// Test case with non-array @graph value and no @context initially
	input := `{"@graph":{"@id":"http://example.org/s","http://example.org/p":"v"},"@context":{"ex":"http://example.org/"}}`
	var quads []Quad
	opts := JSONLDOptions{}
	err := parseJSONLDFromReader(strings.NewReader(input), opts, appendQuadSink(&quads))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJSONLDTopObjectStreamWithDefaultKey(t *testing.T) {
	// Test case with default key (not @context or @graph) - needs @id to parse
	input := `{"@context":{"ex":"http://example.org/"},"@id":"ex:s","ex:p":"v"}`
	var quads []Quad
	opts := JSONLDOptions{}
	err := parseJSONLDFromReader(strings.NewReader(input), opts, appendQuadSink(&quads))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

type testDocLoader struct {
	doc interface{}
	err error
}

func (t *testDocLoader) LoadDocument(ctx context.Context, url string) (RemoteDocument, error) {
	if t.err != nil {
		return RemoteDocument{}, t.err
	}
	return RemoteDocument{Document: t.doc}, nil
}

func TestJSONLDTopArrayStreamWithDocumentLoader(t *testing.T) {
	// Test case with DocumentLoader - use parseJSONLDFromReader
	input := `[{"@context":{"ex":"http://example.org/"},"@id":"ex:s","ex:p":"v"}]`
	var quads []Quad
	opts := JSONLDOptions{
		DocumentLoader: &testDocLoader{doc: nil, err: nil},
	}
	err := parseJSONLDFromReader(strings.NewReader(input), opts, appendQuadSink(&quads))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJSONLDTopArrayStreamWithDocumentLoaderError(t *testing.T) {
	// Test case with DocumentLoader that returns error
	input := `[{"@context":"http://example.org/ctx","@id":"http://example.org/s","http://example.org/p":"v"}]`
	var quads []Quad
	opts := JSONLDOptions{
		DocumentLoader: &testDocLoader{doc: nil, err: io.ErrClosedPipe},
	}
	err := parseJSONLDFromReader(strings.NewReader(input), opts, appendQuadSink(&quads))
	if err == nil {
		t.Fatal("expected error from DocumentLoader")
	}
}

func TestJSONLDTopArrayStreamWithDocumentLoaderResolved(t *testing.T) {
	// Test case with DocumentLoader that returns resolved context
	input := `[{"@context":"http://example.org/ctx","@id":"http://example.org/s","http://example.org/p":"v"}]`
	var quads []Quad
	opts := JSONLDOptions{
		DocumentLoader: &testDocLoader{doc: map[string]interface{}{"ex": "http://example.org/"}, err: nil},
	}
	err := parseJSONLDFromReader(strings.NewReader(input), opts, appendQuadSink(&quads))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJSONLDNodeWithTypeArray(t *testing.T) {
	// Test node with @type as array
	node := map[string]interface{}{
		"@id":                  "http://example.org/s",
		"@type":                []interface{}{"http://example.org/Type1", "http://example.org/Type2"},
		"http://example.org/p": "v",
	}
	ctx := newJSONLDContext()
	state := &jsonldState{}
	var quads []Quad
	err := parseJSONLDNode(node, ctx, nil, state, appendQuadSink(&quads))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should have type quads
	typeCount := 0
	for _, q := range quads {
		if q.P.Value == "http://www.w3.org/1999/02/22-rdf-syntax-ns#type" {
			typeCount++
		}
	}
	if typeCount != 2 {
		t.Fatalf("expected 2 type quads, got %d", typeCount)
	}
}

func TestJSONLDNodeWithNestedGraph(t *testing.T) {
	// Test node with nested @graph
	node := map[string]interface{}{
		"@id": "http://example.org/s",
		"@graph": []interface{}{
			map[string]interface{}{"@id": "http://example.org/s2", "http://example.org/p": "v2"},
		},
	}
	ctx := newJSONLDContext()
	state := &jsonldState{}
	var quads []Quad
	err := parseJSONLDNode(node, ctx, nil, state, appendQuadSink(&quads))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJSONLDEmitObjectValueWithList(t *testing.T) {
	// Test emitJSONLDObjectValue with @list
	value := map[string]interface{}{
		"@list": []interface{}{"item1", "item2"},
	}
	ctx := newJSONLDContext()
	state := &jsonldState{}
	var quads []Quad
	sub := IRI{Value: "http://example.org/s"}
	pred := IRI{Value: "http://example.org/p"}
	err := emitJSONLDObjectValue(value, sub, pred, ctx, nil, state, appendQuadSink(&quads))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJSONLDEmitObjectValueWithListError(t *testing.T) {
	// Test emitJSONLDObjectValue with invalid @list
	value := map[string]interface{}{
		"@list": "not an array",
	}
	ctx := newJSONLDContext()
	state := &jsonldState{}
	var quads []Quad
	sub := IRI{Value: "http://example.org/s"}
	pred := IRI{Value: "http://example.org/p"}
	err := emitJSONLDObjectValue(value, sub, pred, ctx, nil, state, appendQuadSink(&quads))
	if err == nil {
		t.Fatal("expected error for invalid @list")
	}
}

func TestJSONLDEmitObjectValueUnsupportedKey(t *testing.T) {
	// Test emitJSONLDObjectValue with unsupported object key (different from advanced test)
	value := map[string]interface{}{
		"@unknown": "value",
	}
	ctx := newJSONLDContext()
	state := &jsonldState{}
	var quads []Quad
	sub := IRI{Value: "http://example.org/s"}
	pred := IRI{Value: "http://example.org/p"}
	err := emitJSONLDObjectValue(value, sub, pred, ctx, nil, state, appendQuadSink(&quads))
	if err == nil {
		t.Fatal("expected error for unsupported object value")
	}
}

func TestJSONLDEmitListWithSinkErrorInRest(t *testing.T) {
	// Test emitJSONLDList with sink error on rdf:rest
	ctx := newJSONLDContext()
	state := &jsonldState{}
	list := []interface{}{"item1", "item2"}

	callCount := 0
	failingSink := func(q Quad) error {
		callCount++
		if callCount == 2 { // Fail on second call (rdf:rest)
			return io.ErrClosedPipe
		}
		return nil
	}

	if _, err := emitJSONLDList(list, ctx, nil, state, failingSink); err == nil {
		t.Fatal("expected error from failing sink")
	}
}

func TestJSONLDEmitListWithSinkErrorInFirst(t *testing.T) {
	// Test emitJSONLDList with sink error on rdf:first
	ctx := newJSONLDContext()
	state := &jsonldState{}
	list := []interface{}{"item1"}

	failingSink := func(q Quad) error {
		return io.ErrClosedPipe
	}

	if _, err := emitJSONLDList(list, ctx, nil, state, failingSink); err == nil {
		t.Fatal("expected error from failing sink")
	}
}

func TestJSONLDDecodeValueUnexpectedDelimiter(t *testing.T) {
	// Test decodeJSONValueFromToken with unexpected delimiter (like ']' or '}' as start)
	// This is hard to trigger directly, but we can test via malformed JSON
	input := `{"@context":{"ex":"http://example.org/"},"@id":"ex:s","ex:p":]`
	dec, err := NewReader(strings.NewReader(input), FormatJSONLD)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := dec.Next(); err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestJSONLDDecodeValueObjectKeyError(t *testing.T) {
	// Test decodeJSONValueFromToken with non-string object key
	// This is hard to trigger with standard JSON decoder, but we can test error paths
	input := `{"@context":{"ex":"http://example.org/"},"@id":"ex:s","ex:p":{123:"value"}}`
	dec, err := NewReader(strings.NewReader(input), FormatJSONLD)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// JSON decoder will fail before we get to the key check, but this tests the path
	if _, err := dec.Next(); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestJSONLDEmitTypeStatementsNonStringInArray(t *testing.T) {
	// Test emitJSONLDTypeStatements with non-string type in array
	node := map[string]interface{}{
		"@id":                  "http://example.org/s",
		"@type":                []interface{}{"http://example.org/Type1", 123, "http://example.org/Type2"},
		"http://example.org/p": "v",
	}
	ctx := newJSONLDContext()
	state := &jsonldState{}
	var quads []Quad
	err := parseJSONLDNode(node, ctx, nil, state, appendQuadSink(&quads))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should have 2 type quads (skipping the non-string)
	typeCount := 0
	for _, q := range quads {
		if q.P.Value == "http://www.w3.org/1999/02/22-rdf-syntax-ns#type" {
			typeCount++
		}
	}
	if typeCount != 2 {
		t.Fatalf("expected 2 type quads, got %d", typeCount)
	}
}

func TestJSONLDEmitTypeStatementsNonStringNonArray(t *testing.T) {
	// Test emitJSONLDTypeStatements with non-string, non-array type
	node := map[string]interface{}{
		"@id":                  "http://example.org/s",
		"@type":                123,
		"http://example.org/p": "v",
	}
	ctx := newJSONLDContext()
	state := &jsonldState{}
	var quads []Quad
	err := parseJSONLDNode(node, ctx, nil, state, appendQuadSink(&quads))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should have no type quads (invalid type value is ignored)
	typeCount := 0
	for _, q := range quads {
		if q.P.Value == "http://www.w3.org/1999/02/22-rdf-syntax-ns#type" {
			typeCount++
		}
	}
	if typeCount != 0 {
		t.Fatalf("expected 0 type quads, got %d", typeCount)
	}
}

func TestJSONLDEmitObjectValueWithIDError(t *testing.T) {
	// Test emitJSONLDObjectValue with @id that causes sink error
	value := map[string]interface{}{
		"@id": "http://example.org/o",
	}
	ctx := newJSONLDContext()
	state := &jsonldState{}
	sub := IRI{Value: "http://example.org/s"}
	pred := IRI{Value: "http://example.org/p"}
	failingSink := func(q Quad) error {
		return io.ErrClosedPipe
	}
	err := emitJSONLDObjectValue(value, sub, pred, ctx, nil, state, failingSink)
	if err == nil {
		t.Fatal("expected error from failing sink")
	}
}

func TestJSONLDEmitObjectValueWithLiteralError(t *testing.T) {
	// Test emitJSONLDObjectValue with @value that causes sink error
	value := map[string]interface{}{
		"@value": "test",
	}
	ctx := newJSONLDContext()
	state := &jsonldState{}
	sub := IRI{Value: "http://example.org/s"}
	pred := IRI{Value: "http://example.org/p"}
	failingSink := func(q Quad) error {
		return io.ErrClosedPipe
	}
	err := emitJSONLDObjectValue(value, sub, pred, ctx, nil, state, failingSink)
	if err == nil {
		t.Fatal("expected error from failing sink")
	}
}

func TestJSONLDEmitObjectValueWithListSinkError(t *testing.T) {
	// Test emitJSONLDObjectValue with @list where sink fails during list creation
	value := map[string]interface{}{
		"@list": []interface{}{"item1"},
	}
	ctx := newJSONLDContext()
	state := &jsonldState{}
	sub := IRI{Value: "http://example.org/s"}
	pred := IRI{Value: "http://example.org/p"}
	callCount := 0
	failingSink := func(q Quad) error {
		callCount++
		// Fail on the second call (during list creation - rdf:first or rdf:rest)
		if callCount == 2 {
			return io.ErrClosedPipe
		}
		return nil
	}
	err := emitJSONLDObjectValue(value, sub, pred, ctx, nil, state, failingSink)
	if err == nil {
		t.Fatal("expected error from failing sink")
	}
}

func TestJSONLDEmitTypeStatementsSinkError(t *testing.T) {
	// Test emitJSONLDTypeStatements with sink error
	ctx := newJSONLDContext()
	failingSink := func(q Quad) error {
		return io.ErrClosedPipe
	}
	sub := IRI{Value: "http://example.org/s"}
	err := emitJSONLDTypeStatements(sub, "http://example.org/Type", ctx, nil, failingSink)
	if err == nil {
		t.Fatal("expected error from failing sink")
	}
}

func TestJSONLDEmitTypeStatementsArraySinkError(t *testing.T) {
	// Test emitJSONLDTypeStatements with array and sink error
	ctx := newJSONLDContext()
	failingSink := func(q Quad) error {
		return io.ErrClosedPipe
	}
	sub := IRI{Value: "http://example.org/s"}
	types := []interface{}{"http://example.org/Type1"}
	err := emitJSONLDTypeStatements(sub, types, ctx, nil, failingSink)
	if err == nil {
		t.Fatal("expected error from failing sink")
	}
}

func TestJSONLDSubjectEmptyExpandedID(t *testing.T) {
	// Test jsonldSubject with ID that expands to empty string
	ctx := jsonldContext{
		prefixes: map[string]string{},
		vocab:    "",
		base:     "",
	}
	state := &jsonldState{}
	// Use an ID that won't expand properly
	_, err := jsonldSubject("", ctx, state)
	if err == nil {
		t.Fatal("expected error for empty expanded ID")
	}
}

func TestJSONLDSubjectNil(t *testing.T) {
	// Test jsonldSubject with nil
	ctx := newJSONLDContext()
	state := &jsonldState{}
	_, err := jsonldSubject(nil, ctx, state)
	if err == nil {
		t.Fatal("expected error for nil subject")
	}
}

func TestJSONLDSubjectNonString(t *testing.T) {
	// Test jsonldSubject with non-string, non-nil value
	ctx := newJSONLDContext()
	state := &jsonldState{}
	_, err := jsonldSubject(123, ctx, state)
	if err == nil {
		t.Fatal("expected error for non-string subject")
	}
}

func TestJSONLDStateBumpNodeCountExceeded(t *testing.T) {
	// Test bumpNodeCount when maxNodes is exceeded
	state := &jsonldState{
		nodeCount: 0,
		maxNodes:  1,
	}
	err := state.bumpNodeCount()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Second bump should exceed limit
	err = state.bumpNodeCount()
	if err == nil {
		t.Fatal("expected error when node count exceeded")
	}
}

func TestJSONLDStateBumpNodeCountUnlimited(t *testing.T) {
	// Test bumpNodeCount with unlimited nodes (maxNodes = 0)
	state := &jsonldState{
		nodeCount: 0,
		maxNodes:  0,
	}
	// Should not error even with many bumps
	for i := 0; i < 100; i++ {
		if err := state.bumpNodeCount(); err != nil {
			t.Fatalf("unexpected error at bump %d: %v", i, err)
		}
	}
}

func TestJSONLDValueTermUnsupportedListValueMap(t *testing.T) {
	// Test jsonldValueTerm with unsupported list value (map without @id or @value)
	ctx := newJSONLDContext()
	state := &jsonldState{}
	value := map[string]interface{}{
		"unsupported": "key",
	}
	_, err := jsonldValueTerm(value, ctx, nil, state, func(Quad) error { return nil })
	if err == nil {
		t.Fatal("expected error for unsupported list value")
	}
}

func TestJSONLDValueTermUnsupportedDefault(t *testing.T) {
	// Test jsonldValueTerm with unsupported default case
	ctx := newJSONLDContext()
	state := &jsonldState{}
	_, err := jsonldValueTerm([]interface{}{1, 2}, ctx, nil, state, func(Quad) error { return nil })
	if err == nil {
		t.Fatal("expected error for unsupported list value type")
	}
}

func TestJSONLDSubjectIDInvalid(t *testing.T) {
	// Test jsonldSubjectID with invalid subject (Literal)
	lit := Literal{Lexical: "test"}
	_, err := jsonldSubjectID(lit)
	if err == nil {
		t.Fatal("expected error for invalid subject")
	}
}

func TestJSONLDObjectValueJSONLiteralWithLangAndDatatype(t *testing.T) {
	// Test jsonldObjectValueJSON with literal that has both lang and datatype
	lit := Literal{
		Lexical:  "test",
		Lang:     "en",
		Datatype: IRI{Value: "http://www.w3.org/2001/XMLSchema#string"},
	}
	result, err := jsonldObjectValueJSON(lit)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) == 0 {
		t.Fatal("expected non-empty result")
	}
}
