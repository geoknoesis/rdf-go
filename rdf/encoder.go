package rdf

import (
	"context"
	"io"
)

// tripleEncoder streams RDF triples to an output.
// This interface is used internally by the unified Writer adapter.
type tripleEncoder interface {
	Write(Triple) error
	Flush() error
	Close() error
}

// quadEncoder streams RDF quads to an output.
// This interface is used internally by the unified Writer adapter.
type quadEncoder interface {
	Write(Quad) error
	Flush() error
	Close() error
}

// decoderOption configures decoder behavior using functional options.
// This is kept for internal use with the old decoder implementations.
type decoderOption func(*decodeOptions)

// withMaxLineBytes sets the maximum line size limit.
func withMaxLineBytes(maxBytes int) decoderOption {
	return func(opts *decodeOptions) {
		opts.MaxLineBytes = maxBytes
	}
}

// withMaxStatementBytes sets the maximum statement size limit.
func withMaxStatementBytes(maxBytes int) decoderOption {
	return func(opts *decodeOptions) {
		opts.MaxStatementBytes = maxBytes
	}
}

// withMaxDepth sets the maximum nesting depth limit.
func withMaxDepth(maxDepth int) decoderOption {
	return func(opts *decodeOptions) {
		opts.MaxDepth = maxDepth
	}
}

// withMaxTriples sets the maximum number of triples/quads to process.
func withMaxTriples(maxTriples int64) decoderOption {
	return func(opts *decodeOptions) {
		opts.MaxTriples = maxTriples
	}
}

// withContext sets the context for cancellation and timeouts.
func withContext(ctx context.Context) decoderOption {
	return func(opts *decodeOptions) {
		opts.Context = ctx
	}
}

// withAllowQuotedTripleStatement enables quoted triple statements in Turtle/TriG.
func withAllowQuotedTripleStatement(allow bool) decoderOption {
	return func(opts *decodeOptions) {
		opts.AllowQuotedTripleStatement = allow
	}
}

// withDebugStatements enables debug statement wrapping in errors.
func withDebugStatements(debug bool) decoderOption {
	return func(opts *decodeOptions) {
		opts.DebugStatements = debug
	}
}

// withSafeLimits applies safe limits suitable for untrusted input.
func withSafeLimits() decoderOption {
	return func(opts *decodeOptions) {
		safe := safeDecodeOptions()
		opts.MaxLineBytes = safe.MaxLineBytes
		opts.MaxStatementBytes = safe.MaxStatementBytes
		opts.MaxDepth = safe.MaxDepth
		opts.MaxTriples = safe.MaxTriples
	}
}

// newTripleDecoderWithOptions creates a decoder using the old format types (internal use only).
func newTripleDecoderWithOptions(r io.Reader, format string, opts decodeOptions) (tripleDecoder, error) {
	decodeOpts := normalizeDecodeOptions(opts)
	switch format {
	case "turtle":
		return newTurtletripleDecoderWithOptions(r, decodeOpts), nil
	case "ntriples":
		return newNTriplestripleDecoderWithOptions(r, decodeOpts), nil
	case "rdfxml":
		return newRDFXMLtripleDecoderWithOptions(r, decodeOpts), nil
	case "jsonld":
		return newJSONLDtripleDecoder(r), nil
	default:
		return nil, ErrUnsupportedFormat
	}
}

// newQuadDecoderWithOptions creates a decoder using the old format types (internal use only).
func newQuadDecoderWithOptions(r io.Reader, format string, opts decodeOptions) (quadDecoder, error) {
	decodeOpts := normalizeDecodeOptions(opts)
	switch format {
	case "trig":
		return newTriGquadDecoderWithOptions(r, decodeOpts), nil
	case "nquads":
		return newNQuadsquadDecoderWithOptions(r, decodeOpts), nil
	default:
		return nil, ErrUnsupportedFormat
	}
}

// newTripleEncoder creates an encoder using the old format types (internal use only).
func newTripleEncoder(w io.Writer, format string) (tripleEncoder, error) {
	switch format {
	case "turtle":
		return newTurtletripleEncoder(w), nil
	case "ntriples":
		return newNTriplestripleEncoder(w), nil
	case "rdfxml":
		return newRDFXMLtripleEncoder(w), nil
	case "jsonld":
		return newJSONLDtripleEncoder(w), nil
	default:
		return nil, ErrUnsupportedFormat
	}
}

// newQuadEncoder creates an encoder using the old format types (internal use only).
func newQuadEncoder(w io.Writer, format string) (quadEncoder, error) {
	switch format {
	case "trig":
		return newTriGquadEncoder(w), nil
	case "nquads":
		return newNQuadsquadEncoder(w), nil
	default:
		return nil, ErrUnsupportedFormat
	}
}
