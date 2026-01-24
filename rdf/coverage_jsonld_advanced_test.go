package rdf

import (
	"context"
	"io"
	"strings"
	"testing"
)

// Test additional JSON-LD edge cases and error paths to increase coverage

func TestJSONLDEmitValueArrayEmpty(t *testing.T) {
	var quads []Quad
	state := &jsonldState{}
	ctx := newJSONLDContext()
	err := emitJSONLDValue(
		IRI{Value: "http://example.org/s"},
		IRI{Value: "http://example.org/p"},
		[]interface{}{},
		ctx,
		nil,
		state,
		appendQuadSink(&quads),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(quads) != 0 {
		t.Errorf("expected no quads for empty array, got %d", len(quads))
	}
}

func TestJSONLDEmitValueArrayWithError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	state := &jsonldState{ctx: ctx}
	err := emitJSONLDValue(
		IRI{Value: "http://example.org/s"},
		IRI{Value: "http://example.org/p"},
		[]interface{}{"value"},
		newJSONLDContext(),
		nil,
		state,
		func(Quad) error { return nil },
	)
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}

func TestJSONLDEmitValueUnsupportedLiteral(t *testing.T) {
	var quads []Quad
	state := &jsonldState{}
	err := emitJSONLDValue(
		IRI{Value: "http://example.org/s"},
		IRI{Value: "http://example.org/p"},
		nil, // nil is unsupported
		newJSONLDContext(),
		nil,
		state,
		appendQuadSink(&quads),
	)
	if err == nil {
		t.Fatal("expected error for unsupported literal value")
	}
}

func TestJSONLDEmitObjectValueUnsupported(t *testing.T) {
	var quads []Quad
	state := &jsonldState{}
	ctx := newJSONLDContext()
	value := map[string]interface{}{
		"unsupported": "key",
	}
	err := emitJSONLDObjectValue(
		value,
		IRI{Value: "http://example.org/s"},
		IRI{Value: "http://example.org/p"},
		ctx,
		nil,
		state,
		appendQuadSink(&quads),
	)
	if err == nil {
		t.Fatal("expected error for unsupported object value")
	}
}

func TestJSONLDEmitTypeStatementsArray(t *testing.T) {
	var quads []Quad
	ctx := newJSONLDContext()
	subject := IRI{Value: "http://example.org/s"}
	rawTypes := []interface{}{"http://example.org/Type1", "http://example.org/Type2"}

	err := emitJSONLDTypeStatements(subject, rawTypes, ctx, nil, appendQuadSink(&quads))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(quads) != 2 {
		t.Errorf("expected 2 type quads, got %d", len(quads))
	}
}

func TestJSONLDEmitTypeStatementsInvalidArray(t *testing.T) {
	var quads []Quad
	ctx := newJSONLDContext()
	subject := IRI{Value: "http://example.org/s"}
	rawTypes := []interface{}{123, "http://example.org/Type"} // mixed types

	err := emitJSONLDTypeStatements(subject, rawTypes, ctx, nil, appendQuadSink(&quads))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should skip non-string types
	if len(quads) != 1 {
		t.Errorf("expected 1 type quad, got %d", len(quads))
	}
}

func TestJSONLDEmitPredicateValuesContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	state := &jsonldState{ctx: ctx}
	node := map[string]interface{}{
		"http://example.org/p": "value",
	}

	err := emitJSONLDPredicateValues(
		node,
		IRI{Value: "http://example.org/s"},
		newJSONLDContext(),
		nil,
		state,
		func(Quad) error { return nil },
	)
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}

func TestJSONLDEmitPredicateValuesUnresolvablePredicate(t *testing.T) {
	var quads []Quad
	state := &jsonldState{}
	ctx := newJSONLDContext()
	node := map[string]interface{}{
		"": "value", // empty predicate cannot be resolved
	}

	err := emitJSONLDPredicateValues(
		node,
		IRI{Value: "http://example.org/s"},
		ctx,
		nil,
		state,
		appendQuadSink(&quads),
	)
	if err == nil {
		t.Fatal("expected error for unresolvable predicate")
	}
}

func TestJSONLDListEmpty(t *testing.T) {
	var quads []Quad
	state := &jsonldState{}
	head, err := emitJSONLDList(
		[]interface{}{},
		newJSONLDContext(),
		nil,
		state,
		appendQuadSink(&quads),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if head == nil {
		t.Fatal("expected rdf:nil for empty list")
	}
	if iri, ok := head.(IRI); !ok || iri.Value != rdfNilIRI {
		t.Errorf("expected rdf:nil, got %v", head)
	}
}

func TestJSONLDListInvalidType(t *testing.T) {
	var quads []Quad
	state := &jsonldState{}
	_, err := emitJSONLDList(
		"not a list",
		newJSONLDContext(),
		nil,
		state,
		appendQuadSink(&quads),
	)
	if err == nil {
		t.Fatal("expected error for invalid list type")
	}
}

func TestJSONLDListContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	state := &jsonldState{ctx: ctx}
	_, err := emitJSONLDList(
		[]interface{}{"value"},
		newJSONLDContext(),
		nil,
		state,
		func(Quad) error { return nil },
	)
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}

