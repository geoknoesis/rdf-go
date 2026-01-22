package rdf

import (
	"bufio"
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestJSONLDDecoderErrClose(t *testing.T) {
	dec := &jsonldDecoder{err: io.ErrUnexpectedEOF}
	if _, err := dec.Next(); err != io.ErrUnexpectedEOF {
		t.Fatalf("expected err, got %v", err)
	}
	if dec.Err() != io.ErrUnexpectedEOF {
		t.Fatalf("expected Err() to return underlying error")
	}
	if err := dec.Close(); err != nil {
		t.Fatalf("expected Close to be nil, got %v", err)
	}
}

func TestJSONLDLoad_ArrayTopLevel(t *testing.T) {
	input := `[{"@context":{"ex":"http://example.org/"},"@id":"ex:s","ex:p":"v"}]`
	dec := newJSONLDDecoder(strings.NewReader(input))
	if _, err := dec.Next(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJSONLDWithVocab(t *testing.T) {
	input := `{"@context":{"@vocab":"http://example.org/"},"@id":"thing","p":"v"}`
	dec := newJSONLDDecoder(strings.NewReader(input))
	quad, err := dec.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if quad.P.Value != "http://example.org/p" {
		t.Fatalf("unexpected predicate: %s", quad.P.Value)
	}
}

func TestJSONLDValueTypes(t *testing.T) {
	input := `{"@context":{"ex":"http://example.org/"},"@id":"ex:s","ex:p":[{"@id":"ex:o"},{"@value":"x"},1,true,"str"]}`
	dec := newJSONLDDecoder(strings.NewReader(input))
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
	dec := newJSONLDDecoder(strings.NewReader(input))
	if _, err := dec.Next(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJSONLDErrorUnsupportedLiteral(t *testing.T) {
	input := `{"@context":{"ex":"http://example.org/"},"@id":"ex:s","ex:p":null}`
	dec := newJSONLDDecoder(strings.NewReader(input))
	if _, err := dec.Next(); err == nil {
		t.Fatal("expected error for unsupported literal value")
	}
}

func TestJSONLDEncoderErrors(t *testing.T) {
	enc := newJSONLDEncoder(failingWriter{})
	if err := enc.Write(Quad{S: IRI{Value: "s"}, P: IRI{Value: "p"}, O: Literal{Lexical: "v"}}); err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}
	if err := enc.Flush(); err == nil {
		t.Fatal("expected flush error")
	}

	var buf bytes.Buffer
	enc = newJSONLDEncoder(&buf)
	_ = enc.Write(Quad{S: IRI{Value: "s"}, P: IRI{Value: "p"}, O: Literal{Lexical: "v"}})
	_ = enc.Write(Quad{S: IRI{Value: "s2"}, P: IRI{Value: "p2"}, O: IRI{Value: "o2"}})
	if err := enc.Close(); err != nil {
		t.Fatalf("unexpected close error: %v", err)
	}
	enc = newJSONLDEncoder(&buf)
	_ = enc.Close()
	if err := enc.Write(Quad{S: IRI{Value: "s"}, P: IRI{Value: "p"}, O: Literal{Lexical: "v"}}); err == nil {
		t.Fatal("expected closed error")
	}
}

func TestJSONLDDecoderEOF(t *testing.T) {
	input := `{"@context":{"ex":"http://example.org/"},"@id":"ex:s","ex:p":"v"}`
	dec := newJSONLDDecoder(strings.NewReader(input))
	if _, err := dec.Next(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := dec.Next(); err != io.EOF {
		t.Fatalf("expected EOF, got %v", err)
	}
}

func TestJSONLDLoadNonCollection(t *testing.T) {
	dec := newJSONLDDecoder(strings.NewReader("5"))
	if _, err := dec.Next(); err != io.EOF {
		t.Fatalf("expected EOF for non-collection, got %v", err)
	}
}

func TestJSONLDLoadArraySkipsNonObjects(t *testing.T) {
	input := `[{"@context":{"ex":"http://example.org/"},"@id":"ex:s","ex:p":"v"}, "skip"]`
	dec := newJSONLDDecoder(strings.NewReader(input))
	if _, err := dec.Next(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJSONLDLoadGraphError(t *testing.T) {
	input := `{"@context":{"ex":"http://example.org/"},"@graph":{"ex:p":"v"}}`
	dec := newJSONLDDecoder(strings.NewReader(input))
	if _, err := dec.Next(); err == nil {
		t.Fatal("expected graph parse error")
	}
}

func TestJSONLDLoadNodeError(t *testing.T) {
	input := `{"@context":{"ex":"http://example.org/"},"ex:p":"v"}`
	dec := newJSONLDDecoder(strings.NewReader(input))
	if _, err := dec.Next(); err == nil {
		t.Fatal("expected node parse error")
	}
}

func TestJSONLDGraphUnsupportedType(t *testing.T) {
	var quads []Quad
	if err := parseJSONLDGraph("bad", newJSONLDContext(), &quads); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestJSONLDGraphErrorPath(t *testing.T) {
	var quads []Quad
	err := parseJSONLDGraph(map[string]interface{}{"ex:p": "v"}, newJSONLDContext(), &quads)
	if err == nil {
		t.Fatal("expected graph parse error")
	}
}

func TestJSONLDGraphArraySkipsInvalid(t *testing.T) {
	var quads []Quad
	err := parseJSONLDGraph([]interface{}{"skip"}, newJSONLDContext(), &quads)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJSONLDGraphArrayError(t *testing.T) {
	var quads []Quad
	err := parseJSONLDGraph([]interface{}{map[string]interface{}{"ex:p": "v"}}, newJSONLDContext(), &quads)
	if err == nil {
		t.Fatal("expected error for graph node without @id")
	}
}

func TestJSONLDLoadEmptyArray(t *testing.T) {
	dec := newJSONLDDecoder(strings.NewReader("[]"))
	if _, err := dec.Next(); err != io.EOF {
		t.Fatalf("expected EOF, got %v", err)
	}
}

func TestJSONLDLoadArrayNodeError(t *testing.T) {
	input := `[{"ex:p":"v"}]`
	dec := newJSONLDDecoder(strings.NewReader(input))
	if _, err := dec.Next(); err == nil {
		t.Fatal("expected node error in array")
	}
}

func TestJSONLDLoadDecodeError(t *testing.T) {
	dec := newJSONLDDecoder(strings.NewReader("{"))
	if _, err := dec.Next(); err == nil {
		t.Fatal("expected decode error")
	}
}

func TestJSONLDPredicateResolutionError(t *testing.T) {
	node := map[string]interface{}{
		"@id": "http://example.org/s",
		"":    "v",
	}
	if err := parseJSONLDNode(node, newJSONLDContext(), &[]Quad{}); err == nil {
		t.Fatal("expected predicate resolution error")
	}
}

func TestJSONLDEncoderErrorState(t *testing.T) {
	enc := newJSONLDEncoder(failingWriter{})
	enc.(*jsonldEncoder).err = io.ErrClosedPipe
	if err := enc.Write(Quad{S: IRI{Value: "s"}, P: IRI{Value: "p"}, O: Literal{Lexical: "v"}}); err == nil {
		t.Fatal("expected cached error")
	}
	if err := enc.Flush(); err == nil {
		t.Fatal("expected cached flush error")
	}
	if err := enc.Close(); err == nil {
		t.Fatal("expected cached close error")
	}
}

func TestJSONLDObjectValueBranches(t *testing.T) {
	if got := jsonldObjectValue(IRI{Value: "http://example.org/o"}); !strings.Contains(got, "@id") {
		t.Fatalf("expected @id value, got %s", got)
	}
	if got := jsonldObjectValue(Literal{Lexical: "v"}); !strings.Contains(got, "@value") {
		t.Fatalf("expected @value literal, got %s", got)
	}
	if got := jsonldObjectValue(customTerm{}); got == "" {
		t.Fatalf("expected fallback value, got %s", got)
	}
}

func TestJSONLDEmitArrayBranch(t *testing.T) {
	var quads []Quad
	ctx := newJSONLDContext().withContext(map[string]interface{}{"ex": "http://example.org/"})
	sub := IRI{Value: "http://example.org/s"}
	pred := IRI{Value: "http://example.org/p"}
	value := []interface{}{"v1", "v2"}
	if err := emitJSONLDValue(sub, pred, value, ctx, &quads); err != nil {
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
	if err := emitJSONLDValue(sub, pred, value, ctx, &quads); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJSONLDEmitValueUnsupportedMap(t *testing.T) {
	var quads []Quad
	ctx := newJSONLDContext()
	sub := IRI{Value: "http://example.org/s"}
	pred := IRI{Value: "http://example.org/p"}
	value := map[string]interface{}{"bad": "x"}
	if err := emitJSONLDValue(sub, pred, value, ctx, &quads); err == nil {
		t.Fatal("expected unsupported object error")
	}
}

func TestJSONLDEmitValueScalarBranches(t *testing.T) {
	var quads []Quad
	ctx := newJSONLDContext()
	sub := IRI{Value: "http://example.org/s"}
	pred := IRI{Value: "http://example.org/p"}
	if err := emitJSONLDValue(sub, pred, 1.5, ctx, &quads); err != nil {
		t.Fatalf("unexpected float error: %v", err)
	}
	if err := emitJSONLDValue(sub, pred, true, ctx, &quads); err != nil {
		t.Fatalf("unexpected bool error: %v", err)
	}
}

func TestJSONLDEmitArrayError(t *testing.T) {
	var quads []Quad
	ctx := newJSONLDContext()
	sub := IRI{Value: "http://example.org/s"}
	pred := IRI{Value: "http://example.org/p"}
	value := []interface{}{map[string]interface{}{"bad": "x"}}
	if err := emitJSONLDValue(sub, pred, value, ctx, &quads); err == nil {
		t.Fatal("expected error for invalid array value")
	}
}

func TestJSONLDWriteErrors(t *testing.T) {
	enc := newJSONLDEncoder(failingWriter{}).(*jsonldEncoder)
	enc.writer = bufio.NewWriterSize(failingWriter{}, 1)
	if err := enc.Write(Quad{S: IRI{Value: "s"}, P: IRI{Value: "p"}, O: Literal{Lexical: "v"}}); err == nil {
		t.Fatal("expected write error")
	}
	enc = newJSONLDEncoder(failingWriter{}).(*jsonldEncoder)
	enc.writer = bufio.NewWriterSize(failingWriter{}, 1)
	_ = enc.Write(Quad{S: IRI{Value: "s"}, P: IRI{Value: "p"}, O: Literal{Lexical: "v"}})
	if err := enc.Close(); err == nil {
		t.Fatal("expected close error")
	}
	enc = newJSONLDEncoder(&bytes.Buffer{}).(*jsonldEncoder)
	_ = enc.Close()
	if err := enc.Close(); err != nil {
		t.Fatalf("expected close idempotent: %v", err)
	}
}

type errWriter struct{}

func (errWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestJSONLDWriteCommaError(t *testing.T) {
	enc := newJSONLDEncoder(&bytes.Buffer{}).(*jsonldEncoder)
	_ = enc.Write(Quad{S: IRI{Value: "s"}, P: IRI{Value: "p"}, O: Literal{Lexical: "v"}})
	enc.writer = bufio.NewWriterSize(errWriter{}, 1)
	if err := enc.Write(Quad{S: IRI{Value: "s2"}, P: IRI{Value: "p2"}, O: Literal{Lexical: "v2"}}); err == nil {
		t.Fatal("expected comma write error")
	}
}

func TestJSONLDWriteOpenError(t *testing.T) {
	enc := newJSONLDEncoder(&bytes.Buffer{}).(*jsonldEncoder)
	enc.writer = bufio.NewWriterSize(errWriter{}, 1)
	if err := enc.Write(Quad{S: IRI{Value: "s"}, P: IRI{Value: "p"}, O: Literal{Lexical: "v"}}); err == nil {
		t.Fatal("expected opening write error")
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
	enc := newJSONLDEncoder(&bytes.Buffer{}).(*jsonldEncoder)
	enc.writer = bufio.NewWriterSize(writer, 1)
	if err := enc.Write(Quad{S: IRI{Value: "s"}, P: IRI{Value: "p"}, O: Literal{Lexical: "v"}}); err == nil {
		t.Fatal("expected fragment write error")
	}
}

func TestJSONLDCloseWriteError(t *testing.T) {
	enc := newJSONLDEncoder(&bytes.Buffer{}).(*jsonldEncoder)
	_ = enc.Write(Quad{S: IRI{Value: "s"}, P: IRI{Value: "p"}, O: Literal{Lexical: "v"}})
	enc.writer = bufio.NewWriterSize(errWriter{}, 1)
	if err := enc.Close(); err == nil {
		t.Fatal("expected close write error")
	}
}

func TestJSONLDCloseWithErrorState(t *testing.T) {
	enc := newJSONLDEncoder(&bytes.Buffer{}).(*jsonldEncoder)
	enc.err = io.ErrClosedPipe
	enc.closed = true
	if err := enc.Close(); err == nil {
		t.Fatal("expected cached error on close")
	}
}
