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
	return NewTripleDecoderWithOptions(r, format, DefaultDecodeOptions())
}

// NewTripleDecoderWithOptions creates a decoder for triple-only formats with options.
func NewTripleDecoderWithOptions(r io.Reader, format TripleFormat, opts DecodeOptions) (TripleDecoder, error) {
	switch format {
	case TripleFormatTurtle:
		return newTurtleTripleDecoderWithOptions(r, opts), nil
	case TripleFormatNTriples:
		return newNTriplesTripleDecoderWithOptions(r, opts), nil
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
	return NewQuadDecoderWithOptions(r, format, DefaultDecodeOptions())
}

// NewQuadDecoderWithOptions creates a decoder for quad-capable formats with options.
func NewQuadDecoderWithOptions(r io.Reader, format QuadFormat, opts DecodeOptions) (QuadDecoder, error) {
	switch format {
	case QuadFormatTriG:
		return newTriGQuadDecoderWithOptions(r, opts), nil
	case QuadFormatNQuads:
		return newNQuadsQuadDecoderWithOptions(r, opts), nil
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
