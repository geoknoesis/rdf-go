package rdf

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strings"
)

// TurtleEncodeOptions configures Turtle encoding.
type TurtleEncodeOptions struct {
	Pretty   bool
	Indent   string
	Prefixes map[string]string
	BaseIRI  string
}

// TriGEncodeOptions configures TriG encoding.
type TriGEncodeOptions struct {
	Pretty   bool
	Indent   string
	Prefixes map[string]string
	BaseIRI  string
}

// Triple encoder for Turtle
type turtleTripleEncoder struct {
	writer  *bufio.Writer
	err     error
	started bool
	opts    TurtleEncodeOptions
}

func newTurtleTripleEncoder(w io.Writer) TripleEncoder {
	return newTurtleTripleEncoderWithOptions(w, TurtleEncodeOptions{})
}

func newTurtleTripleEncoderWithOptions(w io.Writer, opts TurtleEncodeOptions) TripleEncoder {
	return &turtleTripleEncoder{writer: bufio.NewWriter(w), opts: opts}
}

// NewTurtleTripleEncoder creates a Turtle triple encoder with options.
func NewTurtleTripleEncoder(w io.Writer, opts TurtleEncodeOptions) TripleEncoder {
	return newTurtleTripleEncoderWithOptions(w, opts)
}

func (e *turtleTripleEncoder) Write(t Triple) error {
	if e.err != nil {
		return e.err
	}
	if !e.started {
		if err := e.writeHeader(); err != nil {
			return err
		}
	}
	if t.S == nil || t.P.Value == "" || t.O == nil {
		return fmt.Errorf("turtle: missing statement fields")
	}
	line := renderTermWithPrefixes(t.S, e.opts.Prefixes) + " " + renderIRIWithPrefixes(t.P, e.opts.Prefixes) + " " + renderTermWithPrefixes(t.O, e.opts.Prefixes) + " .\n"
	if e.opts.Indent != "" {
		line = e.opts.Indent + line
	}
	_, err := e.writer.WriteString(line)
	if err != nil {
		e.err = err
	}
	return err
}

func (e *turtleTripleEncoder) Flush() error {
	if e.err != nil {
		return e.err
	}
	return e.writer.Flush()
}

func (e *turtleTripleEncoder) Close() error {
	if e.err != nil {
		return e.err
	}
	if err := e.writer.Flush(); err != nil {
		e.err = err
		return err
	}
	e.err = fmt.Errorf("turtle: writer closed")
	return nil
}

func (e *turtleTripleEncoder) writeHeader() error {
	e.started = true
	if e.opts.BaseIRI != "" {
		if _, err := e.writer.WriteString("@base <" + e.opts.BaseIRI + "> .\n"); err != nil {
			e.err = err
			return err
		}
	}
	if len(e.opts.Prefixes) == 0 {
		return nil
	}
	for _, prefix := range sortedPrefixKeys(e.opts.Prefixes) {
		ns := e.opts.Prefixes[prefix]
		label := prefix + ":"
		if prefix == "" {
			label = ":"
		}
		line := "@prefix " + label + " <" + ns + "> .\n"
		if _, err := e.writer.WriteString(line); err != nil {
			e.err = err
			return err
		}
	}
	return nil
}

// Quad encoder for TriG
type trigQuadEncoder struct {
	writer  *bufio.Writer
	err     error
	started bool
	opts    TriGEncodeOptions
}

func newTriGQuadEncoder(w io.Writer) QuadEncoder {
	return newTriGQuadEncoderWithOptions(w, TriGEncodeOptions{})
}

func newTriGQuadEncoderWithOptions(w io.Writer, opts TriGEncodeOptions) QuadEncoder {
	return &trigQuadEncoder{writer: bufio.NewWriter(w), opts: opts}
}

// NewTriGQuadEncoder creates a TriG quad encoder with options.
func NewTriGQuadEncoder(w io.Writer, opts TriGEncodeOptions) QuadEncoder {
	return newTriGQuadEncoderWithOptions(w, opts)
}

