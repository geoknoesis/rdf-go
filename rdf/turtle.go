package rdf

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

const rdfTypeIRI = "http://www.w3.org/1999/02/22-rdf-syntax-ns#type"

type turtleDecoder struct {
	reader   *bufio.Reader
	err      error
	prefixes map[string]string
	baseIRI  string
	graph    Term
	format   Format
}

func newTurtleDecoder(r io.Reader) Decoder {
	return &turtleDecoder{
		reader:   bufio.NewReader(r),
		prefixes: map[string]string{},
		format:   FormatTurtle,
	}
}

func newTriGDecoder(r io.Reader) Decoder {
	return &turtleDecoder{
		reader:   bufio.NewReader(r),
		prefixes: map[string]string{},
		format:   FormatTriG,
	}
}

func (d *turtleDecoder) Next() (Quad, error) {
	for {
		line, err := d.readLine()
		if err != nil {
			if err == io.EOF {
				return Quad{}, io.EOF
			}
			d.err = err
			return Quad{}, err
		}
		line = strings.TrimSpace(stripComment(line))
		if line == "" {
			continue
		}

		if d.handleDirective(line) {
			continue
		}

		if d.format == FormatTriG {
			openIdx := strings.Index(line, "{")
			closeIdx := strings.LastIndex(line, "}")
			if openIdx >= 0 && closeIdx > openIdx {
				graphToken := strings.TrimSpace(line[:openIdx])
				inner := strings.TrimSpace(line[openIdx+1 : closeIdx])
				cursor := &turtleCursor{input: graphToken, prefixes: d.prefixes, base: d.baseIRI}
				graphTerm, err := cursor.parseTerm(false)
				if err != nil {
					d.err = err
					return Quad{}, err
				}
				quad, err := d.parseTripleLine(inner)
				if err != nil {
					d.err = err
					return Quad{}, err
				}
				quad.G = graphTerm
				return quad, nil
			}
			if strings.HasSuffix(line, "{") {
				graphToken := strings.TrimSpace(strings.TrimSuffix(line, "{"))
				cursor := &turtleCursor{input: graphToken, prefixes: d.prefixes, base: d.baseIRI}
				graphTerm, err := cursor.parseTerm(false)
				if err != nil {
					d.err = err
					return Quad{}, err
				}
				d.graph = graphTerm
				continue
			}
			if line == "}" {
				d.graph = nil
				continue
			}
		}

		quad, err := d.parseTripleLine(line)
		if err != nil {
			d.err = err
			return Quad{}, err
		}
		quad.G = d.graph
		return quad, nil
	}
}

func (d *turtleDecoder) Err() error { return d.err }
func (d *turtleDecoder) Close() error {
	return nil
}

func (d *turtleDecoder) readLine() (string, error) {
	line, err := d.reader.ReadString('\n')
	if err != nil {
		if err == io.EOF && len(line) > 0 {
			return line, nil
		}
		return "", err
	}
	return line, nil
}

func (d *turtleDecoder) handleDirective(line string) bool {
	if strings.HasPrefix(strings.ToLower(line), "@prefix") || strings.HasPrefix(strings.ToLower(line), "prefix") {
		parts := strings.Fields(line)
		if len(parts) >= 3 {
			prefix := strings.TrimSuffix(parts[1], ":")
			iri := strings.Trim(parts[2], "<>")
			d.prefixes[prefix] = iri
		}
		return true
	}
	if strings.HasPrefix(strings.ToLower(line), "@base") || strings.HasPrefix(strings.ToLower(line), "base") {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			d.baseIRI = strings.Trim(parts[1], "<>")
		}
		return true
	}
	return false
}

func (d *turtleDecoder) parseTripleLine(line string) (Quad, error) {
	cursor := &turtleCursor{input: line, prefixes: d.prefixes, base: d.baseIRI}
	subject, err := cursor.parseSubject()
	if err != nil {
		return Quad{}, err
	}
	predicate, err := cursor.parsePredicate()
	if err != nil {
		return Quad{}, err
	}
	object, err := cursor.parseObject()
	if err != nil {
		return Quad{}, err
	}
	cursor.skipWS()
	if !cursor.consume('.') {
		return Quad{}, cursor.errorf("expected '.' at end of statement")
	}
	return Quad{S: subject, P: predicate, O: object}, nil
}

type turtleCursor struct {
	input    string
	pos      int
	prefixes map[string]string
	base     string
}

func (c *turtleCursor) skipWS() {
	for c.pos < len(c.input) {
		switch c.input[c.pos] {
		case ' ', '\t', '\r', '\n':
			c.pos++
		default:
			return
		}
	}
}

func (c *turtleCursor) consume(ch byte) bool {
	c.skipWS()
	if c.pos < len(c.input) && c.input[c.pos] == ch {
		c.pos++
		return true
	}
	return false
}

func (c *turtleCursor) parseSubject() (Term, error) {
	c.skipWS()
	term, err := c.parseTerm(false)
	if err != nil {
		return nil, err
	}
	return term, nil
}

func (c *turtleCursor) parsePredicate() (IRI, error) {
	c.skipWS()
	if strings.HasPrefix(c.input[c.pos:], "a") && isTermDelimiter(c.peekNext()) {
		c.pos++
		return IRI{Value: rdfTypeIRI}, nil
	}
	term, err := c.parseTerm(false)
	if err != nil {
		return IRI{}, err
	}
	if iri, ok := term.(IRI); ok {
		return iri, nil
	}
	return IRI{}, c.errorf("predicate must be IRI")
}

