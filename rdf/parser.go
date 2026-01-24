package rdf

// tripleDecoder streams RDF triples from an input.
// This interface is used internally by the unified Reader adapter.
type tripleDecoder interface {
	Next() (Triple, error)
	Err() error
	Close() error
}

// quadDecoder streams RDF quads from an input.
// This interface is used internally by the unified Reader adapter.
type quadDecoder interface {
	Next() (Quad, error)
	Err() error
	Close() error
}

// tripleHandler processes triples in push mode.
type tripleHandler interface {
	Handle(Triple) error
}

// tripleHandlerFunc adapts a function to a tripleHandler.
type tripleHandlerFunc func(Triple) error

// Handle calls the underlying function.
func (h tripleHandlerFunc) Handle(t Triple) error { return h(t) }

// quadHandler processes quads in push mode.
type quadHandler interface {
	Handle(Quad) error
}

// quadHandlerFunc adapts a function to a quadHandler.
type quadHandlerFunc func(Quad) error

// Handle calls the underlying function.
func (h quadHandlerFunc) Handle(q Quad) error { return h(q) }
