package rdf

import "strings"

// TripleFormat represents RDF formats that only support triples.
type TripleFormat string

const (
	// TripleFormatTurtle is the Turtle format.
	TripleFormatTurtle TripleFormat = "turtle"
	// TripleFormatNTriples is the N-Triples format.
	TripleFormatNTriples TripleFormat = "ntriples"
	// TripleFormatRDFXML is the RDF/XML format.
	TripleFormatRDFXML TripleFormat = "rdfxml"
	// TripleFormatJSONLD is the JSON-LD format.
	TripleFormatJSONLD TripleFormat = "jsonld"
)

// QuadFormat represents RDF formats that support quads (named graphs).
type QuadFormat string

const (
	// QuadFormatTriG is the TriG format.
	QuadFormatTriG QuadFormat = "trig"
	// QuadFormatNQuads is the N-Quads format.
	QuadFormatNQuads QuadFormat = "nquads"
)

// ParseTripleFormat normalizes a format string and returns a TripleFormat if valid.
func ParseTripleFormat(value string) (TripleFormat, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "turtle", "ttl":
		return TripleFormatTurtle, true
	case "ntriples", "nt":
		return TripleFormatNTriples, true
	case "rdfxml", "rdf", "xml":
		return TripleFormatRDFXML, true
	case "jsonld", "json-ld", "json":
		return TripleFormatJSONLD, true
	default:
		return "", false
	}
}

// ParseQuadFormat normalizes a format string and returns a QuadFormat if valid.
func ParseQuadFormat(value string) (QuadFormat, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "trig":
		return QuadFormatTriG, true
	case "nquads", "nq":
		return QuadFormatNQuads, true
	default:
		return "", false
	}
}
