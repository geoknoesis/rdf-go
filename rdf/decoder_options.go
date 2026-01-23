package rdf

import "context"

const (
	DefaultMaxLineBytes      = 1 << 20
	DefaultMaxStatementBytes = 4 << 20
)

// DecodeOptions configures parser behavior and limits.
// Zero values use defaults. Use negative values to disable specific limits.
type DecodeOptions struct {
	MaxLineBytes      int
	MaxStatementBytes int
	// AllowQuotedTripleStatement enables quoted triple statements in Turtle/TriG.
	AllowQuotedTripleStatement bool
	// DebugStatements wraps parse errors with the offending statement.
	DebugStatements bool
	// AllowEnvOverrides enables parsing behavior overrides via environment variables.
	AllowEnvOverrides bool
	// Context provides cancellation for decoding work.
	Context context.Context
}

// DefaultDecodeOptions returns safe defaults for parser limits.
func DefaultDecodeOptions() DecodeOptions {
	return DecodeOptions{
		MaxLineBytes:      DefaultMaxLineBytes,
		MaxStatementBytes: DefaultMaxStatementBytes,
	}
}

func normalizeDecodeOptions(opts DecodeOptions) DecodeOptions {
	if opts.MaxLineBytes == 0 {
		opts.MaxLineBytes = DefaultMaxLineBytes
	}
	if opts.MaxStatementBytes == 0 {
		opts.MaxStatementBytes = DefaultMaxStatementBytes
	}
	return opts
}
