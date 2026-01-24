package rdf

import (
	"strings"
	"testing"
)

func TestValidateIRI(t *testing.T) {
	tests := []struct {
		name    string
		iri     string
		wantErr bool
	}{
		// Valid IRIs
		{
			name:    "valid absolute IRI with http scheme",
			iri:     "http://example.org/resource",
			wantErr: false,
		},
		{
			name:    "valid absolute IRI with https scheme",
			iri:     "https://example.org/resource",
			wantErr: false,
		},
		{
			name:    "valid absolute IRI with custom scheme",
			iri:     "urn:example:resource",
			wantErr: false,
		},
		{
			name:    "valid IRI with path",
			iri:     "http://example.org/path/to/resource",
			wantErr: false,
		},
		{
			name:    "valid IRI with query",
			iri:     "http://example.org/resource?param=value",
			wantErr: false,
		},
		{
			name:    "valid IRI with fragment",
			iri:     "http://example.org/resource#fragment",
			wantErr: false,
		},
		{
			name:    "valid relative IRI",
			iri:     "/path/to/resource",
			wantErr: false,
		},
		{
			name:    "valid relative IRI with dot",
			iri:     "./path/to/resource",
			wantErr: false,
		},
		{
			name:    "valid relative IRI with dot dot",
			iri:     "../path/to/resource",
			wantErr: false,
		},

		// Invalid IRIs
		{
			name:    "empty IRI",
			iri:     "",
			wantErr: true,
		},
		{
			name:    "relative IRI without scheme (network-path)",
			iri:     "//example.org/resource",
			wantErr: true,
		},
		{
			name:    "IRI with invalid control character",
			iri:     "http://example.org/resource\x00",
			wantErr: true,
		},
		{
			name:    "IRI with invalid character <",
			iri:     "http://example.org/resource<invalid",
			wantErr: true,
		},
		{
			name:    "IRI with invalid character >",
			iri:     "http://example.org/resource>invalid",
			wantErr: true,
		},
		{
			name:    "IRI with scheme starting with number",
			iri:     "123scheme://example.org/resource",
			wantErr: true,
		},
		// Note: "example:org:resource" is actually valid - url.Parse treats "example" as a scheme
		// This is technically valid per RFC 3987, so we accept it
		// For stricter validation, applications can add custom checks
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIRI(tt.iri)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateIRI(%q) error = %v, wantErr %v", tt.iri, err, tt.wantErr)
			}
		})
	}
}

func TestOptStrictIRIValidation(t *testing.T) {
	// Test that strict validation rejects invalid IRIs
	// Use an IRI with invalid characters that will fail validation
	invalidIRIInput := `<http://example.org/resource<invalid> <http://example.org/p> <http://example.org/o> .`

	// With strict validation, should fail
	dec, err := NewReader(strings.NewReader(invalidIRIInput), FormatTurtle, OptStrictIRIValidation())
	if err != nil {
		t.Fatalf("unexpected error creating reader: %v", err)
	}
	_, err = dec.Next()
	if err == nil {
		t.Error("expected error with strict IRI validation enabled for invalid IRI, got nil")
	} else {
		// Verify the error mentions IRI validation
		if err.Error() != "" && !strings.Contains(err.Error(), "IRI") && !strings.Contains(err.Error(), "invalid") {
			t.Logf("Warning: error message doesn't mention IRI validation: %v", err)
		}
	}
	dec.Close()

	// Test that strict validation accepts valid IRIs
	validIRIInput := `<http://example.org/s> <http://example.org/p> <http://example.org/o> .`
	dec, err = NewReader(strings.NewReader(validIRIInput), FormatTurtle, OptStrictIRIValidation())
	if err != nil {
		t.Fatalf("unexpected error creating reader: %v", err)
	}
	stmt, err := dec.Next()
	if err != nil {
		t.Errorf("unexpected error parsing valid IRI with strict validation: %v", err)
	}
	if stmt.S.String() != "http://example.org/s" {
		t.Errorf("unexpected subject: got %s, want http://example.org/s", stmt.S.String())
	}
	dec.Close()
}

func TestValidateIRI_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		iri     string
		wantErr bool
	}{
		{
			name:    "IRI with port",
			iri:     "http://example.org:8080/resource",
			wantErr: false,
		},
		{
			name:    "IRI with user info",
			iri:     "http://user:pass@example.org/resource",
			wantErr: false,
		},
		{
			name:    "IRI with percent encoding",
			iri:     "http://example.org/resource%20with%20spaces",
			wantErr: false,
		},
		{
			name:    "file scheme IRI",
			iri:     "file:///path/to/file",
			wantErr: false,
		},
		{
			name:    "data URI",
			iri:     "data:text/plain;base64,SGVsbG8=",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIRI(tt.iri)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateIRI(%q) error = %v, wantErr %v", tt.iri, err, tt.wantErr)
			}
		})
	}
}
