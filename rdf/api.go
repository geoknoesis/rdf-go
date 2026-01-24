package rdf

import (
	"bytes"
	"context"
	"io"
)

// Reader streams RDF statements from an input.
// A statement can be either a triple (G is nil) or a quad (G is non-nil).
type Reader interface {
	Next() (Statement, error)
	Close() error
}

// Writer streams RDF statements to an output.
// For triple-only formats, the graph (G) field is ignored.
type Writer interface {
	Write(Statement) error
	Flush() error
	Close() error
}

// Handler processes statements in push mode.
type Handler func(Statement) error

// Option configures reader/writer behavior.
type Option func(*Options)

// Options configures parser/encoder behavior.
type Options struct {
	// Context for cancellation and timeouts
	Context context.Context

	// Security limits for untrusted input
	MaxLineBytes      int
	MaxStatementBytes int
	MaxDepth          int
	MaxTriples        int64

	// Format-specific options
	AllowQuotedTripleStatement bool
	DebugStatements            bool

	// IRI validation
	StrictIRIValidation bool // Enable strict IRI validation according to RFC 3987

	// RDF/XML container expansion
	ExpandRDFXMLContainers bool // Enable RDF/XML container membership expansion (default: true)
}

// NewReader creates a reader for the specified format.
// If format is FormatAuto (empty string), the format is automatically detected.
// Auto-detection reads from the reader, so the reader position will be advanced.
func NewReader(r io.Reader, format Format, opts ...Option) (Reader, error) {
	options := defaultOptions()
	for _, opt := range opts {
		opt(&options)
	}

	// Auto-detect format if needed
	if format == FormatAuto {
		detected, reader, ok := detectFormat(r)
		if !ok {
			return nil, ErrUnsupportedFormat
		}
		format = detected
		r = reader // Use reader that includes buffered bytes
	}

	return newDecoder(r, format, options)
}

// Parse parses RDF from the reader and streams statements to the handler.
// If format is FormatAuto (empty string), the format is automatically detected.
// If ctx is nil, context.Background() is used as the default.
func Parse(ctx context.Context, r io.Reader, format Format, handler Handler, opts ...Option) error {
	options := defaultOptions()
	// Default to context.Background() if ctx is nil
	if ctx == nil {
		ctx = context.Background()
	}
	options.Context = ctx
	for _, opt := range opts {
		opt(&options)
	}

	reader, err := NewReader(r, format, opts...)
	if err != nil {
		return err
	}
	defer reader.Close()

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		stmt, err := reader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		if err := handler(stmt); err != nil {
			return err
		}
	}
}

// NewWriter creates a writer for the specified format.
func NewWriter(w io.Writer, format Format, opts ...Option) (Writer, error) {
	options := defaultOptions()
	for _, opt := range opts {
		opt(&options)
	}

	return newEncoder(w, format, options)
}

// Option helpers

// OptContext sets the context for cancellation and timeouts.
func OptContext(ctx context.Context) Option {
	return func(opts *Options) {
		opts.Context = ctx
	}
}

// OptMaxLineBytes sets the maximum line size limit.
func OptMaxLineBytes(maxBytes int) Option {
	return func(opts *Options) {
		opts.MaxLineBytes = maxBytes
	}
}

// OptMaxStatementBytes sets the maximum statement size limit.
func OptMaxStatementBytes(maxBytes int) Option {
	return func(opts *Options) {
		opts.MaxStatementBytes = maxBytes
	}
}

// OptMaxDepth sets the maximum nesting depth limit.
func OptMaxDepth(maxDepth int) Option {
	return func(opts *Options) {
		opts.MaxDepth = maxDepth
	}
}

// OptMaxTriples sets the maximum number of triples/quads to process.
func OptMaxTriples(maxTriples int64) Option {
	return func(opts *Options) {
		opts.MaxTriples = maxTriples
	}
}

// OptSafeLimits applies safe limits suitable for untrusted input.
func OptSafeLimits() Option {
	return func(opts *Options) {
		safe := safeOptions()
		opts.MaxLineBytes = safe.MaxLineBytes
		opts.MaxStatementBytes = safe.MaxStatementBytes
		opts.MaxDepth = safe.MaxDepth
		opts.MaxTriples = safe.MaxTriples
	}
}