func (e *trigQuadEncoder) Write(q Quad) error {
	if e.err != nil {
		return e.err
	}
	if !e.started {
		if err := e.writeHeader(); err != nil {
			return err
		}
	}
	if q.S == nil || q.P.Value == "" || q.O == nil {
		return fmt.Errorf("trig: missing statement fields")
	}
	subject := renderTermWithPrefixes(q.S, e.opts.Prefixes)
	predicate := renderIRIWithPrefixes(q.P, e.opts.Prefixes)
	object := renderTermWithPrefixes(q.O, e.opts.Prefixes)
	line := subject + " " + predicate + " " + object + " ."
	indent := e.opts.Indent
	if e.opts.Pretty && indent == "" {
		indent = "  "
	}
	if q.G != nil && e.opts.Pretty {
		graph := renderTermWithPrefixes(q.G, e.opts.Prefixes)
		if _, err := e.writer.WriteString(graph + " {\n"); err != nil {
			e.err = err
			return err
		}
		if _, err := e.writer.WriteString(indent + line + "\n"); err != nil {
			e.err = err
			return err
		}
		_, err := e.writer.WriteString("}\n")
		if err != nil {
			e.err = err
		}
		return err
	}
	if q.G != nil {
		graph := renderTermWithPrefixes(q.G, e.opts.Prefixes)
		line = graph + " { " + line + " }"
	}
	if e.opts.Indent != "" {
		line = e.opts.Indent + line
	}
	_, err := e.writer.WriteString(line + "\n")
	if err != nil {
		e.err = err
	}
	return err
}

func (e *trigQuadEncoder) Flush() error {
	if e.err != nil {
		return e.err
	}
	return e.writer.Flush()
}

func (e *trigQuadEncoder) Close() error {
	if e.err != nil {
		return e.err
	}
	if err := e.writer.Flush(); err != nil {
		e.err = err
		return err
	}
	e.err = fmt.Errorf("trig: writer closed")
	return nil
}

func (e *trigQuadEncoder) writeHeader() error {
	e.started = true
	if e.opts.BaseIRI != "" {
		if _, err := e.writer.WriteString("@base <" + e.opts.BaseIRI + "> .\n"); err != nil {
			e.err = err
			return err
		}
	}
	if len(e.opts.Prefixes) == 0 {
		return nil
	}
	for _, prefix := range sortedPrefixKeys(e.opts.Prefixes) {
		ns := e.opts.Prefixes[prefix]
		label := prefix + ":"
		if prefix == "" {
			label = ":"
		}
		line := "@prefix " + label + " <" + ns + "> .\n"
		if _, err := e.writer.WriteString(line); err != nil {
			e.err = err
			return err
		}
	}
	return nil
}

func sortedPrefixKeys(prefixes map[string]string) []string {
	keys := make([]string, 0, len(prefixes))
	for key := range prefixes {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func renderIRIWithPrefixes(iri IRI, prefixes map[string]string) string {
	if qname, ok := abbreviateQName(iri.Value, prefixes, true); ok {
		return qname
	}
	return renderIRI(iri)
}

func renderTermWithPrefixes(term Term, prefixes map[string]string) string {
	switch value := term.(type) {
	case IRI:
		return renderIRIWithPrefixes(value, prefixes)
	case BlankNode:
		return value.String()
	case Literal:
		if value.Lang != "" {
			return fmt.Sprintf("%q@%s", value.Lexical, value.Lang)
		}
		if value.Datatype.Value != "" {
			return fmt.Sprintf("%q^^%s", value.Lexical, renderIRIWithPrefixes(value.Datatype, prefixes))
		}
		return fmt.Sprintf("%q", value.Lexical)
	case TripleTerm:
		return value.String()
	default:
		return ""
	}
}

func abbreviateQName(iri string, prefixes map[string]string, allowEmptyPrefix bool) (string, bool) {
	if len(prefixes) == 0 {
		return "", false
	}
	bestNS := ""
	bestPrefix := ""
	found := false
	for prefix, ns := range prefixes {
		if prefix == "" && !allowEmptyPrefix {
			continue
		}
		if !strings.HasPrefix(iri, ns) {
			continue
		}
		local := iri[len(ns):]
		if !isQNameLocal(local) {
			continue
		}
		if len(ns) > len(bestNS) {
			bestNS = ns
			bestPrefix = prefix
			found = true
		}
	}
	if !found {
		return "", false
	}
	local := iri[len(bestNS):]
	if bestPrefix == "" {
		return ":" + local, true
	}
	return bestPrefix + ":" + local, true
}
