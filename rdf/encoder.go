package rdf

import (
	"context"
	"fmt"
	"io"
)

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

// DecoderOption configures decoder behavior using functional options.
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
// This allows backward compatibility with code that passes DecodeOptions structs.
func DecodeOptionsToOptions(opts DecodeOptions) []DecoderOption {
	return []DecoderOption{
		func(o *DecodeOptions) {
			*o = opts
		},
	}
}

// NewTripleDecoder creates a decoder for triple-only formats.
func NewTripleDecoder(r io.Reader, format TripleFormat) (TripleDecoder, error) {
	return NewTripleDecoderWithOptions(r, format, DecodeOptionsToOptions(DefaultDecodeOptions())...)
}

// NewTripleDecoderWithOptions creates a decoder using functional options.
// Example:
//   dec, err := NewTripleDecoderWithOptions(r, TripleFormatTurtle,
//       WithMaxLineBytes(64<<10),
//       WithMaxDepth(50),
//       WithContext(ctx))
func NewTripleDecoderWithOptions(r io.Reader, format TripleFormat, opts ...DecoderOption) (TripleDecoder, error) {
	options := DefaultDecodeOptions()
	for _, opt := range opts {
		opt(&options)
	}
	return newTripleDecoderWithOptions(r, format, options)
}

// newTripleDecoderWithOptions is the internal implementation that takes DecodeOptions struct.
func newTripleDecoderWithOptions(r io.Reader, format TripleFormat, opts DecodeOptions) (TripleDecoder, error) {
	if r == nil {
		return nil, fmt.Errorf("reader cannot be nil")
	}
	opts = normalizeDecodeOptions(opts)
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
	return NewQuadDecoderWithOptions(r, format, DecodeOptionsToOptions(DefaultDecodeOptions())...)
}

// NewQuadDecoderWithOptions creates a decoder using functional options.
// Example:
//   dec, err := NewQuadDecoderWithOptions(r, QuadFormatTriG,
//       WithMaxLineBytes(64<<10),
//       WithMaxDepth(50),
//       WithContext(ctx))
func NewQuadDecoderWithOptions(r io.Reader, format QuadFormat, opts ...DecoderOption) (QuadDecoder, error) {
	options := DefaultDecodeOptions()
	for _, opt := range opts {
		opt(&options)
	}
	return newQuadDecoderWithOptions(r, format, options)
}

// newQuadDecoderWithOptions is the internal implementation that takes DecodeOptions struct.
func newQuadDecoderWithOptions(r io.Reader, format QuadFormat, opts DecodeOptions) (QuadDecoder, error) {
	if r == nil {
		return nil, fmt.Errorf("reader cannot be nil")
	}
	opts = normalizeDecodeOptions(opts)
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
