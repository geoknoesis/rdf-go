package rdf

import "io"

// Encoder streams RDF quads to an output.
type Encoder interface {
	Write(Quad) error
	Flush() error
	Close() error
}

// NewDecoder creates a decoder for the given format.
func NewDecoder(r io.Reader, format Format) (Decoder, error) {
	switch format {
	case FormatNTriples:
		return newNTriplesDecoder(r), nil
	case FormatNQuads:
		return newNQuadsDecoder(r), nil
	case FormatTurtle:
		return newTurtleDecoder(r), nil
	case FormatTriG:
		return newTriGDecoder(r), nil
	case FormatRDFXML:
		return newRDFXMLDecoder(r), nil
	case FormatJSONLD:
		return newJSONLDDecoder(r), nil
	default:
		return nil, ErrUnsupportedFormat
	}
}

// NewEncoder creates an encoder for the given format.
func NewEncoder(w io.Writer, format Format) (Encoder, error) {
	switch format {
	case FormatNTriples:
		return newNTriplesEncoder(w), nil
	case FormatNQuads:
		return newNQuadsEncoder(w), nil
	case FormatTurtle:
		return newTurtleEncoder(w), nil
	case FormatTriG:
		return newTriGEncoder(w), nil
	case FormatRDFXML:
		return newRDFXMLEncoder(w), nil
	case FormatJSONLD:
		return newJSONLDEncoder(w), nil
	default:
		return nil, ErrUnsupportedFormat
	}
}
