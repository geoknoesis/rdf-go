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
//
// Example:
//
//	dec, err := rdf.NewDecoder(strings.NewReader(input), rdf.FormatNTriples)
//	if err != nil {
//	    // handle error
//	}
//	defer dec.Close()
//
//	for {
//	    quad, err := dec.Next()
//	    if err == io.EOF {
//	        break
//	    }
//	    if err != nil {
//	        // handle error
//	    }
//	    // process quad.S, quad.P, quad.O, quad.G
//	}
//
// For unsupported formats, NewDecoder and NewEncoder return ErrUnsupportedFormat.
//
// The API is intentionally small and favors streaming. For large inputs,
// prefer NewDecoder or Parse instead of buffering all results.
package rdf
