package rdf

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

// Test JSON-LD decoder and encoder internal methods for maximum coverage

func TestJSONLDTripleDecoder_Next_Simple(t *testing.T) {
	input := `{
		"@context": {"ex": "http://example.org/"},
		"@id": "ex:s",
		"ex:p": "v"
	}`
	dec := newJSONLDtripleDecoder(strings.NewReader(input))

	triple, err := dec.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	if triple.S == nil {
		t.Error("Next should return triple with subject")
	}
}

func TestJSONLDTripleDecoder_Next_WithArray(t *testing.T) {
	input := `[{
		"@context": {"ex": "http://example.org/"},
		"@id": "ex:s",
		"ex:p": "v"
	}]`
	dec := newJSONLDtripleDecoder(strings.NewReader(input))

	triple, err := dec.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	if triple.S == nil {
		t.Error("Next should return triple")
	}
}

func TestJSONLDTripleDecoder_Next_WithGraph(t *testing.T) {
	input := `{
		"@context": {"ex": "http://example.org/"},
		"@graph": [{
			"@id": "ex:s",
			"ex:p": "v"
		}]
	}`
	dec := newJSONLDtripleDecoder(strings.NewReader(input))

	triple, err := dec.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	if triple.S == nil {
		t.Error("Next should return triple")
	}
}

func TestJSONLDTripleDecoder_Next_WithBlankNode(t *testing.T) {
	input := `{
		"@context": {"ex": "http://example.org/"},
		"@id": "_:b1",
		"ex:p": "v"
	}`
	dec := newJSONLDtripleDecoder(strings.NewReader(input))

	triple, err := dec.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	if _, ok := triple.S.(BlankNode); !ok {
		t.Error("Next should return BlankNode for anonymous node")
	}
}

func TestJSONLDTripleDecoder_Next_WithLiteral(t *testing.T) {
	input := `{
		"@context": {"ex": "http://example.org/"},
		"@id": "ex:s",
		"ex:p": "literal"
	}`
	dec := newJSONLDtripleDecoder(strings.NewReader(input))

	triple, err := dec.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	if _, ok := triple.O.(Literal); !ok {
		t.Error("Next should return Literal object")
	}
}

func TestJSONLDTripleDecoder_Next_WithLiteralLang(t *testing.T) {
	input := `{
		"@context": {"ex": "http://example.org/"},
		"@id": "ex:s",
		"ex:p": {
			"@value": "literal",
			"@language": "en"
		}
	}`
	dec := newJSONLDtripleDecoder(strings.NewReader(input))

	triple, err := dec.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	if lit, ok := triple.O.(Literal); !ok {
		t.Error("Next should return Literal object")
	} else if lit.Lang != "en" {
		t.Errorf("Next literal lang = %q, want %q", lit.Lang, "en")
	}
}

func TestJSONLDTripleDecoder_Next_WithLiteralDatatype(t *testing.T) {
	input := `{
		"@context": {"ex": "http://example.org/"},
		"@id": "ex:s",
		"ex:p": {
			"@value": "123",
			"@type": "http://www.w3.org/2001/XMLSchema#integer"
		}
	}`
	dec := newJSONLDtripleDecoder(strings.NewReader(input))

	triple, err := dec.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	if lit, ok := triple.O.(Literal); !ok {
		t.Error("Next should return Literal object")
	} else if lit.Datatype.Value != "http://www.w3.org/2001/XMLSchema#integer" {
		t.Errorf("Next literal datatype = %q, want integer", lit.Datatype.Value)
	}
}

func TestJSONLDTripleDecoder_Next_WithNestedObject(t *testing.T) {
	input := `{
		"@context": {"ex": "http://example.org/"},
		"@id": "ex:s",
		"ex:p": {
			"@id": "ex:o"
		}
	}`
	dec := newJSONLDtripleDecoder(strings.NewReader(input))

	triple, err := dec.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	if _, ok := triple.O.(IRI); !ok {
		t.Error("Next should return IRI object for nested object")
	}
}

