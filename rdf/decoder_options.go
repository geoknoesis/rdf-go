package rdf

import "context"

const (
	DefaultMaxLineBytes      = 1 << 20    // 1MB
	DefaultMaxStatementBytes = 4 << 20    // 4MB
	DefaultMaxDepth          = 100        // Maximum nesting depth for collections, blank node lists, etc.
	DefaultMaxTriples        = 10_000_000 // Maximum number of triples/quads to process (0 = unlimited)
)

// decodeOptions configures parser behavior and limits.
// Zero values use defaults. Use negative values to disable specific limits.
// For untrusted input, always set explicit limits.
type decodeOptions struct {
	// MaxLineBytes limits the size of a single line. Zero uses default (1MB).
	MaxLineBytes int
	// MaxStatementBytes limits the size of a complete statement. Zero uses default (4MB).
	MaxStatementBytes int
	// MaxDepth limits nesting depth for collections, blank node lists, and nested structures.
	// Zero uses default (100). Negative values disable the limit (not recommended for untrusted input).
	MaxDepth int
	// MaxTriples limits the total number of triples/quads to process.
	// Zero uses default (10M). Negative values disable the limit (not recommended for untrusted input).
	MaxTriples int64
	// AllowQuotedTripleStatement enables quoted triple statements in Turtle/TriG.
	AllowQuotedTripleStatement bool
	// DebugStatements wraps parse errors with the offending statement.
	DebugStatements bool
	// AllowEnvOverrides enables parsing behavior overrides via environment variables.
	AllowEnvOverrides bool
	// Context provides cancellation for decoding work. Should be checked periodically.
	Context context.Context
	// StrictIRIValidation enables strict IRI validation according to RFC 3987.
	// When enabled, all IRIs are validated for correct syntax during parsing.
	StrictIRIValidation bool
	// ExpandRDFXMLContainers enables RDF/XML container membership expansion.
	// When enabled (default), container elements automatically generate container
	// membership properties (rdf:_1, rdf:_2, etc.) from rdf:li elements.
	ExpandRDFXMLContainers bool
}

// defaultDecodeOptions returns safe defaults for parser limits.
// These defaults are suitable for trusted input. For untrusted input, use stricter limits.
func defaultDecodeOptions() decodeOptions {
	return decodeOptions{
		MaxLineBytes:           DefaultMaxLineBytes,
		MaxStatementBytes:      DefaultMaxStatementBytes,
		MaxDepth:               DefaultMaxDepth,
		MaxTriples:             DefaultMaxTriples,
		ExpandRDFXMLContainers: true, // Container expansion enabled by default
	}
}

// safeDecodeOptions returns stricter limits suitable for untrusted input.
func safeDecodeOptions() decodeOptions {
	return decodeOptions{
		MaxLineBytes:      64 << 10,  // 64KB per line
		MaxStatementBytes: 256 << 10, // 256KB per statement
		MaxDepth:          50,        // 50 levels of nesting
		MaxTriples:        1_000_000, // 1M triples
	}
}

func normalizeDecodeOptions(opts decodeOptions) decodeOptions {
	if opts.MaxLineBytes == 0 {
		opts.MaxLineBytes = DefaultMaxLineBytes
	}
	if opts.MaxStatementBytes == 0 {
		opts.MaxStatementBytes = DefaultMaxStatementBytes
	}
	if opts.MaxDepth == 0 {
		opts.MaxDepth = DefaultMaxDepth
	}
	if opts.MaxTriples == 0 {
		opts.MaxTriples = DefaultMaxTriples
	}
	// ExpandRDFXMLContainers is handled by defaultOptions() which sets it to true
	// If it's false here, it means OptDisableRDFXMLContainerExpansion() was called
	// So we respect the user's choice and don't override it
	return opts
}
