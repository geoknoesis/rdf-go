package rdf

import "strings"

// Format represents an RDF serialization format.
// Use FormatAuto (empty string) to enable automatic format detection.
type Format string

const (
	// FormatAuto enables automatic format detection from input.
	FormatAuto Format = ""
	
	// Triple formats
	FormatTurtle   Format = "turtle"
	FormatNTriples Format = "ntriples"
	FormatRDFXML   Format = "rdfxml"
	FormatJSONLD   Format = "jsonld"
	
	// Quad formats
	FormatTriG   Format = "trig"
	FormatNQuads Format = "nquads"
)

// ParseFormat normalizes a format string and returns a Format.
// Supports common aliases (e.g., "ttl" -> FormatTurtle, "nt" -> FormatNTriples).
func ParseFormat(s string) (Format, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "", "auto":
		return FormatAuto, true
	case "turtle", "ttl":
		return FormatTurtle, true
	case "ntriples", "nt":
		return FormatNTriples, true
	case "rdfxml", "rdf", "xml":
		return FormatRDFXML, true
	case "jsonld", "json-ld", "json":
		return FormatJSONLD, true
	case "trig":
		return FormatTriG, true
	case "nquads", "nq":
		return FormatNQuads, true
	default:
		return "", false
	}
}

// IsQuadFormat reports whether the format supports quads (named graphs).
func (f Format) IsQuadFormat() bool {
	return f == FormatTriG || f == FormatNQuads
}

// String returns the canonical format name.
func (f Format) String() string {
	if f == "" {
		return "auto"
	}
	return string(f)
}
