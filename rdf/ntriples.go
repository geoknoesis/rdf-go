package rdf

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// Triple decoder for N-Triples
type ntTripleDecoder struct {
	reader      *bufio.Reader
	err         error
	opts        DecodeOptions
	lineNum     int    // Current line number (1-based)
	tripleCount int64  // Number of triples processed
}

func newNTriplesTripleDecoder(r io.Reader) TripleDecoder {
	return newNTriplesTripleDecoderWithOptions(r, DefaultDecodeOptions())
}

func newNTriplesTripleDecoderWithOptions(r io.Reader, opts DecodeOptions) TripleDecoder {
	return &ntTripleDecoder{
		reader:      bufio.NewReader(r),
		opts:        normalizeDecodeOptions(opts),
		lineNum:     0,
		tripleCount: 0,
	}
}

func (d *ntTripleDecoder) Next() (Triple, error) {
	for {
		if err := checkDecodeContext(d.opts.Context); err != nil {
			d.err = err
			return Triple{}, err
		}
		line, err := d.readLine()
		if err != nil {
			if err == io.EOF {
				return Triple{}, io.EOF
			}
			d.err = err
			return Triple{}, err
		}
		d.lineNum++
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// Check triple count limit
		if d.opts.MaxTriples > 0 && d.tripleCount >= d.opts.MaxTriples {
			err := WrapParseErrorWithPosition("ntriples", line, d.lineNum, 0, -1, ErrTripleLimitExceeded)
			d.err = err
			return Triple{}, err
		}
		
		triple, err := parseNTTripleLine(line)
		if err != nil {
			err = WrapParseErrorWithPosition("ntriples", line, d.lineNum, 0, -1, err)
			d.err = err
			return Triple{}, err
		}
		d.tripleCount++
		return triple, nil
	}
}

func (d *ntTripleDecoder) Err() error { return d.err }
func (d *ntTripleDecoder) Close() error {
	return nil
}

// Quad decoder for N-Quads
type ntQuadDecoder struct {
	reader      *bufio.Reader
	err         error
	opts        DecodeOptions
	lineNum     int    // Current line number (1-based)
	quadCount   int64  // Number of quads processed
}

func newNQuadsQuadDecoder(r io.Reader) QuadDecoder {
	return newNQuadsQuadDecoderWithOptions(r, DefaultDecodeOptions())
}

func newNQuadsQuadDecoderWithOptions(r io.Reader, opts DecodeOptions) QuadDecoder {
	return &ntQuadDecoder{
		reader:    bufio.NewReader(r),
		opts:      normalizeDecodeOptions(opts),
		lineNum:   0,
		quadCount: 0,
	}
}

func (d *ntQuadDecoder) Next() (Quad, error) {
	for {
		if err := checkDecodeContext(d.opts.Context); err != nil {
			d.err = err
			return Quad{}, err
		}
		line, err := d.readLine()
		if err != nil {
			if err == io.EOF {
				return Quad{}, io.EOF
			}
			d.err = err
			return Quad{}, err
		}
		d.lineNum++
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// Check quad count limit
		if d.opts.MaxTriples > 0 && d.quadCount >= d.opts.MaxTriples {
			err := WrapParseErrorWithPosition("nquads", line, d.lineNum, 0, -1, ErrTripleLimitExceeded)
			d.err = err
			return Quad{}, err
		}
		
		quad, err := parseNTQuadLine(line)
		if err != nil {
			err = WrapParseErrorWithPosition("nquads", line, d.lineNum, 0, -1, err)
			d.err = err
			return Quad{}, err
		}
		d.quadCount++
		return quad, nil
	}
}

func (d *ntQuadDecoder) Err() error { return d.err }
func (d *ntQuadDecoder) Close() error {
	return nil
}

// Shared readLine method
func (d *ntTripleDecoder) readLine() (string, error) {
	return readLineWithLimit(d.reader, d.opts.MaxLineBytes)
}

func (d *ntQuadDecoder) readLine() (string, error) {
	return readLineWithLimit(d.reader, d.opts.MaxLineBytes)
}
func parseNTTripleLine(line string) (Triple, error) {
	cursor, subject, predicate, object, err := parseNTCore(line, "N-Triples")
	if err != nil {
		return Triple{}, err
	}
	cursor.skipWS()
	if !cursor.consume('.') {
		return Triple{}, cursor.errorf("expected '.' at end of statement")
	}
	// Check for graph term (not allowed in N-Triples)
	// But allow comments (starting with #)
	cursor.skipWS()
	if cursor.pos < len(cursor.input) {
		// Allow comments
		if cursor.input[cursor.pos] == '#' {
			// Comment - rest of line is ignored, this is valid
			return Triple{S: subject, P: predicate, O: object}, nil
		}
		// If not a comment and not end of line, it's an error
		if cursor.input[cursor.pos] != '\n' && cursor.input[cursor.pos] != '\r' {
			return Triple{}, cursor.errorf("graph term not allowed in N-Triples")
		}
	}
	return Triple{S: subject, P: predicate, O: object}, nil
}

