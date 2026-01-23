// Package rdf provides a compact RDF model with streaming parsers/encoders.
//
// Copyright 2026 Geoknoesis LLC (www.geoknoesis.com)
//
// Author: Stephane Fellah (stephanef@geoknoesis.com)
// Geosemantic-AI expert with 30 years of experience
//
// It focuses on fast, low-allocation I/O with a small surface area and type safety:
//   - Decode: NewTripleDecoder() and NewQuadDecoder() return pull-style decoders.
//   - Encode: NewTripleEncoder() and NewQuadEncoder() return push-style encoders.
//   - Parse: ParseTriples() and ParseQuads() provide streaming helpers.
//   - ParseChan: ParseTriplesChan() and ParseQuadsChan() provide channel-based parsing.
//
// The API enforces type safety: triple formats can only be used with triple decoders/encoders,
// and quad formats can only be used with quad decoders/encoders. This prevents format
// mismatches at compile time.
//
// Supported formats:
//   - Triple formats: Turtle, N-Triples, RDF/XML, JSON-LD
//   - Quad formats: TriG, N-Quads
//
// RDF-star is represented via TripleTerm, allowing quoted triples to appear
// as subjects or objects.
//
// Example (decoding triples):
//
//	dec, err := rdf.NewTripleDecoder(strings.NewReader(input), rdf.TripleFormatNTriples)
//	if err != nil {
//	    // handle error
//	}
//	defer dec.Close()
//
//	for {
//	    triple, err := dec.Next()
//	    if err == io.EOF {
//	        break
//	    }
//	    if err != nil {
//	        // handle error
//	    }
//	    // process triple.S, triple.P, triple.O
//	}
//
// Example (decoding quads):
//
//	dec, err := rdf.NewQuadDecoder(strings.NewReader(input), rdf.QuadFormatNQuads)
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
// For unsupported formats, NewTripleDecoder, NewQuadDecoder, NewTripleEncoder,
// and NewQuadEncoder return ErrUnsupportedFormat.
//
// The API is intentionally small and favors streaming. For large inputs,
// prefer NewTripleDecoder/NewQuadDecoder or ParseTriples/ParseQuads instead
// of buffering all results.
//
// Decoder options can be provided via NewTripleDecoderWithOptions and
// NewQuadDecoderWithOptions to enforce line/statement limits for untrusted input.
// Streaming helpers also have WithOptions variants for the same purpose.
//
// RDF/XML container elements (rdf:Bag, rdf:Seq, rdf:Alt, rdf:List) are parsed as node
// elements; container membership expansion is not implemented.
package rdf