// OptStrictIRIValidation enables strict IRI validation according to RFC 3987.
// When enabled, all IRIs are validated for correct syntax during parsing.
// Default is lenient (no validation) for backward compatibility.
//
// Note: This option affects parser behavior. Invalid IRIs will cause parse errors
// when this option is enabled.
func OptStrictIRIValidation() Option {
	return func(opts *Options) {
		opts.StrictIRIValidation = true
	}
}

// OptExpandRDFXMLContainers enables RDF/XML container membership expansion.
// When enabled (default), container elements (rdf:Bag, rdf:Seq, rdf:Alt) automatically
// generate container membership properties (rdf:_1, rdf:_2, etc.) from rdf:li elements.
//
// Container expansion is enabled by default. Use OptDisableRDFXMLContainerExpansion()
// to disable it if you want to preserve the original container structure.
func OptExpandRDFXMLContainers() Option {
	return func(opts *Options) {
		opts.ExpandRDFXMLContainers = true
	}
}

// OptDisableRDFXMLContainerExpansion disables RDF/XML container membership expansion.
// When disabled, container elements are parsed as regular node elements without
// automatically generating container membership properties.
func OptDisableRDFXMLContainerExpansion() Option {
	return func(opts *Options) {
		opts.ExpandRDFXMLContainers = false
	}
}

// Internal helpers

func defaultOptions() Options {
	return Options{
		MaxLineBytes:           DefaultMaxLineBytes,
		MaxStatementBytes:      DefaultMaxStatementBytes,
		MaxDepth:               DefaultMaxDepth,
		MaxTriples:             DefaultMaxTriples,
		ExpandRDFXMLContainers: true, // Default: enable container expansion
	}
}

func safeOptions() Options {
	safe := safeDecodeOptions()
	return Options{
		MaxLineBytes:      safe.MaxLineBytes,
		MaxStatementBytes: safe.MaxStatementBytes,
		MaxDepth:          safe.MaxDepth,
		MaxTriples:        safe.MaxTriples,
	}
}

// detectFormat attempts to detect the format from the reader.
// It reads a sample and returns both the detected format and a reader that includes
// the buffered bytes so the decoder can read from the beginning.
func detectFormat(r io.Reader) (Format, io.Reader, bool) {
	const formatDetectionBufferSize = 512
	buf := make([]byte, formatDetectionBufferSize)
	n, err := r.Read(buf)
	if err != nil && err != io.EOF {
		return FormatAuto, r, false
	}

	sample := buf[:n]

	// Try quad formats first
	if quadFormat, ok := detectQuadFormat(bytes.NewReader(sample)); ok {
		// Combine buffered bytes with remaining reader
		return quadFormat, io.MultiReader(bytes.NewReader(sample), r), true
	}

	// Try triple formats
	if tripleFormat, ok := detectFormatFromSample(bytes.NewReader(sample)); ok {
		// Combine buffered bytes with remaining reader
		return tripleFormat, io.MultiReader(bytes.NewReader(sample), r), true
	}

	return FormatAuto, io.MultiReader(bytes.NewReader(sample), r), false
}

// newDecoder creates a reader for the specified format.
func newDecoder(r io.Reader, format Format, opts Options) (Reader, error) {
	// Convert Options to decodeOptions for internal use
	decodeOpts := decodeOptions{
		Context:                    opts.Context,
		MaxLineBytes:               opts.MaxLineBytes,
		MaxStatementBytes:          opts.MaxStatementBytes,
		MaxDepth:                   opts.MaxDepth,
		MaxTriples:                 opts.MaxTriples,
		AllowQuotedTripleStatement: opts.AllowQuotedTripleStatement,
		DebugStatements:            opts.DebugStatements,
		StrictIRIValidation:        opts.StrictIRIValidation,
		ExpandRDFXMLContainers:     opts.ExpandRDFXMLContainers,
	}

	switch format {
	case FormatTurtle:
		dec, err := newTripleDecoderWithOptions(r, "turtle", decodeOpts)
		if err != nil {
			return nil, err
		}
		return &quadReaderAdapter{dec: dec, isTriple: true}, nil
	case FormatNTriples:
		dec, err := newTripleDecoderWithOptions(r, "ntriples", decodeOpts)
		if err != nil {
			return nil, err
		}
		return &quadReaderAdapter{dec: dec, isTriple: true}, nil
	case FormatRDFXML:
		dec, err := newTripleDecoderWithOptions(r, "rdfxml", decodeOpts)
		if err != nil {
			return nil, err
		}
		return &quadReaderAdapter{dec: dec, isTriple: true}, nil
	case FormatJSONLD:
		dec, err := newTripleDecoderWithOptions(r, "jsonld", decodeOpts)
		if err != nil {
			return nil, err
		}
		return &quadReaderAdapter{dec: dec, isTriple: true}, nil
	case FormatTriG:
		dec, err := newQuadDecoderWithOptions(r, "trig", decodeOpts)
		if err != nil {
			return nil, err
		}
		return &quadReaderAdapter{dec: dec, isTriple: false}, nil
	case FormatNQuads:
		dec, err := newQuadDecoderWithOptions(r, "nquads", decodeOpts)
		if err != nil {
			return nil, err
		}
		return &quadReaderAdapter{dec: dec, isTriple: false}, nil
	default:
		return nil, ErrUnsupportedFormat
	}
}

