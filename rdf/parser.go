package rdf

import (
	"context"
	"io"
)

// TripleDecoder streams RDF triples from an input.
type TripleDecoder interface {
	Next() (Triple, error)
	Err() error
	Close() error
}

// QuadDecoder streams RDF quads from an input.
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

// ParseTriples streams RDF triples to a handler.
func ParseTriples(ctx context.Context, r io.Reader, format TripleFormat, handler TripleHandler) error {
	return ParseTriplesWithOptions(ctx, r, format, DefaultDecodeOptions(), handler)
}

// ParseTriplesWithOptions streams RDF triples to a handler with decoder options.
func ParseTriplesWithOptions(ctx context.Context, r io.Reader, format TripleFormat, opts DecodeOptions, handler TripleHandler) error {
	if opts.Context == nil {
		opts.Context = ctx
	}
	if ctx != nil {
		r = &contextReader{ctx: ctx, r: r}
	}
	decoder, err := NewTripleDecoderWithOptions(r, format, DecodeOptionsToOptions(opts)...)
	if err != nil {
		return err
	}
	defer decoder.Close()
	return parseTriplesWithDecoder(ctx, decoder, handler)
}

func parseTriplesWithDecoder(ctx context.Context, decoder TripleDecoder, handler TripleHandler) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		triple, err := decoder.Next()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if err := handler.Handle(triple); err != nil {
			return err
		}
	}
}

// ParseQuads streams RDF quads to a handler.
func ParseQuads(ctx context.Context, r io.Reader, format QuadFormat, handler QuadHandler) error {
	return ParseQuadsWithOptions(ctx, r, format, DefaultDecodeOptions(), handler)
}

// ParseQuadsWithOptions streams RDF quads to a handler with decoder options.
func ParseQuadsWithOptions(ctx context.Context, r io.Reader, format QuadFormat, opts DecodeOptions, handler QuadHandler) error {
	if opts.Context == nil {
		opts.Context = ctx
	}
	if ctx != nil {
		r = &contextReader{ctx: ctx, r: r}
	}
	decoder, err := NewQuadDecoderWithOptions(r, format, DecodeOptionsToOptions(opts)...)
	if err != nil {
		return err
	}
	defer decoder.Close()
	return parseQuadsWithDecoder(ctx, decoder, handler)
}

func parseQuadsWithDecoder(ctx context.Context, decoder QuadDecoder, handler QuadHandler) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		quad, err := decoder.Next()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if quad.IsZero() {
			continue
		}
		if err := handler.Handle(quad); err != nil {
			return err
		}
	}
}

// ParseTriplesChan returns a channel of triples and an error channel.
func ParseTriplesChan(ctx context.Context, r io.Reader, format TripleFormat) (<-chan Triple, <-chan error) {
	return ParseTriplesChanWithOptions(ctx, r, format, DefaultDecodeOptions())
}

// ParseTriplesChanWithOptions returns a channel of triples and an error channel with decoder options.
func ParseTriplesChanWithOptions(ctx context.Context, r io.Reader, format TripleFormat, opts DecodeOptions) (<-chan Triple, <-chan error) {
	out := make(chan Triple)
	errs := make(chan error, 1)
	go func() {
		defer close(out)
		defer close(errs)
		err := ParseTriplesWithOptions(ctx, r, format, opts, TripleHandlerFunc(func(t Triple) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case out <- t:
				return nil
			}
		}))
		if err != nil && err != context.Canceled {
			errs <- err
		}
	}()
	return out, errs
}

// ParseQuadsChan returns a channel of quads and an error channel.
func ParseQuadsChan(ctx context.Context, r io.Reader, format QuadFormat) (<-chan Quad, <-chan error) {
	return ParseQuadsChanWithOptions(ctx, r, format, DefaultDecodeOptions())
}

// ParseQuadsChanWithOptions returns a channel of quads and an error channel with decoder options.
func ParseQuadsChanWithOptions(ctx context.Context, r io.Reader, format QuadFormat, opts DecodeOptions) (<-chan Quad, <-chan error) {
	out := make(chan Quad)
	errs := make(chan error, 1)
	go func() {
		defer close(out)
		defer close(errs)
		err := ParseQuadsWithOptions(ctx, r, format, opts, QuadHandlerFunc(func(q Quad) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case out <- q:
				return nil
			}
		}))
		if err != nil && err != context.Canceled {
			errs <- err
		}
	}()
	return out, errs
}
