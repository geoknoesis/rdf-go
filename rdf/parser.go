package rdf

// TripleDecoder streams RDF triples from an input.
// This interface is used internally by the unified Reader adapter.
type TripleDecoder interface {
	Next() (Triple, error)
	Err() error
	Close() error
}

// QuadDecoder streams RDF quads from an input.
// This interface is used internally by the unified Reader adapter.
type QuadDecoder interface {
	Next() (Quad, error)
	Err() error
	Close() error
}

// TripleHandler processes triples in push mode.
type TripleHandler interface {
	Handle(Triple) error
}

// TripleHandlerFunc adapts a function to a TripleHandler.
type TripleHandlerFunc func(Triple) error

// Handle calls the underlying function.
func (h TripleHandlerFunc) Handle(t Triple) error { return h(t) }

// QuadHandler processes quads in push mode.
type QuadHandler interface {
	Handle(Quad) error
}

// QuadHandlerFunc adapts a function to a QuadHandler.
type QuadHandlerFunc func(Quad) error

// Handle calls the underlying function.
func (h QuadHandlerFunc) Handle(q Quad) error { return h(q) }


