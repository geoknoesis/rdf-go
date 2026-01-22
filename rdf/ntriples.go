package rdf

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

type ntDecoder struct {
	reader *bufio.Reader
	err    error
	format Format
}

func newNTriplesDecoder(r io.Reader) Decoder {
	return &ntDecoder{reader: bufio.NewReader(r), format: FormatNTriples}
}

func newNQuadsDecoder(r io.Reader) Decoder {
	return &ntDecoder{reader: bufio.NewReader(r), format: FormatNQuads}
}

func (d *ntDecoder) Next() (Quad, error) {
	for {
		line, err := d.readLine()
		if err != nil {
			if err == io.EOF {
				return Quad{}, io.EOF
			}
			d.err = err
			return Quad{}, err
		}
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		quad, err := parseNTLine(line, d.format)
		if err != nil {
			d.err = err
			return Quad{}, err
		}
		return quad, nil
	}
}

func (d *ntDecoder) Err() error { return d.err }
func (d *ntDecoder) Close() error {
	return nil
}

func (d *ntDecoder) readLine() (string, error) {
	line, err := d.reader.ReadString('\n')
	if err != nil {
		if err == io.EOF && len(line) > 0 {
			return line, nil
		}
		return "", err
	}
	return line, nil
}

func parseNTLine(line string, format Format) (Quad, error) {
	cursor := &ntCursor{input: line}
	subject, err := cursor.parseSubject()
	if err != nil {
		return Quad{}, err
	}
	predicate, err := cursor.parseIRI()
	if err != nil {
		return Quad{}, err
	}
	object, err := cursor.parseObject()
	if err != nil {
		return Quad{}, err
	}

	var graph Term
	if format == FormatNQuads {
		graph = cursor.parseOptionalTerm()
	}
	cursor.skipWS()
	if !cursor.consume('.') {
		return Quad{}, cursor.errorf("expected '.' at end of statement")
	}
	if format == FormatNTriples && graph != nil {
		return Quad{}, cursor.errorf("graph term not allowed in N-Triples")
	}

	return Quad{S: subject, P: predicate, O: object, G: graph}, nil
}

type ntCursor struct {
	input string
	pos   int
}

func (c *ntCursor) skipWS() {
	for c.pos < len(c.input) {
		switch c.input[c.pos] {
		case ' ', '\t', '\r', '\n':
			c.pos++
		default:
			return
		}
	}
}

func (c *ntCursor) consume(ch byte) bool {
	c.skipWS()
	if c.pos < len(c.input) && c.input[c.pos] == ch {
		c.pos++
		return true
	}
	return false
}

func (c *ntCursor) parseSubject() (Term, error) {
	c.skipWS()
	term, err := c.parseTerm(false)
	if err != nil {
		return nil, err
	}
	return term, nil
}

func (c *ntCursor) parseObject() (Term, error) {
	c.skipWS()
	return c.parseTerm(true)
}

func (c *ntCursor) parseOptionalTerm() Term {
	c.skipWS()
	if c.pos >= len(c.input) {
		return nil
	}
	if c.input[c.pos] == '.' {
		return nil
	}
	term, _ := c.parseTerm(false)
	return term
}

func (c *ntCursor) parseTerm(allowLiteral bool) (Term, error) {
	c.skipWS()
	if c.pos >= len(c.input) {
		return nil, c.errorf("unexpected end of line")
	}
	switch {
	case strings.HasPrefix(c.input[c.pos:], "<<"):
		return c.parseTripleTerm()
	case c.input[c.pos] == '<':
		iri, err := c.parseIRI()
		return iri, err
	case strings.HasPrefix(c.input[c.pos:], "_:"):
		return c.parseBlankNode()
	case c.input[c.pos] == '"':
		if !allowLiteral {
			return nil, c.errorf("literal not allowed here")
		}
		return c.parseLiteral()
	default:
		return nil, c.errorf("unexpected token")
	}
}

func (c *ntCursor) parseIRI() (IRI, error) {
	c.skipWS()
	if !c.consume('<') {
		return IRI{}, c.errorf("expected IRI")
	}
	start := c.pos
	for c.pos < len(c.input) && c.input[c.pos] != '>' {
		c.pos++
	}
	if c.pos >= len(c.input) {
		return IRI{}, c.errorf("unterminated IRI")
	}
	value := c.input[start:c.pos]
	c.pos++
	return IRI{Value: value}, nil
}

func (c *ntCursor) parseBlankNode() (BlankNode, error) {
	c.skipWS()
	if !strings.HasPrefix(c.input[c.pos:], "_:") {
		return BlankNode{}, c.errorf("expected blank node")
	}
	c.pos += 2
	start := c.pos
	for c.pos < len(c.input) && !isTermDelimiter(c.input[c.pos]) {
		c.pos++
	}
	if start == c.pos {
		return BlankNode{}, c.errorf("blank node id missing")
	}
	return BlankNode{ID: c.input[start:c.pos]}, nil
}