func parseNTQuadLine(line string) (Quad, error) {
	cursor, subject, predicate, object, err := parseNTCore(line, "N-Quads")
	if err != nil {
		return Quad{}, err
	}
	graph := cursor.parseOptionalTerm()
	if graph != nil {
		if _, ok := graph.(TripleTerm); ok {
			return Quad{}, cursor.errorf("triple term cannot be used as graph name")
		}
		// If graph term is present, validate it's not a relative IRI
		if iri, ok := graph.(IRI); ok {
			// Validate IRI is absolute (has scheme)
			if !strings.Contains(iri.Value, ":") || strings.HasPrefix(iri.Value, "//") {
				// Check if it's a relative IRI (no scheme)
				hasScheme := false
				for i, ch := range iri.Value {
					if ch == ':' {
						if i > 0 {
							scheme := iri.Value[:i]
							validScheme := true
							for _, sch := range scheme {
								if !((sch >= 'a' && sch <= 'z') || (sch >= 'A' && sch <= 'Z') ||
									(sch >= '0' && sch <= '9') || sch == '+' || sch == '-' || sch == '.') {
									validScheme = false
									break
								}
							}
							if validScheme && len(scheme) > 0 {
								hasScheme = true
								break
							}
						}
					}
					if ch == '/' || ch == '?' || ch == '#' {
						break
					}
				}
				if !hasScheme {
					return Quad{}, cursor.errorf("invalid IRI: relative IRI not allowed in N-Quads")
				}
			}
		}
	}
	cursor.skipWS()
	if !cursor.consume('.') {
		return Quad{}, cursor.errorf("expected '.' at end of statement")
	}
	return Quad{S: subject, P: predicate, O: object, G: graph}, nil
}

func parseNTCore(line string, context string) (*ntCursor, Term, IRI, Term, error) {
	cursor := &ntCursor{input: line}
	cursor.skipWS()
	subject, err := cursor.parseSubject()
	if err != nil {
		return cursor, nil, IRI{}, nil, err
	}
	if _, ok := subject.(TripleTerm); ok {
		return cursor, nil, IRI{}, nil, cursor.errorf("triple term cannot be used as subject in %s", context)
	}
	// Predicate must be an IRI, not a triple term
	cursor.skipWS()
	if strings.HasPrefix(cursor.input[cursor.pos:], "<<") {
		return cursor, nil, IRI{}, nil, cursor.errorf("triple term cannot be used as predicate")
	}
	predicate, err := cursor.parseIRI()
	if err != nil {
		return cursor, nil, IRI{}, nil, err
	}
	object, err := cursor.parseObject()
	if err != nil {
		return cursor, nil, IRI{}, nil, err
	}
	return cursor, subject, predicate, object, nil
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
	// Check if we're already at '<' (handles minimal whitespace case)
	if c.pos >= len(c.input) || c.input[c.pos] != '<' {
		return IRI{}, c.errorf("expected IRI")
	}
	c.pos++ // Consume '<'
	start := c.pos
	for c.pos < len(c.input) && c.input[c.pos] != '>' {
		ch := c.input[c.pos]
		// Validate IRI characters - spaces, newlines, and control chars are invalid
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			return IRI{}, c.errorf("invalid character in IRI")
		}
		// Check for escape sequences - only Unicode escapes are allowed in IRIs, not other escapes
		if ch == '\\' {
			if c.pos+1 < len(c.input) {
				next := c.input[c.pos+1]
				if next == 'u' {
					// Unicode escape \uXXXX - validate it has 4 valid hex digits
					if c.pos+5 >= len(c.input) {
						return IRI{}, c.errorf("invalid character in IRI")
					}
					for i := 2; i < 6; i++ {
						hex := c.input[c.pos+i]
						if !((hex >= '0' && hex <= '9') || (hex >= 'a' && hex <= 'f') || (hex >= 'A' && hex <= 'F')) {
							return IRI{}, c.errorf("invalid character in IRI")
						}
					}
					c.pos += 6
					continue
				} else if next == 'U' {
					// Unicode escape \UXXXXXXXX - validate it has 8 valid hex digits
					if c.pos+9 >= len(c.input) {
						return IRI{}, c.errorf("invalid character in IRI")
					}
					for i := 2; i < 10; i++ {
						hex := c.input[c.pos+i]
						if !((hex >= '0' && hex <= '9') || (hex >= 'a' && hex <= 'f') || (hex >= 'A' && hex <= 'F')) {
							return IRI{}, c.errorf("invalid character in IRI")
						}
					}
					c.pos += 10
					continue
				} else {
					// Other escape sequences (like \n, \t, etc.) are not allowed in IRIs
					return IRI{}, c.errorf("invalid character in IRI")
				}
			}
		}
		c.pos++
	}
	if c.pos >= len(c.input) {
		return IRI{}, c.errorf("unterminated IRI")
	}
	value := c.input[start:c.pos]
	c.pos++ // Advance past '>'

	// Validate IRI value - reject relative IRIs
	// IRIs must be absolute (have a scheme like http:, https:, etc.)
	if strings.HasPrefix(value, "//") {
		return IRI{}, c.errorf("invalid IRI: relative IRI without scheme")
	}
	// Check if it's a relative IRI (no scheme, just a path like <s> or <p>)
	// Valid IRIs must have a scheme followed by :
	hasScheme := false
	for i, ch := range value {
		if ch == ':' {
			// Found colon - check if there's a scheme before it
			if i > 0 {
				// Check if the part before : looks like a scheme (letters, digits, +, -, .)
				scheme := value[:i]
				validScheme := true
				for _, sch := range scheme {
					if !((sch >= 'a' && sch <= 'z') || (sch >= 'A' && sch <= 'Z') ||
						(sch >= '0' && sch <= '9') || sch == '+' || sch == '-' || sch == '.') {
						validScheme = false
						break
					}
				}
				if validScheme && len(scheme) > 0 {
					hasScheme = true
					break
				}
			}
		}
		// Stop checking if we hit a non-scheme character
		if ch == '/' || ch == '?' || ch == '#' {
			break
		}
	}
	if !hasScheme {
		return IRI{}, c.errorf("invalid IRI: relative IRI not allowed in N-Triples")
	}

	return IRI{Value: value}, nil
}

