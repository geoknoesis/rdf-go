package rdf

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// AnyFormat describes a parse/serialize format.
type AnyFormat struct {
	Name      string
	Kind      FormatKind
	TripleFmt TripleFormat
	QuadFmt   QuadFormat
	IsJSONLD  bool
}

// FormatKind describes whether a format is triple or quad oriented.
type FormatKind int

const (
	FormatUnknown FormatKind = iota
	FormatTriples
	FormatQuads
)

// AnyFormatOptions provides per-format options.
// Extend this struct as other formats gain configurable options.
type AnyFormatOptions struct {
	JSONLD *JSONLDOptions
	Turtle *TurtleEncodeOptions
	TriG   *TriGEncodeOptions
	RDFXML *RDFXMLEncodeOptions
}

// ResolveAnyFormat resolves a canonical format name into its configuration.
func ResolveAnyFormat(name string) (AnyFormat, error) {
	switch name {
	case "turtle":
		return AnyFormat{Name: name, Kind: FormatTriples, TripleFmt: TripleFormatTurtle}, nil
	case "ntriples":
		return AnyFormat{Name: name, Kind: FormatTriples, TripleFmt: TripleFormatNTriples}, nil
	case "rdfxml":
		return AnyFormat{Name: name, Kind: FormatTriples, TripleFmt: TripleFormatRDFXML}, nil
	case "jsonld":
		return AnyFormat{Name: name, Kind: FormatTriples, TripleFmt: TripleFormatJSONLD, IsJSONLD: true}, nil
	case "trig":
		return AnyFormat{Name: name, Kind: FormatQuads, QuadFmt: QuadFormatTriG}, nil
	case "nquads":
		return AnyFormat{Name: name, Kind: FormatQuads, QuadFmt: QuadFormatNQuads}, nil
	default:
		return AnyFormat{}, fmt.Errorf("unknown format: %s", name)
	}
}

// ResolveAnyFormatFromPath infers format from filename extension.
func ResolveAnyFormatFromPath(path string) (AnyFormat, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".ttl":
		return ResolveAnyFormat("turtle")
	case ".nt":
		return ResolveAnyFormat("ntriples")
	case ".trig":
		return ResolveAnyFormat("trig")
	case ".nq":
		return ResolveAnyFormat("nquads")
	case ".rdf", ".xml":
		return ResolveAnyFormat("rdfxml")
	case ".jsonld", ".json":
		return ResolveAnyFormat("jsonld")
	default:
		return AnyFormat{}, fmt.Errorf("unknown format for path: %s", path)
	}
}

// ResolveAnyFormatFromContentType infers format from a content type.
func ResolveAnyFormatFromContentType(contentType string) (AnyFormat, error) {
	mediaType := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	switch mediaType {
	case "text/turtle":
		return ResolveAnyFormat("turtle")
	case "application/n-triples":
		return ResolveAnyFormat("ntriples")
	case "application/trig":
		return ResolveAnyFormat("trig")
	case "application/n-quads":
		return ResolveAnyFormat("nquads")
	case "application/rdf+xml", "application/xml", "text/xml":
		return ResolveAnyFormat("rdfxml")
	case "application/ld+json":
		return ResolveAnyFormat("jsonld")
	default:
		return AnyFormat{}, fmt.Errorf("unknown content type: %s", contentType)
	}
}

// ParseAnyAuto parses input using inferred format from path or content type.
func ParseAnyAuto(ctx context.Context, r io.Reader, path string, contentType string, opts AnyFormatOptions) ([]Quad, error) {
	if path != "" {
		if format, err := ResolveAnyFormatFromPath(path); err == nil {
			return ParseAnyWithFormat(ctx, r, format, opts)
		}
	}
	if contentType != "" {
		format, err := ResolveAnyFormatFromContentType(contentType)
		if err != nil {
			return nil, err
		}
		return ParseAnyWithFormat(ctx, r, format, opts)
	}
	return nil, fmt.Errorf("unable to infer format")
}

// SerializeAnyAuto writes quads using inferred format from path or content type.
func SerializeAnyAuto(ctx context.Context, w io.Writer, path string, contentType string, quads []Quad, opts AnyFormatOptions) error {
	if path != "" {
		if format, err := ResolveAnyFormatFromPath(path); err == nil {
			return SerializeAnyWithFormat(ctx, w, format, quads, opts)
		}
	}
	if contentType != "" {
		format, err := ResolveAnyFormatFromContentType(contentType)
		if err != nil {
			return err
		}
		return SerializeAnyWithFormat(ctx, w, format, quads, opts)
	}
	return fmt.Errorf("unable to infer format")
}

