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
//	// Simplest: pass nil for context (uses context.Background() automatically)
//	err := rdf.Parse(nil, reader, rdf.FormatTurtle, func(s rdf.Statement) error {
//	    // process statement
//	    return nil
//	})
//
//	// For cancellation or timeouts, pass an explicit context:
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//	err := rdf.Parse(ctx, reader, rdf.FormatTurtle, handler)
//
// For unsupported formats, NewReader and NewWriter return ErrUnsupportedFormat.
//
// The API is intentionally small and favors streaming. For large inputs,
// prefer NewReader or Parse for streaming.
//
// Options can be provided via functional options (OptContext, OptMaxDepth, etc.)
// to configure behavior and enforce limits for untrusted input.
//
// RDF/XML container elements (rdf:Bag, rdf:Seq, rdf:Alt, rdf:List) support
// container membership expansion via the ExpandRDFXMLContainers option (enabled by default).
//
// Error Handling:
//
// The library uses structured error codes for programmatic error handling.
// Use Code() to get error codes, and ParseError for detailed error context.
// See ERROR_HANDLING.md for comprehensive error handling guide and recovery strategies.
//
// Performance:
//
// The library is optimized for streaming and low memory allocation.
// Comprehensive benchmarks and profiling tools are available (see benchmarks_test.go
// and benchmark_profiling.go). Performance regression tests ensure consistent performance.
//
// Security:
//
// For untrusted input, use SafeMode() or OptSafeLimits() to enforce conservative limits.
// Default limits are suitable for trusted input but may be too permissive for untrusted data.
//
// Documentation:
//
// - Package API: See this file (doc.go) for complete API documentation
// - Error Handling: See ERROR_HANDLING.md for error handling guide
// - Examples: See examples_test.go for usage examples
// - Benchmarks: See benchmarks_test.go for performance benchmarks
package rdf
