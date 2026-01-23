// Package rdf provides a compact RDF model with streaming parsers/encoders.
//
// Copyright 2026 Geoknoesis LLC (www.geoknoesis.com)
//
// Author: Stephane Fellah (stephanef@geoknoesis.com)
// Geosemantic-AI expert with 30 years of experience
//
// It focuses on fast, low-allocation I/O with a small surface area and unified API:
//   - Read: NewReader() returns a unified reader that works with all formats.
//   - Write: NewWriter() returns a unified writer that works with all formats.
//   - Parse: Parse() provides streaming parsing with a handler function.
//   - ReadAll: ReadAll() provides convenience for small datasets.
//
// The API uses a unified Format type for all formats. Use FormatAuto to enable
// automatic format detection from input.
//
// Supported formats:
//   - Triple formats: Turtle, N-Triples, RDF/XML, JSON-LD
//   - Quad formats: TriG, N-Quads
//
// RDF-star is represented via TripleTerm, allowing quoted triples to appear
// as subjects or objects.
//
// Example (decoding with auto-detection):
//
//	dec, err := rdf.NewReader(strings.NewReader(input), rdf.FormatAuto)
//	if err != nil {
//	    // handle error
//	}
//	defer dec.Close()
//
//	for {
//	    stmt, err := dec.Next()
//	    if err == io.EOF {
//	        break
//	    }
//	    if err != nil {
//	        // handle error
//	    }
//	    // process statement (use stmt.IsTriple() or stmt.IsQuad() to check type)
//	}
//
// Example (creating statements):
//
//	// Option 1: Omit G (defaults to nil for triples)
//	stmt := rdf.Statement{
//	    S: rdf.IRI{Value: "http://example.org/s"},
//	    P: rdf.IRI{Value: "http://example.org/p"},
//	    O: rdf.IRI{Value: "http://example.org/o"},
//	    // G omitted - defaults to nil (triple)
//	}
//
//	// Option 2: Use convenience function
//	stmt := rdf.NewTriple(
//	    rdf.IRI{Value: "http://example.org/s"},
//	    rdf.IRI{Value: "http://example.org/p"},
//	    rdf.IRI{Value: "http://example.org/o"},
//	)
//
// Example (parsing with handler):
//
//	err := rdf.Parse(context.Background(), reader, rdf.FormatTurtle, func(s rdf.Statement) error {
//	    // process statement
//	    return nil
//	})
//
// For unsupported formats, NewReader and NewWriter return ErrUnsupportedFormat.
//
// The API is intentionally small and favors streaming. For large inputs,
// prefer NewReader or Parse instead of ReadAll.
//
// Options can be provided via functional options (OptContext, OptMaxDepth, etc.)
// to configure behavior and enforce limits for untrusted input.
//
// RDF/XML container elements (rdf:Bag, rdf:Seq, rdf:Alt, rdf:List) are parsed as node
// elements; container membership expansion is not implemented.
package rdf
