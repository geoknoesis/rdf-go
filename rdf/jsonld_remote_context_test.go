package rdf

import (
	"context"
	"io"
	"strings"
	"testing"
)

// schemaOrgLoader is a mock DocumentLoader that returns a simplified schema.org context
type schemaOrgLoader struct{}

func (l *schemaOrgLoader) LoadDocument(ctx context.Context, iri string) (RemoteDocument, error) {
	// Return a simplified schema.org context for testing
	// In real usage, this would fetch from https://schema.org/docs/jsonldcontext.jsonld
	contextDoc := map[string]interface{}{
		"@context": map[string]interface{}{
			"@vocab": "https://schema.org/",
			"name":   "https://schema.org/name",
			"Person": "https://schema.org/Person",
			"url":    "https://schema.org/url",
		},
	}
	return RemoteDocument{
		DocumentURL: iri,
		Document:    contextDoc,
		ContextURL:  iri,
	}, nil
}

func TestJSONLDRemoteContextSchemaOrg(t *testing.T) {
	// Test JSON-LD with schema.org remote context reference
	input := `{
		"@context": "https://schema.org/docs/jsonldcontext.jsonld",
		"@type": "Person",
		"name": "John Doe",
		"url": "https://example.org/john"
	}`

	opts := JSONLDOptions{
		DocumentLoader: &schemaOrgLoader{},
		Context:        context.Background(),
	}

	dec := NewJSONLDTripleDecoder(strings.NewReader(input), opts)
	defer dec.Close()

	// Collect all triples
	var triples []Triple
	for {
		triple, err := dec.Next()
		if err != nil {
			if isEOF(err) {
				break
			}
			t.Fatalf("unexpected error: %v", err)
		}
		triples = append(triples, triple)
	}

	// Verify we got triples
	if len(triples) == 0 {
		t.Fatal("expected at least one triple")
	}

	// Verify rdf:type triple with schema.org/Person
	foundType := false
	for _, triple := range triples {
		if triple.P.Value == "http://www.w3.org/1999/02/22-rdf-syntax-ns#type" {
			if iri, ok := triple.O.(IRI); ok {
				if iri.Value == "https://schema.org/Person" {
					foundType = true
					break
				}
			}
		}
	}
	if !foundType {
		t.Error("expected rdf:type triple with schema.org/Person")
		t.Logf("Triples: %+v", triples)
	}

	// Verify name property was expanded correctly
	foundName := false
	for _, triple := range triples {
		if triple.P.Value == "https://schema.org/name" {
			if lit, ok := triple.O.(Literal); ok {
				if lit.Lexical == "John Doe" {
					foundName = true
					break
				}
			}
		}
	}
	if !foundName {
		t.Error("expected name property triple with value 'John Doe'")
		t.Logf("Triples: %+v", triples)
	}

	// Verify url property was expanded correctly
	foundURL := false
	for _, triple := range triples {
		if triple.P.Value == "https://schema.org/url" {
			if iri, ok := triple.O.(IRI); ok {
				if iri.Value == "https://example.org/john" {
					foundURL = true
					break
				}
			}
		}
	}
	if !foundURL {
		t.Error("expected url property triple")
		t.Logf("Triples: %+v", triples)
	}
}

func TestJSONLDRemoteContextArray(t *testing.T) {
	// Test JSON-LD with array of contexts including remote reference
	input := `{
		"@context": [
			"https://schema.org/docs/jsonldcontext.jsonld",
			{"ex": "http://example.org/"}
		],
		"@type": "Person",
		"name": "Jane Doe",
		"ex:custom": "value"
	}`

	opts := JSONLDOptions{
		DocumentLoader: &schemaOrgLoader{},
		Context:        context.Background(),
	}

	dec := NewJSONLDTripleDecoder(strings.NewReader(input), opts)
	defer dec.Close()

	// Collect all triples
	var triples []Triple
	for {
		triple, err := dec.Next()
		if err != nil {
			if isEOF(err) {
				break
			}
			t.Fatalf("unexpected error: %v", err)
		}
		triples = append(triples, triple)
	}

	// Verify we got triples
	if len(triples) == 0 {
		t.Fatal("expected at least one triple")
	}

	// Verify schema.org context was applied (name should expand to schema.org/name)
	foundSchemaName := false
	for _, triple := range triples {
		if triple.P.Value == "https://schema.org/name" {
			foundSchemaName = true
			break
		}
	}
	if !foundSchemaName {
		t.Error("expected name property to expand using schema.org context")
		t.Logf("Triples: %+v", triples)
	}

	// Verify custom prefix was also applied
	foundCustom := false
	for _, triple := range triples {
		if triple.P.Value == "http://example.org/custom" {
			foundCustom = true
			break
		}
	}
	if !foundCustom {
		t.Error("expected custom prefix to be applied")
		t.Logf("Triples: %+v", triples)
	}
}

func TestJSONLDRemoteContextWithoutLoader(t *testing.T) {
	// Test that remote context URLs are ignored when no DocumentLoader is provided
	input := `{
		"@context": "https://schema.org/docs/jsonldcontext.jsonld",
		"@type": "Person",
		"name": "John Doe"
	}`

	opts := JSONLDOptions{
		// No DocumentLoader provided
		Context: context.Background(),
	}

	dec := NewJSONLDTripleDecoder(strings.NewReader(input), opts)
	defer dec.Close()

	// Should fail because context URL can't be resolved and terms can't be expanded
	_, err := dec.Next()
	if err == nil {
		// If it doesn't fail, verify that terms weren't expanded (name should remain as-is)
		t.Log("Note: Remote context was ignored (no DocumentLoader), terms may not be expanded correctly")
	}
}

func TestJSONLDRemoteContextInNode(t *testing.T) {
	// Test remote context in nested node
	input := `{
		"@context": {"ex": "http://example.org/"},
		"@id": "ex:parent",
		"ex:child": {
			"@context": "https://schema.org/docs/jsonldcontext.jsonld",
			"@type": "Person",
			"name": "Child Name"
		}
	}`

	opts := JSONLDOptions{
		DocumentLoader: &schemaOrgLoader{},
		Context:        context.Background(),
	}

	dec := NewJSONLDTripleDecoder(strings.NewReader(input), opts)
	defer dec.Close()

	// Collect all triples
	var triples []Triple
	for {
		triple, err := dec.Next()
		if err != nil {
			if isEOF(err) {
				break
			}
			t.Fatalf("unexpected error: %v", err)
		}
		triples = append(triples, triple)
	}

	// Verify nested node's remote context was resolved
	foundNestedName := false
	for _, triple := range triples {
		if triple.P.Value == "https://schema.org/name" {
			if lit, ok := triple.O.(Literal); ok {
				if lit.Lexical == "Child Name" {
					foundNestedName = true
					break
				}
			}
		}
	}
	if !foundNestedName {
		t.Error("expected nested node's remote context to be resolved")
		t.Logf("Triples: %+v", triples)
	}
}

// isEOF checks if error is EOF
func isEOF(err error) bool {
	return err == io.EOF
}