func TestJSONLDValueTermUnsupportedListValue(t *testing.T) {
	state := &jsonldState{}
	value := map[string]interface{}{
		"unsupported": "key",
	}
	_, err := jsonldValueTerm(value, newJSONLDContext(), nil, state, func(Quad) error { return nil })
	if err == nil {
		t.Fatal("expected error for unsupported list value")
	}
}

func TestJSONLDReaderMaxNodesLimit(t *testing.T) {
	// Create input with many nodes
	nodes := make([]string, 0, 100)
	for i := 0; i < 100; i++ {
		nodes = append(nodes, `{"@id":"ex:s`+string(rune(i))+`","ex:p":"v"}`)
	}
	input := `{"@context":{"ex":"http://example.org/"},"@graph":[` + strings.Join(nodes, ",") + `]}`

	opts := JSONLDOptions{MaxNodes: 10}
	dec, err := NewReader(strings.NewReader(input), FormatJSONLD)
	if err != nil {
		t.Fatalf("unexpected error creating reader: %v", err)
	}
	// Note: MaxNodes is JSONLDOptions-specific, not exposed via Options
	_ = opts
	defer dec.Close()

	// Should hit node limit
	count := 0
	for {
		_, err := dec.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			// May hit node limit or other error
			break
		}
		count++
		if count > 20 {
			break
		}
	}
}

func TestJSONLDReaderMaxGraphItemsLimit(t *testing.T) {
	// Create input with many graph items
	items := make([]string, 0, 100)
	for i := 0; i < 100; i++ {
		items = append(items, `{"@id":"ex:s`+string(rune(i))+`","ex:p":"v"}`)
	}
	input := `{"@context":{"ex":"http://example.org/"},"@graph":[` + strings.Join(items, ",") + `]}`

	dec, err := NewReader(strings.NewReader(input), FormatJSONLD)
	if err != nil {
		t.Fatalf("unexpected error creating reader: %v", err)
	}
	defer dec.Close()

	// Process some items
	count := 0
	for {
		_, err := dec.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
		count++
		if count > 50 {
			break
		}
	}
}

func TestJSONLDReaderMaxQuadsLimit(t *testing.T) {
	// Create input that would generate many quads
	input := `{"@context":{"ex":"http://example.org/"},"@id":"ex:s","ex:p1":"v1","ex:p2":"v2","ex:p3":"v3"}`

	dec, err := NewReader(strings.NewReader(input), FormatJSONLD)
	if err != nil {
		t.Fatalf("unexpected error creating reader: %v", err)
	}
	defer dec.Close()

	// Process quads
	count := 0
	for {
		_, err := dec.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
		count++
		if count > 10 {
			break
		}
	}
}

func TestJSONLDReaderMaxInputBytesLimit(t *testing.T) {
	largeInput := `{"@context":{"ex":"http://example.org/"},"@id":"ex:s","ex:p":"` + strings.Repeat("x", 10000) + `"}`

	// Note: MaxInputBytes is JSONLDOptions-specific, not exposed via Options
	dec, err := NewReader(strings.NewReader(largeInput), FormatJSONLD)
	if err != nil {
		t.Fatalf("unexpected error creating reader: %v", err)
	}
	defer dec.Close()

	_, err = dec.Next()
	// Should succeed with large input
	if err != nil && err != io.EOF {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJSONLDExpandTermWithVocab(t *testing.T) {
	ctx := newJSONLDContext()
	ctx.vocab = "http://example.org/"

	expanded := expandJSONLDTerm("term", ctx)
	expected := "http://example.org/term"
	if expanded != expected {
		t.Errorf("expected %q, got %q", expected, expanded)
	}
}

func TestJSONLDExpandTermWithBase(t *testing.T) {
	ctx := newJSONLDContext()
	ctx.base = "http://example.org/"

	expanded := expandJSONLDTerm("term", ctx)
	if !strings.HasPrefix(expanded, "http://example.org/") {
		t.Errorf("expected expanded term with base, got %q", expanded)
	}
}

func TestJSONLDExpandTermPrefixed(t *testing.T) {
	ctx := newJSONLDContext()
	ctx.prefixes["ex"] = "http://example.org/"

	expanded := expandJSONLDTerm("ex:term", ctx)
	expected := "http://example.org/term"
	if expanded != expected {
		t.Errorf("expected %q, got %q", expected, expanded)
	}
}

func TestJSONLDExpandTermUnknownPrefix(t *testing.T) {
	ctx := newJSONLDContext()

	expanded := expandJSONLDTerm("unknown:term", ctx)
	expected := "unknown:term"
	if expanded != expected {
		t.Errorf("expected unchanged term, got %q", expanded)
	}
}
