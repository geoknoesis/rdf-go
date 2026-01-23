package rdf

import "io"

// TripleEncoder streams RDF triples to an output.
type TripleEncoder interface {
	Write(Triple) error
	Flush() error
	Close() error
}

// QuadEncoder streams RDF quads to an output.
type QuadEncoder interface {
	Write(Quad) error
	Flush() error
	Close() error
}

// NewTripleDecoder creates a decoder for triple-only formats.
func NewTripleDecoder(r io.Reader, format TripleFormat) (TripleDecoder, error) {
	switch format {
	case TripleFormatTurtle:
		return newTurtleTripleDecoder(r), nil
	case TripleFormatNTriples:
		return newNTriplesTripleDecoder(r), nil
	case TripleFormatRDFXML:
		return newRDFXMLTripleDecoder(r), nil
	case TripleFormatJSONLD:
		return newJSONLDTripleDecoder(r), nil
	default:
		return nil, ErrUnsupportedFormat
	}
}

// NewQuadDecoder creates a decoder for quad-capable formats.
func NewQuadDecoder(r io.Reader, format QuadFormat) (QuadDecoder, error) {
	switch format {
	case QuadFormatTriG:
		return newTriGQuadDecoder(r), nil
	case QuadFormatNQuads:
		return newNQuadsQuadDecoder(r), nil
	default:
		return nil, ErrUnsupportedFormat
	}
}

// NewTripleEncoder creates an encoder for triple-only formats.
func NewTripleEncoder(w io.Writer, format TripleFormat) (TripleEncoder, error) {
	switch format {
	case TripleFormatTurtle:
		return newTurtleTripleEncoder(w), nil
	case TripleFormatNTriples:
		return newNTriplesTripleEncoder(w), nil
	case TripleFormatRDFXML:
		return newRDFXMLTripleEncoder(w), nil
	case TripleFormatJSONLD:
		return newJSONLDTripleEncoder(w), nil
	default:
		return nil, ErrUnsupportedFormat
	}
}

// NewQuadEncoder creates an encoder for quad-capable formats.
func NewQuadEncoder(w io.Writer, format QuadFormat) (QuadEncoder, error) {
	switch format {
	case QuadFormatTriG:
		return newTriGQuadEncoder(w), nil
	case QuadFormatNQuads:
		return newNQuadsQuadEncoder(w), nil
	default:
		return nil, ErrUnsupportedFormat
	}
}

