package rdf

import (
	"bytes"
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
