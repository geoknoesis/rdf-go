package rdf

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

const rdfXMLNS = "http://www.w3.org/1999/02/22-rdf-syntax-ns#"

type rdfxmlDecoder struct {
	dec   *xml.Decoder
	queue []Quad
	err   error
}

func newRDFXMLDecoder(r io.Reader) Decoder {
	return &rdfxmlDecoder{dec: xml.NewDecoder(r)}
}

func (d *rdfxmlDecoder) Next() (Quad, error) {
	for {
		if len(d.queue) > 0 {
			next := d.queue[0]
			d.queue = d.queue[1:]
			return next, nil
		}
		tok, err := d.dec.Token()
		if err != nil {
			if err == io.EOF {
				return Quad{}, io.EOF
			}
			d.err = err
			return Quad{}, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Space == rdfXMLNS && t.Name.Local == "RDF" {
				continue
			}
			if isNodeElement(t) {
				subject := subjectFromNode(t)
				if t.Name.Space != rdfXMLNS || t.Name.Local != "Description" {
					typ := IRI{Value: t.Name.Space + t.Name.Local}
					d.queue = append(d.queue, Quad{S: subject, P: IRI{Value: rdfXMLNS + "type"}, O: typ})
				}
				if err := d.readPredicateElements(subject); err != nil {
					d.err = err
					return Quad{}, err
				}
			}
		}
	}
}

func (d *rdfxmlDecoder) Err() error { return d.err }
func (d *rdfxmlDecoder) Close() error {
	return nil
}

func (d *rdfxmlDecoder) readPredicateElements(subject Term) error {
	for {
		tok, err := d.dec.Token()
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			pred := IRI{Value: t.Name.Space + t.Name.Local}
			obj, err := objectFromPredicate(d.dec, t)
			if err != nil {
				return err
			}
			d.queue = append(d.queue, Quad{S: subject, P: pred, O: obj})
		case xml.EndElement:
			return nil
		}
	}
}

func objectFromPredicate(dec *xml.Decoder, start xml.StartElement) (Term, error) {
	if iri := attrValue(start.Attr, rdfXMLNS, "resource"); iri != "" {
		return IRI{Value: iri}, consumeElement(dec)
	}
	if nodeID := attrValue(start.Attr, rdfXMLNS, "nodeID"); nodeID != "" {
		return BlankNode{ID: nodeID}, consumeElement(dec)
	}
	var content strings.Builder
	for {
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.CharData:
			content.WriteString(string(t))
		case xml.EndElement:
			return Literal{Lexical: strings.TrimSpace(content.String())}, nil
		case xml.StartElement:
			return nil, fmt.Errorf("rdfxml: nested elements not supported in predicate objects")
		}
	}
}

func consumeElement(dec *xml.Decoder) error {
	for {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		if _, ok := tok.(xml.EndElement); ok {
			return nil
		}
	}
}

func isNodeElement(el xml.StartElement) bool {
	return el.Name.Space == rdfXMLNS && el.Name.Local == "Description" || el.Name.Space != rdfXMLNS
}

func subjectFromNode(el xml.StartElement) Term {
	if about := attrValue(el.Attr, rdfXMLNS, "about"); about != "" {
		return IRI{Value: about}
	}
	if id := attrValue(el.Attr, rdfXMLNS, "ID"); id != "" {
		return IRI{Value: id}
	}
	if nodeID := attrValue(el.Attr, rdfXMLNS, "nodeID"); nodeID != "" {
		return BlankNode{ID: nodeID}
	}
	return BlankNode{ID: "genid"}
}

func attrValue(attrs []xml.Attr, space, local string) string {
	for _, attr := range attrs {
		if attr.Name.Space == space && attr.Name.Local == local {
			return attr.Value
		}
	}
	return ""
}

type rdfxmlEncoder struct {
	writer  *bufio.Writer
	started bool
	closed  bool
	err     error
}

func newRDFXMLEncoder(w io.Writer) Encoder {
	return &rdfxmlEncoder{writer: bufio.NewWriter(w)}
}

func (e *rdfxmlEncoder) Write(q Quad) error {
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
		if _, err := e.writer.WriteString(`<rdf:RDF xmlns:rdf="` + rdfXMLNS + `">` + "\n"); err != nil {
			e.err = err
			return err
		}
	}
	subject := renderTerm(q.S)
	predicate := q.P.Value
	if iri, ok := q.O.(IRI); ok {
		line := fmt.Sprintf(`<rdf:Description rdf:about="%s"><%s rdf:resource="%s"/></rdf:Description>`+"\n", escapeXML(subject), predicate, escapeXML(iri.Value))
		_, err := e.writer.WriteString(line)
		if err != nil {
			e.err = err
		}
		return err
	}
	lit, ok := q.O.(Literal)
	if !ok {
		return fmt.Errorf("rdfxml: unsupported object type")
	}
	line := fmt.Sprintf(`<rdf:Description rdf:about="%s"><%s>%s</%s></rdf:Description>`+"\n", escapeXML(subject), predicate, escapeXML(lit.Lexical), predicate)
	_, err := e.writer.WriteString(line)
	if err != nil {
		e.err = err
	}
	return err
}

func (e *rdfxmlEncoder) Flush() error {
	if e.err != nil {
		return e.err
	}
	return e.writer.Flush()
}

func (e *rdfxmlEncoder) Close() error {
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
	}
	return e.Flush()
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
