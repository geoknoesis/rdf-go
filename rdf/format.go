package rdf

import "strings"

// Format identifies RDF serialization formats.
type Format string

const (
	FormatTurtle   Format = "turtle"
	FormatTriG     Format = "trig"
	FormatNTriples Format = "ntriples"
	FormatNQuads   Format = "nquads"
	FormatRDFXML   Format = "rdfxml"
	FormatJSONLD   Format = "jsonld"
)

// ParseFormat normalizes a format string.
func ParseFormat(value string) (Format, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "turtle", "ttl":
		return FormatTurtle, true
	case "trig":
		return FormatTriG, true
	case "ntriples", "nt":
		return FormatNTriples, true
	case "nquads", "nq":
		return FormatNQuads, true
	case "rdfxml", "rdf", "xml":
		return FormatRDFXML, true
	case "jsonld", "json-ld", "json":
		return FormatJSONLD, true
	default:
		return "", false
	}
}
