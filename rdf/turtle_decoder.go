package rdf

import "io"

// New triple decoder for Turtle
type turtleTripleDecoder struct {
	parser *turtleParser
}

func newTurtleTripleDecoder(r io.Reader) TripleDecoder {
	return newTurtleTripleDecoderWithOptions(r, DefaultDecodeOptions())
}

func newTurtleTripleDecoderWithOptions(r io.Reader, opts DecodeOptions) TripleDecoder {
	return &turtleTripleDecoder{
		parser: newTurtleParser(r, opts),
	}
}

func (d *turtleTripleDecoder) Next() (Triple, error) {
	return d.parser.NextTriple()
}

func (d *turtleTripleDecoder) Err() error { return d.parser.Err() }
func (d *turtleTripleDecoder) Close() error {
	return nil
}