func (c *turtleCursor) parseObject() (Term, error) {
	c.skipWS()
	return c.parseTerm(true)
}

func (c *turtleCursor) parseTerm(allowLiteral bool) (Term, error) {
	c.skipWS()
	if c.pos >= len(c.input) {
		return nil, c.errorf("unexpected end of line")
	}
	switch {
	case strings.HasPrefix(c.input[c.pos:], "<<"):
		return c.parseTripleTerm()
	case c.input[c.pos] == '<':
		return c.parseIRI()
	case strings.HasPrefix(c.input[c.pos:], "_:"):
		return c.parseBlankNode()
	case c.input[c.pos] == '"':
		if !allowLiteral {
			return nil, c.errorf("literal not allowed here")
		}
		return c.parseLiteral()
	default:
		return c.parsePrefixedName()
	}
}

func (c *turtleCursor) parseIRI() (Term, error) {
	if !c.consume('<') {
		return nil, c.errorf("expected IRI")
	}
	start := c.pos
	for c.pos < len(c.input) && c.input[c.pos] != '>' {
		c.pos++
	}
	if c.pos >= len(c.input) {
		return nil, c.errorf("unterminated IRI")
	}
	value := c.input[start:c.pos]
	c.pos++
	if c.base != "" && !strings.Contains(value, ":") {
		value = c.base + value
	}
	return IRI{Value: value}, nil
}

func (c *turtleCursor) parsePrefixedName() (Term, error) {
	start := c.pos
	for c.pos < len(c.input) && !isTermDelimiter(c.input[c.pos]) {
		c.pos++
	}
	token := c.input[start:c.pos]
	if token == "" {
		return nil, c.errorf("expected term")
	}
	parts := strings.SplitN(token, ":", 2)
	if len(parts) != 2 {
		return nil, c.errorf("invalid token %q", token)
	}
	prefix := parts[0]
	local := parts[1]
	base, ok := c.prefixes[prefix]
	if !ok {
		return nil, c.errorf("unknown prefix %q", prefix)
	}
	return IRI{Value: base + local}, nil
}

func (c *turtleCursor) parseBlankNode() (Term, error) {
	if !strings.HasPrefix(c.input[c.pos:], "_:") {
		return nil, c.errorf("expected blank node")
	}
	c.pos += 2
	start := c.pos
	for c.pos < len(c.input) && !isTermDelimiter(c.input[c.pos]) {
		c.pos++
	}
	if start == c.pos {
		return nil, c.errorf("blank node id missing")
	}
	return BlankNode{ID: c.input[start:c.pos]}, nil
}

func (c *turtleCursor) parseLiteral() (Term, error) {
	if !c.consume('"') {
		return nil, c.errorf("expected literal")
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
				return nil, c.errorf("unterminated escape")
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
		dt, err := c.parseTerm(false)
		if err != nil {
			return nil, err
		}
		iri, ok := dt.(IRI)
		if !ok {
			return nil, c.errorf("datatype must be IRI")
		}
		return Literal{Lexical: lexical, Datatype: iri}, nil
	}
	return Literal{Lexical: lexical}, nil
}

func (c *turtleCursor) parseTripleTerm() (Term, error) {
	if !strings.HasPrefix(c.input[c.pos:], "<<") {
		return nil, c.errorf("expected '<<'")
	}
	c.pos += 2
	subject, err := c.parseSubject()
	if err != nil {
		return nil, err
	}
	predicate, err := c.parsePredicate()
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

func (c *turtleCursor) peekNext() byte {
	if c.pos+1 >= len(c.input) {
		return 0
	}
	return c.input[c.pos+1]
}

func (c *turtleCursor) errorf(format string, args ...interface{}) error {
	return fmt.Errorf("turtle: "+format, args...)
}

func stripComment(line string) string {
	if idx := strings.Index(line, "#"); idx >= 0 {
		return line[:idx]
	}
	return line
}

type turtleEncoder struct {
	writer *bufio.Writer
	format Format
	err    error
}

func newTurtleEncoder(w io.Writer) Encoder {
	return &turtleEncoder{writer: bufio.NewWriter(w), format: FormatTurtle}
}

func newTriGEncoder(w io.Writer) Encoder {
	return &turtleEncoder{writer: bufio.NewWriter(w), format: FormatTriG}
}

func (e *turtleEncoder) Write(q Quad) error {
	if e.err != nil {
		return e.err
	}
	if q.S == nil || q.P.Value == "" || q.O == nil {
		return fmt.Errorf("turtle: missing statement fields")
	}
	line := renderTerm(q.S) + " " + renderIRI(q.P) + " " + renderTerm(q.O) + " ."
	if e.format == FormatTriG && q.G != nil {
		line = renderTerm(q.G) + " { " + line + " }"
	}
	_, err := e.writer.WriteString(line + "\n")
	if err != nil {
		e.err = err
	}
	return err
}

func (e *turtleEncoder) Flush() error {
	if e.err != nil {
		return e.err
	}
	return e.writer.Flush()
}

func (e *turtleEncoder) Close() error {
	return e.Flush()
}