// newEncoder creates a writer for the specified format.
func newEncoder(w io.Writer, format Format, opts Options) (Writer, error) {
	switch format {
	case FormatTurtle:
		enc, err := newTripleEncoder(w, "turtle")
		if err != nil {
			return nil, err
		}
		return &quadWriterAdapter{enc: enc, isTriple: true}, nil
	case FormatNTriples:
		enc, err := newTripleEncoder(w, "ntriples")
		if err != nil {
			return nil, err
		}
		return &quadWriterAdapter{enc: enc, isTriple: true}, nil
	case FormatRDFXML:
		enc, err := newTripleEncoder(w, "rdfxml")
		if err != nil {
			return nil, err
		}
		return &quadWriterAdapter{enc: enc, isTriple: true}, nil
	case FormatJSONLD:
		enc, err := newTripleEncoder(w, "jsonld")
		if err != nil {
			return nil, err
		}
		return &quadWriterAdapter{enc: enc, isTriple: true}, nil
	case FormatTriG:
		enc, err := newQuadEncoder(w, "trig")
		if err != nil {
			return nil, err
		}
		return &quadWriterAdapter{enc: enc, isTriple: false}, nil
	case FormatNQuads:
		enc, err := newQuadEncoder(w, "nquads")
		if err != nil {
			return nil, err
		}
		return &quadWriterAdapter{enc: enc, isTriple: false}, nil
	default:
		return nil, ErrUnsupportedFormat
	}
}

// quadReaderAdapter adapts TripleDecoder/QuadDecoder to unified Reader interface.
type quadReaderAdapter struct {
	dec      interface{}
	isTriple bool
}

func (a *quadReaderAdapter) Next() (Statement, error) {
	if a.isTriple {
		dec := a.dec.(tripleDecoder)
		triple, err := dec.Next()
		if err != nil {
			return Statement{}, err
		}
		return Statement{S: triple.S, P: triple.P, O: triple.O, G: nil}, nil
	} else {
		dec := a.dec.(quadDecoder)
		quad, err := dec.Next()
		if err != nil {
			return Statement{}, err
		}
		return quad.ToStatement(), nil
	}
}

func (a *quadReaderAdapter) Close() error {
	if a.isTriple {
		return a.dec.(tripleDecoder).Close()
	}
	return a.dec.(quadDecoder).Close()
}

// quadWriterAdapter adapts TripleEncoder/QuadEncoder to unified Writer interface.
type quadWriterAdapter struct {
	enc      interface{}
	isTriple bool
}

func (a *quadWriterAdapter) Write(s Statement) error {
	if a.isTriple {
		enc := a.enc.(tripleEncoder)
		return enc.Write(s.AsTriple())
	} else {
		enc := a.enc.(quadEncoder)
		return enc.Write(s.AsQuad())
	}
}

func (a *quadWriterAdapter) Flush() error {
	if a.isTriple {
		return a.enc.(tripleEncoder).Flush()
	}
	return a.enc.(quadEncoder).Flush()
}

func (a *quadWriterAdapter) Close() error {
	if a.isTriple {
		return a.enc.(tripleEncoder).Close()
	}
	return a.enc.(quadEncoder).Close()
}
