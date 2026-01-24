package rdf

import "io"

// New triple decoder for Turtle
type turtletripleDecoder struct {
	parser *turtleParser
}

func newTurtletripleDecoder(r io.Reader) tripleDecoder {
	return newTurtletripleDecoderWithOptions(r, defaultDecodeOptions())
}

func newTurtletripleDecoderWithOptions(r io.Reader, opts decodeOptions) tripleDecoder {
	return &turtletripleDecoder{
		parser: newTurtleParser(r, opts),
	}
}

func (d *turtletripleDecoder) Next() (Triple, error) {
	return d.parser.NextTriple()
}

func (d *turtletripleDecoder) Err() error { return d.parser.Err() }
func (d *turtletripleDecoder) Close() error {
	return nil
}
