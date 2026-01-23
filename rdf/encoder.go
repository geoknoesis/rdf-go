package rdf

import (
	"context"
	"io"
)

// TripleEncoder streams RDF triples to an output.
// This interface is used internally by the unified Writer adapter.
type TripleEncoder interface {
	Write(Triple) error
	Flush() error
	Close() error
}

// QuadEncoder streams RDF quads to an output.
// This interface is used internally by the unified Writer adapter.
type QuadEncoder interface {
	Write(Quad) error
	Flush() error
	Close() error
}

// DecoderOption configures decoder behavior using functional options.
// This is kept for internal use with the old decoder implementations.
type DecoderOption func(*DecodeOptions)

// WithMaxLineBytes sets the maximum line size limit.
func WithMaxLineBytes(maxBytes int) DecoderOption {
	return func(opts *DecodeOptions) {
		opts.MaxLineBytes = maxBytes
	}
}

// WithMaxStatementBytes sets the maximum statement size limit.
func WithMaxStatementBytes(maxBytes int) DecoderOption {
	return func(opts *DecodeOptions) {
		opts.MaxStatementBytes = maxBytes
	}
}

// WithMaxDepth sets the maximum nesting depth limit.
func WithMaxDepth(maxDepth int) DecoderOption {
	return func(opts *DecodeOptions) {
		opts.MaxDepth = maxDepth
	}
}

// WithMaxTriples sets the maximum number of triples/quads to process.
func WithMaxTriples(maxTriples int64) DecoderOption {
	return func(opts *DecodeOptions) {
		opts.MaxTriples = maxTriples
	}
}

// WithContext sets the context for cancellation and timeouts.
func WithContext(ctx context.Context) DecoderOption {
	return func(opts *DecodeOptions) {
		opts.Context = ctx
	}
}

// WithAllowQuotedTripleStatement enables quoted triple statements in Turtle/TriG.
func WithAllowQuotedTripleStatement(allow bool) DecoderOption {
	return func(opts *DecodeOptions) {
		opts.AllowQuotedTripleStatement = allow
	}
}

// WithDebugStatements enables debug statement wrapping in errors.
func WithDebugStatements(debug bool) DecoderOption {
	return func(opts *DecodeOptions) {
		opts.DebugStatements = debug
	}
}

// WithSafeLimits applies safe limits suitable for untrusted input.
func WithSafeLimits() DecoderOption {
	return func(opts *DecodeOptions) {
		safe := SafeDecodeOptions()
		opts.MaxLineBytes = safe.MaxLineBytes
		opts.MaxStatementBytes = safe.MaxStatementBytes
		opts.MaxDepth = safe.MaxDepth
		opts.MaxTriples = safe.MaxTriples
	}
}

// DecodeOptionsToOptions converts a DecodeOptions struct to functional options.
func DecodeOptionsToOptions(opts DecodeOptions) []DecoderOption {
	return []DecoderOption{
		func(o *DecodeOptions) {
			*o = opts
		},
	}
}

// newTripleDecoderWithOptions creates a decoder using the old format types (internal use only).
func newTripleDecoderWithOptions(r io.Reader, format string, opts DecodeOptions) (TripleDecoder, error) {
	decodeOpts := normalizeDecodeOptions(opts)
	switch format {
	case "turtle":
		return newTurtleTripleDecoderWithOptions(r, decodeOpts), nil
	case "ntriples":
		return newNTriplesTripleDecoderWithOptions(r, decodeOpts), nil
	case "rdfxml":
		return newRDFXMLTripleDecoder(r), nil
	case "jsonld":
		return newJSONLDTripleDecoder(r), nil
	default:
		return nil, ErrUnsupportedFormat
	}
}

// newQuadDecoderWithOptions creates a decoder using the old format types (internal use only).
func newQuadDecoderWithOptions(r io.Reader, format string, opts DecodeOptions) (QuadDecoder, error) {
	decodeOpts := normalizeDecodeOptions(opts)
	switch format {
	case "trig":
		return newTriGQuadDecoderWithOptions(r, decodeOpts), nil
	case "nquads":
		return newNQuadsQuadDecoderWithOptions(r, decodeOpts), nil
	default:
		return nil, ErrUnsupportedFormat
	}
}

// newTripleEncoder creates an encoder using the old format types (internal use only).
func newTripleEncoder(w io.Writer, format string) (TripleEncoder, error) {
	switch format {
	case "turtle":
		return newTurtleTripleEncoder(w), nil
	case "ntriples":
		return newNTriplesTripleEncoder(w), nil
	case "rdfxml":
		return newRDFXMLTripleEncoder(w), nil
	case "jsonld":
		return newJSONLDTripleEncoder(w), nil
	default:
		return nil, ErrUnsupportedFormat
	}
}

// newQuadEncoder creates an encoder using the old format types (internal use only).
func newQuadEncoder(w io.Writer, format string) (QuadEncoder, error) {
	switch format {
	case "trig":
		return newTriGQuadEncoder(w), nil
	case "nquads":
		return newNQuadsQuadEncoder(w), nil
	default:
		return nil, ErrUnsupportedFormat
	}
}