// ParseAnyWithFormat parses input into quads for the given format.
func ParseAnyWithFormat(ctx context.Context, r io.Reader, format AnyFormat, opts AnyFormatOptions) ([]Quad, error) {
	return ParseAny(ctx, r, format.Name, opts)
}

// SerializeAnyWithFormat writes quads for the given format.
func SerializeAnyWithFormat(ctx context.Context, w io.Writer, format AnyFormat, quads []Quad, opts AnyFormatOptions) error {
	return SerializeAny(ctx, w, format.Name, quads, opts)
}

// ParseAny parses input into quads for the given format, using per-format options.
func ParseAny(ctx context.Context, r io.Reader, formatName string, opts AnyFormatOptions) ([]Quad, error) {
	format, err := ResolveAnyFormat(formatName)
	if err != nil {
		return nil, err
	}
	if format.IsJSONLD {
		jsonldOpts := JSONLDOptions{}
		if opts.JSONLD != nil {
			jsonldOpts = *opts.JSONLD
		}
		var quads []Quad
		if err := ParseJSONLDQuads(ctx, r, jsonldOpts, QuadHandlerFunc(func(q Quad) error {
			quads = append(quads, q)
			return nil
		})); err != nil {
			return nil, err
		}
		return quads, nil
	}
	switch format.Kind {
	case FormatTriples:
		var quads []Quad
		if err := ParseTriples(ctx, r, format.TripleFmt, TripleHandlerFunc(func(t Triple) error {
			quads = append(quads, Quad{S: t.S, P: t.P, O: t.O})
			return nil
		})); err != nil {
			return nil, err
		}
		return quads, nil
	case FormatQuads:
		var quads []Quad
		if err := ParseQuads(ctx, r, format.QuadFmt, QuadHandlerFunc(func(q Quad) error {
			quads = append(quads, q)
			return nil
		})); err != nil {
			return nil, err
		}
		return quads, nil
	default:
		return nil, fmt.Errorf("unsupported format kind")
	}
}

// SerializeAny writes quads to the given writer using the selected format.
func SerializeAny(ctx context.Context, w io.Writer, formatName string, quads []Quad, opts AnyFormatOptions) error {
	_ = ctx
	format, err := ResolveAnyFormat(formatName)
	if err != nil {
		return err
	}
	if format.IsJSONLD {
		jsonldOpts := JSONLDOptions{}
		if opts.JSONLD != nil {
			jsonldOpts = *opts.JSONLD
		}
		if hasNamedGraphs(quads) {
			enc := NewJSONLDQuadEncoder(w, jsonldOpts)
			defer enc.Close()
			for _, q := range quads {
				if err := enc.Write(q); err != nil {
					return err
				}
			}
			return nil
		}
		enc := NewJSONLDTripleEncoder(w, jsonldOpts)
		defer enc.Close()
		for _, q := range quads {
			if err := enc.Write(Triple{S: q.S, P: q.P, O: q.O}); err != nil {
				return err
			}
		}
		return nil
	}
	switch format.Kind {
	case FormatTriples:
		if hasNamedGraphs(quads) {
			return fmt.Errorf("format %s does not support named graphs", format.Name)
		}
		var enc TripleEncoder
		var err error
		switch format.TripleFmt {
		case TripleFormatTurtle:
			if opts.Turtle != nil {
				enc = NewTurtleTripleEncoder(w, *opts.Turtle)
			} else {
				enc, err = NewTripleEncoder(w, format.TripleFmt)
			}
		case TripleFormatRDFXML:
			if opts.RDFXML != nil {
				enc = NewRDFXMLTripleEncoder(w, *opts.RDFXML)
			} else {
				enc, err = NewTripleEncoder(w, format.TripleFmt)
			}
		default:
			enc, err = NewTripleEncoder(w, format.TripleFmt)
		}
		if err != nil {
			return err
		}
		defer enc.Close()
		for _, q := range quads {
			if err := enc.Write(Triple{S: q.S, P: q.P, O: q.O}); err != nil {
				return err
			}
		}
		return nil
	case FormatQuads:
		var enc QuadEncoder
		var err error
		switch format.QuadFmt {
		case QuadFormatTriG:
			if opts.TriG != nil {
				enc = NewTriGQuadEncoder(w, *opts.TriG)
			} else {
				enc, err = NewQuadEncoder(w, format.QuadFmt)
			}
		default:
			enc, err = NewQuadEncoder(w, format.QuadFmt)
		}
		if err != nil {
			return err
		}
		defer enc.Close()
		for _, q := range quads {
			if err := enc.Write(q); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported format kind")
	}
}

func hasNamedGraphs(quads []Quad) bool {
	for _, q := range quads {
		if q.G != nil {
			return true
		}
	}
	return false
}
