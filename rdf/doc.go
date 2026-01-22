// Package rdf provides a compact RDF model with streaming parsers/encoders.
//
// It focuses on fast, low-allocation I/O with a small surface area:
//   - Decode: NewDecoder() returns a pull-style Decoder.
//   - Encode: NewEncoder() returns a push-style Encoder.
//   - Parse: Parse() and ParseChan() provide streaming helpers.
//
// Supported formats: Turtle, TriG, N-Triples, N-Quads, RDF/XML, JSON-LD.
//
// RDF-star is represented via TripleTerm, allowing quoted triples to appear
// as subjects or objects.
package rdf