func (c *ntCursor) parseLiteral() (Literal, error) {
	c.skipWS()
	if !c.consume('"') {
		return Literal{}, c.errorf("expected literal")
	}
	var builder strings.Builder
	for c.pos < len(c.input) {
		ch := c.input[c.pos]
		if ch == '"' {
			c.pos++
			break
		}
		if ch == '\\' {
			if c.pos+1 >= len(c.input) {
				return Literal{}, c.errorf("unterminated escape")
			}
			next := c.input[c.pos+1]
			switch next {
			case 'n':
				builder.WriteByte('\n')
			case 't':
				builder.WriteByte('\t')
			case 'r':
				builder.WriteByte('\r')
			case '"':
				builder.WriteByte('"')
			case '\\':
				builder.WriteByte('\\')
			default:
				builder.WriteByte(next)
			}
			c.pos += 2
			continue
		}
		builder.WriteByte(ch)
		c.pos++
	}
	lexical := builder.String()
	c.skipWS()
	if strings.HasPrefix(c.input[c.pos:], "@") {
		c.pos++
		start := c.pos
		for c.pos < len(c.input) && !isTermDelimiter(c.input[c.pos]) {
			c.pos++
		}
		return Literal{Lexical: lexical, Lang: c.input[start:c.pos]}, nil
	}
	if strings.HasPrefix(c.input[c.pos:], "^^") {
		c.pos += 2
		dt, err := c.parseIRI()
		if err != nil {
			return Literal{}, err
		}
		return Literal{Lexical: lexical, Datatype: dt}, nil
	}
	return Literal{Lexical: lexical}, nil
}

func (c *ntCursor) parseTripleTerm() (Term, error) {
	if !strings.HasPrefix(c.input[c.pos:], "<<") {
		return nil, c.errorf("expected '<<'")
	}
	c.pos += 2
	subject, err := c.parseSubject()
	if err != nil {
		return nil, err
	}
	predicate, err := c.parseIRI()
	if err != nil {
		return nil, err
	}
	object, err := c.parseObject()
	if err != nil {
		return nil, err
	}
	c.skipWS()
	if !strings.HasPrefix(c.input[c.pos:], ">>") {
		return nil, c.errorf("expected '>>'")
	}
	c.pos += 2
	return TripleTerm{S: subject, P: predicate, O: object}, nil
}

func (c *ntCursor) errorf(format string, args ...interface{}) error {
	return fmt.Errorf("ntriples: "+format, args...)
}

func isTermDelimiter(ch byte) bool {
	switch ch {
	case ' ', '\t', '\r', '\n', '.':
		return true
	default:
		return false
	}
}

type ntEncoder struct {
	writer *bufio.Writer
	format Format
	err    error
}

func newNTriplesEncoder(w io.Writer) Encoder {
	return &ntEncoder{writer: bufio.NewWriter(w), format: FormatNTriples}
}

func newNQuadsEncoder(w io.Writer) Encoder {
	return &ntEncoder{writer: bufio.NewWriter(w), format: FormatNQuads}
}

func (e *ntEncoder) Write(q Quad) error {
	if e.err != nil {
		return e.err
	}
	if q.IsZero() {
		return fmt.Errorf("ntriples: empty statement")
	}
	if q.S == nil || q.P.Value == "" || q.O == nil {
		return fmt.Errorf("ntriples: missing statement fields")
	}
	line := renderTerm(q.S) + " " + renderIRI(q.P) + " " + renderTerm(q.O)
	if e.format == FormatNQuads && q.G != nil {
		line += " " + renderTerm(q.G)
	}
	line += " .\n"
	_, err := e.writer.WriteString(line)
	if err != nil {
		e.err = err
	}
	return err
}

func (e *ntEncoder) Flush() error {
	if e.err != nil {
		return e.err
	}
	return e.writer.Flush()
}

func (e *ntEncoder) Close() error {
	return e.Flush()
}

func renderIRI(iri IRI) string {
	return "<" + iri.Value + ">"
}

func renderTerm(term Term) string {
	switch value := term.(type) {
	case IRI:
		return renderIRI(value)
	case BlankNode:
		return value.String()
	case Literal:
		if value.Lang != "" {
			return fmt.Sprintf("%q@%s", value.Lexical, value.Lang)
		}
		if value.Datatype.Value != "" {
			return fmt.Sprintf("%q^^%s", value.Lexical, renderIRI(value.Datatype))
		}
		return fmt.Sprintf("%q", value.Lexical)
	case TripleTerm:
		return value.String()
	default:
		return ""
	}
}