func (c *ntCursor) parseBlankNode() (BlankNode, error) {
	c.skipWS()
	if !strings.HasPrefix(c.input[c.pos:], "_:") {
		return BlankNode{}, c.errorf("expected blank node")
	}
	c.pos += 2
	// Check for double colon (invalid: _::a)
	if c.pos < len(c.input) && c.input[c.pos] == ':' {
		return BlankNode{}, c.errorf("invalid blank node syntax")
	}
	start := c.pos
	for c.pos < len(c.input) && !isTermDelimiter(c.input[c.pos]) {
		// Blank node IDs cannot contain colons (except the initial _:)
		if c.input[c.pos] == ':' {
			return BlankNode{}, c.errorf("invalid blank node syntax")
		}
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
	// Collect the escaped string (with escape sequences intact)
	// Track whether we're in an escape sequence to handle escaped quotes
	var escapedBuilder strings.Builder
	escapeNext := false
	for c.pos < len(c.input) {
		ch := c.input[c.pos]
		if escapeNext {
			// We're processing an escape sequence
			escapedBuilder.WriteByte('\\')
			escapedBuilder.WriteByte(ch)
			c.pos++
			escapeNext = false
			// For unicode escapes, collect the hex digits
			if ch == 'u' {
				if c.pos+4 > len(c.input) {
					return Literal{}, c.errorf("invalid escape sequence")
				}
				for i := 0; i < 4 && c.pos < len(c.input); i++ {
					escapedBuilder.WriteByte(c.input[c.pos])
					c.pos++
				}
			} else if ch == 'U' {
				if c.pos+8 > len(c.input) {
					return Literal{}, c.errorf("invalid escape sequence")
				}
				for i := 0; i < 8 && c.pos < len(c.input); i++ {
					escapedBuilder.WriteByte(c.input[c.pos])
					c.pos++
				}
			}
			continue
		}
		if ch == '\\' {
			// Start of escape sequence
			if c.pos+1 >= len(c.input) {
				return Literal{}, c.errorf("unterminated escape")
			}
			escapeNext = true
			c.pos++
			continue
		}
		if ch == '"' {
			// End of string
			c.pos++
			break
		}
		escapedBuilder.WriteByte(ch)
		c.pos++
	}
	if escapeNext {
		return Literal{}, c.errorf("unterminated escape")
	}
	if c.pos > len(c.input) || (c.pos == len(c.input) && c.input[c.pos-1] != '"') {
		// Check if we found the closing quote
		if c.pos == 0 || c.input[c.pos-1] != '"' {
			return Literal{}, c.errorf("unterminated string literal")
		}
	}

	// Unescape using shared function
	lexical, err := UnescapeString(escapedBuilder.String())
	if err != nil {
		return Literal{}, c.errorf("%v", err)
	}
	c.skipWS()
	if strings.HasPrefix(c.input[c.pos:], "@") {
		c.pos++
		start := c.pos
		for c.pos < len(c.input) && !isTermDelimiter(c.input[c.pos]) {
			c.pos++
		}
		lang := c.input[start:c.pos]
		if !isValidLangTag(lang) {
			return Literal{}, c.errorf("invalid language tag")
		}
		return Literal{Lexical: lexical, Lang: lang}, nil
	}
	if strings.HasPrefix(c.input[c.pos:], "^^") {
		c.pos += 2
		dt, err := c.parseIRI()
		if err != nil {
			return Literal{}, err
		}
		// langString and dirLangString cannot be used as explicit datatypes
		// They must be expressed using @lang syntax
		if dt.Value == rdfLangStringIRI || dt.Value == rdfDirLangStringIRI {
			return Literal{}, c.errorf("langString and dirLangString cannot be used as explicit datatypes")
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
	c.skipWS()
	if !c.consume('(') {
		return nil, c.errorf("expected '('")
	}
	c.skipWS()

	// Parse subject (nested triple terms are allowed)
	subject, err := c.parseSubject()
	if err != nil {
		return nil, err
	}

	// Predicate must be IRI
	predicate, err := c.parseIRI()
	if err != nil {
		return nil, err
	}

	// Parse object (nested triple terms are allowed)
	object, err := c.parseObject()
	if err != nil {
		return nil, err
	}

	c.skipWS()
	if !c.consume(')') {
		return nil, c.errorf("expected ')'")
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
	case ' ', '\t', '\r', '\n', '.', ')', '<', '>':
		return true
	default:
		return false
	}
}

// Triple encoder for N-Triples
type ntTripleEncoder struct {
	writer *bufio.Writer
	err    error
}

func newNTriplesTripleEncoder(w io.Writer) TripleEncoder {
	return &ntTripleEncoder{writer: bufio.NewWriter(w)}
}

func (e *ntTripleEncoder) Write(t Triple) error {
	if e.err != nil {
		return e.err
	}
	if t.S == nil || t.P.Value == "" || t.O == nil {
		return fmt.Errorf("ntriples: missing statement fields")
	}
	line := renderTerm(t.S) + " " + renderIRI(t.P) + " " + renderTerm(t.O) + " .\n"
	_, err := e.writer.WriteString(line)
	if err != nil {
		e.err = err
	}
	return err
}

func (e *ntTripleEncoder) Flush() error {
	if e.err != nil {
		return e.err
	}
	return e.writer.Flush()
}

func (e *ntTripleEncoder) Close() error {
	if e.err != nil {
		return e.err
	}
	if err := e.writer.Flush(); err != nil {
		e.err = err
		return err
	}
	e.err = fmt.Errorf("ntriples: writer closed")
	return nil
}

// Quad encoder for N-Quads
type ntQuadEncoder struct {
	writer *bufio.Writer
	err    error
}

func newNQuadsQuadEncoder(w io.Writer) QuadEncoder {
	return &ntQuadEncoder{writer: bufio.NewWriter(w)}
}

func (e *ntQuadEncoder) Write(q Quad) error {
	if e.err != nil {
		return e.err
	}
	if q.IsZero() {
		return fmt.Errorf("nquads: empty statement")
	}
	if q.S == nil || q.P.Value == "" || q.O == nil {
		return fmt.Errorf("nquads: missing statement fields")
	}
	line := renderTerm(q.S) + " " + renderIRI(q.P) + " " + renderTerm(q.O)
	if q.G != nil {
		line += " " + renderTerm(q.G)
	}
	line += " .\n"
	_, err := e.writer.WriteString(line)
	if err != nil {
		e.err = err
	}
	return err
}

func (e *ntQuadEncoder) Flush() error {
	if e.err != nil {
		return e.err
	}
	return e.writer.Flush()
}

func (e *ntQuadEncoder) Close() error {
	if e.err != nil {
		return e.err
	}
	if err := e.writer.Flush(); err != nil {
		e.err = err
		return err
	}
	e.err = fmt.Errorf("nquads: writer closed")
	return nil
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