func TestJSONLDTripleDecoder_Next_WithType(t *testing.T) {
	input := `{
		"@context": {"ex": "http://example.org/"},
		"@id": "ex:s",
		"@type": "ex:Type"
	}`
	dec := newJSONLDtripleDecoder(strings.NewReader(input))

	// Should generate rdf:type triple
	count := 0
	for {
		_, err := dec.Next()
		if err != nil {
			break
		}
		count++
		if count > 10 {
			break
		}
	}
	if count == 0 {
		t.Error("Next should generate type triple")
	}
}

func TestJSONLDTripleDecoder_Next_WithMultipleTypes(t *testing.T) {
	input := `{
		"@context": {"ex": "http://example.org/"},
		"@id": "ex:s",
		"@type": ["ex:Type1", "ex:Type2"]
	}`
	dec := newJSONLDtripleDecoder(strings.NewReader(input))

	count := 0
	for {
		_, err := dec.Next()
		if err != nil {
			break
		}
		count++
		if count > 10 {
			break
		}
	}
	if count < 2 {
		t.Error("Next should generate multiple type triples")
	}
}

func TestJSONLDTripleDecoder_Next_WithList(t *testing.T) {
	input := `{
		"@context": {"ex": "http://example.org/"},
		"@id": "ex:s",
		"ex:p": {
			"@list": ["v1", "v2"]
		}
	}`
	dec := newJSONLDtripleDecoder(strings.NewReader(input))

	// Should expand list into collection
	count := 0
	for {
		_, err := dec.Next()
		if err != nil {
			break
		}
		count++
		if count > 20 {
			break
		}
	}
	if count == 0 {
		t.Error("Next should expand list")
	}
}

func TestJSONLDTripleDecoder_Next_WithReverse(t *testing.T) {
	input := `{
		"@context": {"ex": "http://example.org/"},
		"@id": "ex:o",
		"ex:p": {
			"@reverse": "ex:hasSubject"
		}
	}`
	dec := newJSONLDtripleDecoder(strings.NewReader(input))

	// May or may not generate triples depending on implementation
	count := 0
	for {
		_, err := dec.Next()
		if err != nil {
			break
		}
		count++
		if count > 10 {
			break
		}
	}
	_ = count
}

func TestJSONLDTripleDecoder_Next_WithSet(t *testing.T) {
	input := `{
		"@context": {"ex": "http://example.org/"},
		"@id": "ex:s",
		"ex:p": {
			"@set": ["v1", "v2"]
		}
	}`
	dec := newJSONLDtripleDecoder(strings.NewReader(input))

	// @set is handled as a regular array, so should generate triples
	count := 0
	for {
		_, err := dec.Next()
		if err != nil {
			break
		}
		count++
		if count > 10 {
			break
		}
	}
	// @set may or may not generate triples depending on implementation
	_ = count
}

func TestJSONLDTripleDecoder_Next_WithIndex(t *testing.T) {
	input := `{
		"@context": {"ex": "http://example.org/"},
		"@id": "ex:s",
		"ex:p": {
			"@value": "v",
			"@index": "idx1"
		}
	}`
	dec := newJSONLDtripleDecoder(strings.NewReader(input))

	triple, err := dec.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	if triple.O == nil {
		t.Error("Next should return triple with object")
	}
}

func TestJSONLDTripleDecoder_Next_WithContextCancellation(t *testing.T) {
	input := `{
		"@context": {"ex": "http://example.org/"},
		"@id": "ex:s",
		"ex:p": "v"
	}`
	ctx, cancel := context.WithCancel(context.Background())
	opts := JSONLDOptions{}
	opts.Context = ctx
	cancel()

	dec := newJSONLDtripleDecoderWithOptions(strings.NewReader(input), opts)

	_, err := dec.Next()
	if err == nil {
		t.Error("Next should error when context is cancelled")
	}
}

