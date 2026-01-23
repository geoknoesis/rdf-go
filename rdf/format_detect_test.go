package rdf

import (
	"strings"
	"testing"
)

func TestDetectFormatTurtle(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Format
		wantOK   bool
	}{
		{
			name:     "Turtle with prefix",
			input:    "@prefix ex: <http://example.org/> .\nex:s ex:p ex:o .",
			expected: FormatTurtle,
			wantOK:   true,
		},
		{
			name:     "Turtle with base",
			input:    "@base <http://example.org/> .\n<s> <p> <o> .",
			expected: FormatTurtle,
			wantOK:   true,
		},
		{
			name:     "Turtle with prefixes",
			input:    "PREFIX ex: <http://example.org/>\n<s> <p> <o> .",
			expected: FormatTurtle,
			wantOK:   true,
		},
		{
			name:     "Turtle with blank node",
			input:    "[] <p> <o> .",
			expected: FormatTurtle,
			wantOK:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format, ok := DetectFormat(strings.NewReader(tt.input))
			if ok != tt.wantOK {
				t.Errorf("DetectFormat() ok = %v, want %v", ok, tt.wantOK)
			}
			if format != tt.expected {
				t.Errorf("DetectFormat() format = %v, want %v", format, tt.expected)
			}
		})
	}
}

func TestDetectFormatNTriples(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Format
		wantOK   bool
	}{
		{
			name:     "N-Triples basic",
			input:    "<http://example.org/s> <http://example.org/p> <http://example.org/o> .",
			expected: FormatNTriples,
			wantOK:   true,
		},
		{
			name:     "N-Triples with blank node",
			input:    "<http://example.org/s> <http://example.org/p> _:b0 .",
			expected: FormatNTriples,
			wantOK:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format, ok := DetectFormat(strings.NewReader(tt.input))
			if ok != tt.wantOK {
				t.Errorf("DetectFormat() ok = %v, want %v", ok, tt.wantOK)
			}
			if format != tt.expected {
				t.Errorf("DetectFormat() format = %v, want %v", format, tt.expected)
			}
		})
	}
}

func TestDetectFormatJSONLD(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Format
		wantOK   bool
	}{
		{
			name:     "JSON-LD object",
			input:    `{"@context": {"ex": "http://example.org/"}, "@id": "ex:s", "ex:p": "o"}`,
			expected: FormatJSONLD,
			wantOK:   true,
		},
		{
			name:     "JSON-LD array",
			input:    `[{"@id": "ex:s", "ex:p": "o"}]`,
			expected: FormatJSONLD,
			wantOK:   true,
		},
		{
			name:     "JSON-LD with @type",
			input:    `{"@type": "Person", "name": "John"}`,
			expected: FormatJSONLD,
			wantOK:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format, ok := DetectFormat(strings.NewReader(tt.input))
			if ok != tt.wantOK {
				t.Errorf("DetectFormat() ok = %v, want %v", ok, tt.wantOK)
			}
			if format != tt.expected {
				t.Errorf("DetectFormat() format = %v, want %v", format, tt.expected)
			}
		})
	}
}

func TestDetectFormatRDFXML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Format
		wantOK   bool
	}{
		{
			name:     "RDF/XML with XML declaration",
			input:    `<?xml version="1.0"?><rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">`,
			expected: FormatRDFXML,
			wantOK:   true,
		},
		{
			name:     "RDF/XML with rdf: prefix",
			input:    `<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">`,
			expected: FormatRDFXML,
			wantOK:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format, ok := DetectFormat(strings.NewReader(tt.input))
			if ok != tt.wantOK {
				t.Errorf("DetectFormat() ok = %v, want %v", ok, tt.wantOK)
			}
			if format != tt.expected {
				t.Errorf("DetectFormat() format = %v, want %v", format, tt.expected)
			}
		})
	}
}

func TestDetectFormatTriG(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Format
		wantOK   bool
	}{
		{
			name:     "TriG with GRAPH",
			input:    "GRAPH <http://example.org/g> { <s> <p> <o> . }",
			expected: FormatTriG,
			wantOK:   true,
		},
		{
			name:     "TriG with graph block",
			input:    "<http://example.org/g> { <s> <p> <o> . }",
			expected: FormatTriG,
			wantOK:   true,
		},
		{
			name:     "TriG with prefix and graph",
			input:    "@prefix ex: <http://example.org/> .\nGRAPH ex:g { ex:s ex:p ex:o . }",
			expected: FormatTriG,
			wantOK:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TriG is a quad format, use DetectQuadFormat
			format, ok := DetectQuadFormat(strings.NewReader(tt.input))
			if ok != tt.wantOK {
				t.Errorf("DetectQuadFormat() ok = %v, want %v", ok, tt.wantOK)
			}
			if format != tt.expected {
				t.Errorf("DetectQuadFormat() format = %v, want %v", format, tt.expected)
			}
		})
	}
}

func TestDetectFormatNQuads(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Format
		wantOK   bool
	}{
		{
			name:     "N-Quads basic",
			input:    "<http://example.org/s> <http://example.org/p> <http://example.org/o> <http://example.org/g> .",
			expected: FormatNQuads,
			wantOK:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// N-Quads is a quad format, use DetectQuadFormat
			format, ok := DetectQuadFormat(strings.NewReader(tt.input))
			if ok != tt.wantOK {
				t.Errorf("DetectQuadFormat() ok = %v, want %v", ok, tt.wantOK)
			}
			if format != tt.expected {
				t.Errorf("DetectQuadFormat() format = %v, want %v", format, tt.expected)
			}
		})
	}
}

func TestDetectFormatAuto(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantOK   bool
		expected string
	}{
		{
			name:     "TriG format",
			input:    "GRAPH <g> { <s> <p> <o> . }",
			wantOK:   true,
			expected: "trig",
		},
		{
			name:     "Turtle format",
			input:    "@prefix ex: <http://example.org/> .\nex:s ex:p ex:o .",
			wantOK:   true,
			expected: "turtle",
		},
		{
			name:     "N-Quads format",
			input:    "<s> <p> <o> <g> .",
			wantOK:   true,
			expected: "nquads",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format, ok := DetectFormatAuto(strings.NewReader(tt.input))
			if ok != tt.wantOK {
				t.Errorf("DetectFormatAuto() ok = %v, want %v", ok, tt.wantOK)
			}
			if format != tt.expected {
				t.Errorf("DetectFormatAuto() format = %v, want %v", format, tt.expected)
			}
		})
	}
}

