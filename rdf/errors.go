package rdf

import "errors"

var (
	// ErrUnsupportedFormat indicates an unsupported format.
	ErrUnsupportedFormat = errors.New("unsupported RDF format")
)
