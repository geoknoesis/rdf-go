package rdf

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// RDFXMLEncodeOptions configures RDF/XML encoding.
type RDFXMLEncodeOptions struct {
	Pretty   bool
	Indent   string
	Prefixes map[string]string
	BaseIRI  string
}

// Triple encoder for RDF/XML
type rdfxmltripleEncoder struct {
	writer       *bufio.Writer
	started      bool
	closed       bool
	err          error
	opts         RDFXMLEncodeOptions
	indent       string
	prefixes     map[string]string
	rootPrefixes map[string]string
	nsToPref     map[string]string
	autoSeq      int
}

func newRDFXMLtripleEncoder(w io.Writer) tripleEncoder {
	return newRDFXMLtripleEncoderWithOptions(w, RDFXMLEncodeOptions{})
}

func newRDFXMLtripleEncoderWithOptions(w io.Writer, opts RDFXMLEncodeOptions) tripleEncoder {
	indent := opts.Indent
	if opts.Pretty && indent == "" {
		indent = "  "
	}
	rootPrefixes := copyPrefixMap(opts.Prefixes)
	prefixes := copyPrefixMap(opts.Prefixes)
	nsToPref := map[string]string{}
	for prefix, ns := range prefixes {
		nsToPref[ns] = prefix
	}
	return &rdfxmltripleEncoder{
		writer:       bufio.NewWriter(w),
		opts:         opts,
		indent:       indent,
		prefixes:     prefixes,
		rootPrefixes: rootPrefixes,
		nsToPref:     nsToPref,
	}
}

func (e *rdfxmltripleEncoder) Write(t Triple) error {
	if e.err != nil {
		return e.err
	}
	if e.closed {
		return fmt.Errorf("rdfxml: writer closed")
	}
	if !e.started {
		e.started = true
		if _, err := e.writer.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n"); err != nil {
			e.err = err
			return err
		}
		root := `<rdf:RDF xmlns:rdf="` + rdfXMLNS + `"`
		if e.opts.BaseIRI != "" {
			root += ` xml:base="` + escapeXMLAttr(e.opts.BaseIRI) + `"`
		}
		for _, prefix := range sortedPrefixKeys(e.opts.Prefixes) {
			if prefix == "rdf" {
				continue
			}
			ns := e.opts.Prefixes[prefix]
			if prefix == "" {
				root += ` xmlns="` + escapeXMLAttr(ns) + `"`
				continue
			}
			root += ` xmlns:` + prefix + `="` + escapeXMLAttr(ns) + `"`
		}
		root += ">\n"
		if _, err := e.writer.WriteString(root); err != nil {
			e.err = err
			return err
		}
	}
	subjectAttrs, err := rdfxmlSubjectAttrs(t.S)
	if err != nil {
		return err
	}
	predicate, predicateNS, err := e.predicateQName(t.P.Value)
	if err != nil {
		return err
	}
	if iri, ok := t.O.(IRI); ok {
		line := fmt.Sprintf(`%s<rdf:Description %s><%s%s rdf:resource="%s"/></rdf:Description>`+"\n", e.indent, subjectAttrs, predicate, predicateNS, escapeXMLAttr(iri.Value))
		_, err := e.writer.WriteString(line)
		if err != nil {
			e.err = err
		}
		return err
	}
	if bnode, ok := t.O.(BlankNode); ok {
		line := fmt.Sprintf(`%s<rdf:Description %s><%s%s rdf:nodeID="%s"/></rdf:Description>`+"\n", e.indent, subjectAttrs, predicate, predicateNS, escapeXMLAttr(bnode.ID))
		_, err := e.writer.WriteString(line)
		if err != nil {
			e.err = err
		}
		return err
	}
	lit, ok := t.O.(Literal)
	if !ok {
		return fmt.Errorf("rdfxml: unsupported object type")
	}
	if lit.Lang != "" && lit.Datatype.Value != "" {
		return fmt.Errorf("rdfxml: literal cannot have both language and datatype")
	}
	literalAttrs := ""
	if lit.Lang != "" {
		literalAttrs = ` xml:lang="` + escapeXMLAttr(lit.Lang) + `"`
	} else if lit.Datatype.Value != "" {
		literalAttrs = ` rdf:datatype="` + escapeXMLAttr(lit.Datatype.Value) + `"`
	}
	line := fmt.Sprintf(`%s<rdf:Description %s><%s%s%s>%s</%s></rdf:Description>`+"\n", e.indent, subjectAttrs, predicate, predicateNS, literalAttrs, escapeXML(lit.Lexical), predicate)
	_, err = e.writer.WriteString(line)
	if err != nil {
		e.err = err
	}
	return err
}

func (e *rdfxmltripleEncoder) Flush() error {
	if e.err != nil {
		return e.err
	}
	if e.closed {
		return fmt.Errorf("rdfxml: writer closed")
	}
	return e.writer.Flush()
}

func (e *rdfxmltripleEncoder) Close() error {
	if e.closed {
		return e.err
	}
	e.closed = true
	if e.started {
		_, err := e.writer.WriteString(`</rdf:RDF>` + "\n")
		if err != nil {
			e.err = err
			return err
		}
		return e.writer.Flush()
	}
	return nil
}

func escapeXML(value string) string {
	replacer := strings.NewReplacer(
		`&`, "&amp;",
		`<`, "&lt;",
		`>`, "&gt;",
		`"`, "&quot;",
		`'`, "&apos;",
	)
	return replacer.Replace(value)
}

func escapeXMLAttr(value string) string {
	return escapeXML(value)
}

func copyPrefixMap(prefixes map[string]string) map[string]string {
	if len(prefixes) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(prefixes))
	for key, value := range prefixes {
		out[key] = value
	}
	return out
}

func rdfxmlSubjectAttrs(term Term) (string, error) {
	switch value := term.(type) {
	case IRI:
		return `rdf:about="` + escapeXMLAttr(value.Value) + `"`, nil
	case BlankNode:
		return `rdf:nodeID="` + escapeXMLAttr(value.ID) + `"`, nil
	default:
		return "", fmt.Errorf("rdfxml: unsupported subject type")
	}
}

func (e *rdfxmltripleEncoder) predicateQName(iri string) (string, string, error) {
	ns, local, ok := splitIRIForQName(iri)
	if !ok {
		return "", "", fmt.Errorf("rdfxml: unable to abbreviate predicate IRI %q", iri)
	}
	if prefix, ok := e.nsToPref[ns]; ok {
		if prefix == "" {
			return local, "", nil
		}
		if _, ok := e.rootPrefixes[prefix]; ok {
			return prefix + ":" + local, "", nil
		}
		return prefix + ":" + local, ` xmlns:` + prefix + `="` + escapeXMLAttr(ns) + `"`, nil
	}
	prefix := fmt.Sprintf("ns%d", e.autoSeq)
	e.autoSeq++
	e.prefixes[prefix] = ns
	e.nsToPref[ns] = prefix
	return prefix + ":" + local, ` xmlns:` + prefix + `="` + escapeXMLAttr(ns) + `"`, nil
}

func splitIRIForQName(iri string) (string, string, bool) {
	idx := strings.LastIndexAny(iri, "#/")
	if idx <= 0 || idx+1 >= len(iri) {
		return "", "", false
	}
	ns := iri[:idx+1]
	local := iri[idx+1:]
	if !isQNameLocal(local) {
		return "", "", false
	}
	return ns, local, true
}
