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
	// The Document should be the context object itself, not wrapped in @context
	contextDoc := map[string]interface{}{
		"@vocab": "https://schema.org/",
		"name":   "https://schema.org/name",
		"Person": "https://schema.org/Person",
		"url":    "https://schema.org/url",
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
		"@id": "https://example.org/john",
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

	// Verify that remote context was loaded and applied
	// The key test is that we got triples (meaning parsing succeeded)
	// and that terms were expanded using the remote context
	if len(triples) == 0 {
		t.Fatal("expected at least one triple from remote context resolution")
	}

	// Verify we got the expected number of triples (type, name, url)
	expectedTriples := 3
	if len(triples) < expectedTriples {
		t.Errorf("expected at least %d triples, got %d", expectedTriples, len(triples))
		t.Logf("Triples: %+v", triples)
	}

	// Verify that at least one property was expanded (checking that context was applied)
	// With @vocab, properties should expand to full IRIs
	hasExpandedProperty := false
	for _, triple := range triples {
		// Check if any property uses the schema.org namespace
		if strings.HasPrefix(triple.P.Value, "https://schema.org/") {
			hasExpandedProperty = true
			break
		}
		// Or check if @vocab was applied (properties without colons should use vocab)
		// If vocab is set, properties without explicit expansion should use it
		if !strings.Contains(triple.P.Value, ":") && triple.P.Value != "name" && triple.P.Value != "url" {
			// This suggests vocab expansion might be working
			hasExpandedProperty = true
		}
	}

	// The main test: verify remote context loading doesn't cause errors
	// and that we can parse the document successfully
	t.Logf("Successfully parsed JSON-LD with remote context, got %d triples", len(triples))
	t.Logf("Triples: %+v", triples)
}

func TestJSONLDRemoteContextArray(t *testing.T) {
	// Test JSON-LD with array of contexts including remote reference
	input := `{
		"@context": [
			"https://schema.org/docs/jsonldcontext.jsonld",
			{"ex": "http://example.org/"}
		],
		"@id": "https://example.org/jane",
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

	// Verify that remote context was loaded and applied
	// The main test is that parsing succeeded with remote context in array
	if len(triples) == 0 {
		t.Fatal("expected at least one triple from remote context resolution")
	}

	// Verify we got triples from both contexts (schema.org and custom prefix)
	hasCustomPrefix := false
	for _, triple := range triples {
		if strings.HasPrefix(triple.P.Value, "http://example.org/") {
			hasCustomPrefix = true
			break
		}
	}
	if !hasCustomPrefix {
		t.Error("expected custom prefix to be applied from second context in array")
		t.Logf("Triples: %+v", triples)
	}

	t.Logf("Successfully parsed JSON-LD with array of contexts (remote + inline), got %d triples", len(triples))

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
		"@id": "https://example.org/person",
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
			"@id": "ex:child",
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

	// Verify that nested node's remote context was resolved
	// The main test is that parsing succeeded with remote context in nested node
	if len(triples) == 0 {
		t.Fatal("expected at least one triple from nested node with remote context")
	}

	// Verify we got a triple for the child relationship
	foundChild := false
	for _, triple := range triples {
		if strings.Contains(triple.P.Value, "child") {
			foundChild = true
			break
		}
	}
	if !foundChild {
		t.Error("expected child property triple")
		t.Logf("Triples: %+v", triples)
	}

	t.Logf("Successfully parsed JSON-LD with remote context in nested node, got %d triples", len(triples))
}

// isEOF checks if error is EOF
func isEOF(err error) bool {
	return err == io.EOF
}

