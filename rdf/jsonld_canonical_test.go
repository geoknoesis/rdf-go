package rdf

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestCanonicalizeJSONLD(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		validate func(t *testing.T, output []byte)
	}{
		{
			name:    "simple JSON-LD",
			input:   `{"@context":{"ex":"http://example.org/"},"@id":"ex:s","ex:p":"v"}`,
			wantErr: false,
			validate: func(t *testing.T, output []byte) {
				// Should be valid JSON
				var data interface{}
				if err := json.Unmarshal(output, &data); err != nil {
					t.Errorf("canonicalized output is not valid JSON: %v", err)
				}
			},
		},
		{
			name:    "JSON-LD with graph",
			input:   `{"@context":{"ex":"http://example.org/"},"@graph":[{"@id":"ex:s","ex:p":"v"}]}`,
			wantErr: false,
			validate: func(t *testing.T, output []byte) {
				var data interface{}
				if err := json.Unmarshal(output, &data); err != nil {
					t.Errorf("canonicalized output is not valid JSON: %v", err)
				}
			},
		},
		{
			name:    "invalid JSON",
			input:   `{invalid json}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := CanonicalizeJSONLD([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("CanonicalizeJSONLD() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, output)
			}
		})
	}
}

func TestCanonicalizeJSONLDDeterministic(t *testing.T) {
	// Test that canonicalization produces deterministic output
	input := `{"@context":{"ex":"http://example.org/"},"@id":"ex:s","ex:p":"v","ex:q":"w"}`

	output1, err := CanonicalizeJSONLD([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output2, err := CanonicalizeJSONLD([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Output should be identical
	if !bytes.Equal(output1, output2) {
		t.Errorf("canonicalization is not deterministic:\n%q\nvs\n%q", output1, output2)
	}
}

func TestCanonicalizeJSONLDReader(t *testing.T) {
	input := `{"@context":{"ex":"http://example.org/"},"@id":"ex:s","ex:p":"v"}`
	reader := strings.NewReader(input)

	output, err := CanonicalizeJSONLDReader(reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be valid JSON
	var data interface{}
	if err := json.Unmarshal(output, &data); err != nil {
		t.Errorf("canonicalized output is not valid JSON: %v", err)
	}
}

func TestCanonicalizeJSONLDWriter(t *testing.T) {
	input := `{"@context":{"ex":"http://example.org/"},"@id":"ex:s","ex:p":"v"}`
	reader := strings.NewReader(input)
	var buf bytes.Buffer

	err := CanonicalizeJSONLDWriter(&buf, reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be valid JSON
	var data interface{}
	if err := json.Unmarshal(buf.Bytes(), &data); err != nil {
		t.Errorf("canonicalized output is not valid JSON: %v", err)
	}
}
