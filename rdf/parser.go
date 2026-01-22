package rdf

import (
	"context"
	"io"
)

// Decoder streams RDF quads from an input.
type Decoder interface {
	Next() (Quad, error)
	Err() error
	Close() error
}

// Handler processes quads in push mode.
type Handler interface {
	Handle(Quad) error
}

// HandlerFunc adapts a function to a Handler.
type HandlerFunc func(Quad) error

func (h HandlerFunc) Handle(q Quad) error { return h(q) }

// Parse streams RDF data to a handler.
func Parse(ctx context.Context, r io.Reader, format Format, handler Handler) error {
	decoder, err := NewDecoder(r, format)
	if err != nil {
		return err
	}
	defer decoder.Close()
	return parseWithDecoder(ctx, decoder, handler)
}

func parseWithDecoder(ctx context.Context, decoder Decoder, handler Handler) error {
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

// ParseChan returns a channel of quads and an error channel.
func ParseChan(ctx context.Context, r io.Reader, format Format) (<-chan Quad, <-chan error) {
	out := make(chan Quad)
	errs := make(chan error, 1)
	go func() {
		defer close(out)
		defer close(errs)
		err := Parse(ctx, r, format, HandlerFunc(func(q Quad) error {
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