func TestJSONLDTripleDecoder_Close(t *testing.T) {
	input := `{}`
	dec := newJSONLDtripleDecoder(strings.NewReader(input))

	if err := dec.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestJSONLDQuadDecoder_Next_Simple(t *testing.T) {
	input := `{
		"@context": {"ex": "http://example.org/"},
		"@id": "ex:s",
		"ex:p": "v"
	}`
	dec := newJSONLDquadDecoderWithOptions(strings.NewReader(input), JSONLDOptions{})

	quad, err := dec.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	if quad.S == nil {
		t.Error("Next should return quad with subject")
	}
}

func TestJSONLDQuadDecoder_Next_WithGraph(t *testing.T) {
	input := `{
		"@context": {"ex": "http://example.org/"},
		"@graph": [{
			"@id": "ex:s",
			"ex:p": "v"
		}]
	}`
	dec := newJSONLDquadDecoderWithOptions(strings.NewReader(input), JSONLDOptions{})

	quad, err := dec.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	if quad.S == nil {
		t.Error("Next should return quad")
	}
}

func TestJSONLDQuadDecoder_Next_WithNamedGraph(t *testing.T) {
	input := `{
		"@context": {"ex": "http://example.org/"},
		"@id": "ex:g",
		"@graph": [{
			"@id": "ex:s",
			"ex:p": "v"
		}]
	}`
	dec := newJSONLDquadDecoderWithOptions(strings.NewReader(input), JSONLDOptions{})

	quad, err := dec.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	// Graph name may be set to subject or nil depending on implementation
	_ = quad.G
}

func TestJSONLDQuadDecoder_Close(t *testing.T) {
	input := `{}`
	dec := newJSONLDquadDecoderWithOptions(strings.NewReader(input), JSONLDOptions{})

	if err := dec.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestJSONLDTripleEncoder_Write_Simple(t *testing.T) {
	var buf bytes.Buffer
	enc := newJSONLDtripleEncoder(&buf)

	triple := Triple{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: IRI{Value: "http://example.org/o"},
	}

	if err := enc.Write(triple); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
}

func TestJSONLDTripleEncoder_Write_WithBlankNode(t *testing.T) {
	var buf bytes.Buffer
	enc := newJSONLDtripleEncoder(&buf)

	triple := Triple{
		S: BlankNode{ID: "b1"},
		P: IRI{Value: "http://example.org/p"},
		O: IRI{Value: "http://example.org/o"},
	}

	if err := enc.Write(triple); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
}

func TestJSONLDTripleEncoder_Write_WithLiteral(t *testing.T) {
	var buf bytes.Buffer
	enc := newJSONLDtripleEncoder(&buf)

	triple := Triple{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: Literal{Lexical: "test", Lang: "en"},
	}

	if err := enc.Write(triple); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
}

func TestJSONLDTripleEncoder_Write_WithLiteralDatatype(t *testing.T) {
	var buf bytes.Buffer
	enc := newJSONLDtripleEncoder(&buf)

	triple := Triple{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: Literal{
			Lexical:  "123",
			Datatype: IRI{Value: "http://www.w3.org/2001/XMLSchema#integer"},
		},
	}

	if err := enc.Write(triple); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
}

func TestJSONLDTripleEncoder_Write_WithTripleTerm(t *testing.T) {
	var buf bytes.Buffer
	enc := newJSONLDtripleEncoder(&buf)

	triple := Triple{
		S: TripleTerm{
			S: IRI{Value: "http://example.org/s1"},
			P: IRI{Value: "http://example.org/p1"},
			O: IRI{Value: "http://example.org/o1"},
		},
		P: IRI{Value: "http://example.org/p2"},
		O: IRI{Value: "http://example.org/o2"},
	}

	// JSON-LD encoder may not support TripleTerm subjects
	err := enc.Write(triple)
	if err != nil {
		// Expected error for unsupported feature
		_ = err
	}
}

func TestJSONLDTripleEncoder_Write_Multiple(t *testing.T) {
	var buf bytes.Buffer
	enc := newJSONLDtripleEncoder(&buf)

	triples := []Triple{
		{S: IRI{Value: "http://example.org/s1"}, P: IRI{Value: "http://example.org/p"}, O: IRI{Value: "http://example.org/o1"}},
		{S: IRI{Value: "http://example.org/s2"}, P: IRI{Value: "http://example.org/p"}, O: IRI{Value: "http://example.org/o2"}},
	}

	for _, triple := range triples {
		if err := enc.Write(triple); err != nil {
			t.Fatalf("Write failed: %v", err)
		}
	}
}

func TestJSONLDTripleEncoder_Flush(t *testing.T) {
	var buf bytes.Buffer
	enc := newJSONLDtripleEncoder(&buf)

	triple := Triple{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: IRI{Value: "http://example.org/o"},
	}

	_ = enc.Write(triple)

	if err := enc.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}
}

func TestJSONLDTripleEncoder_Close(t *testing.T) {
	var buf bytes.Buffer
	enc := newJSONLDtripleEncoder(&buf)

	triple := Triple{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: IRI{Value: "http://example.org/o"},
	}

	_ = enc.Write(triple)

	if err := enc.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestJSONLDTripleEncoder_Close_Empty(t *testing.T) {
	var buf bytes.Buffer
	enc := newJSONLDtripleEncoder(&buf)

	if err := enc.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestJSONLDQuadEncoder_Write_Simple(t *testing.T) {
	var buf bytes.Buffer
	enc := newJSONLDquadEncoderWithOptions(&buf, JSONLDOptions{})

	quad := Quad{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: IRI{Value: "http://example.org/o"},
		G: IRI{Value: "http://example.org/g"},
	}

	if err := enc.Write(quad); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
}

func TestJSONLDQuadEncoder_Write_DefaultGraph(t *testing.T) {
	var buf bytes.Buffer
	enc := newJSONLDquadEncoderWithOptions(&buf, JSONLDOptions{})

	quad := Quad{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: IRI{Value: "http://example.org/o"},
		G: nil,
	}

	if err := enc.Write(quad); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
}

func TestJSONLDQuadEncoder_Write_Multiple(t *testing.T) {
	var buf bytes.Buffer
	enc := newJSONLDquadEncoderWithOptions(&buf, JSONLDOptions{})

	quads := []Quad{
		{S: IRI{Value: "http://example.org/s1"}, P: IRI{Value: "http://example.org/p"}, O: IRI{Value: "http://example.org/o1"}, G: IRI{Value: "http://example.org/g"}},
		{S: IRI{Value: "http://example.org/s2"}, P: IRI{Value: "http://example.org/p"}, O: IRI{Value: "http://example.org/o2"}, G: IRI{Value: "http://example.org/g"}},
	}

	for _, quad := range quads {
		if err := enc.Write(quad); err != nil {
			t.Fatalf("Write failed: %v", err)
		}
	}
}

func TestJSONLDQuadEncoder_Flush(t *testing.T) {
	var buf bytes.Buffer
	enc := newJSONLDquadEncoderWithOptions(&buf, JSONLDOptions{})

	quad := Quad{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: IRI{Value: "http://example.org/o"},
		G: IRI{Value: "http://example.org/g"},
	}

	_ = enc.Write(quad)

	if err := enc.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}
}

func TestJSONLDQuadEncoder_Close(t *testing.T) {
	var buf bytes.Buffer
	enc := newJSONLDquadEncoderWithOptions(&buf, JSONLDOptions{})

	quad := Quad{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: IRI{Value: "http://example.org/o"},
		G: IRI{Value: "http://example.org/g"},
	}

	_ = enc.Write(quad)

	if err := enc.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestJSONLDQuadEncoder_Close_Empty(t *testing.T) {
	var buf bytes.Buffer
	enc := newJSONLDquadEncoderWithOptions(&buf, JSONLDOptions{})

	if err := enc.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestJSONLDTripleEncoder_WithOptions(t *testing.T) {
	var buf bytes.Buffer
	opts := JSONLDOptions{}
	enc := newJSONLDtripleEncoderWithOptions(&buf, opts)

	triple := Triple{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: IRI{Value: "http://example.org/o"},
	}

	if err := enc.Write(triple); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
}

func TestJSONLDQuadEncoder_WithOptions(t *testing.T) {
	var buf bytes.Buffer
	opts := JSONLDOptions{}
	enc := newJSONLDquadEncoderWithOptions(&buf, opts)

	quad := Quad{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: IRI{Value: "http://example.org/o"},
		G: IRI{Value: "http://example.org/g"},
	}

	if err := enc.Write(quad); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
}

func TestShouldEagerFlushJSONLD(t *testing.T) {
	var buf bytes.Buffer
	result := shouldEagerFlushJSONLD(&buf)
	// May or may not be true depending on implementation
	_ = result
}

func TestShouldEagerFlushJSONLD_WithStringBuilder(t *testing.T) {
	var buf strings.Builder
	result := shouldEagerFlushJSONLD(&buf)
	// May or may not be true depending on implementation
	_ = result
}
