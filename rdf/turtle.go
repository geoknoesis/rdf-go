package rdf

import (
	"fmt"
	"strings"
)

const (
	rdfTypeIRI    = "http://www.w3.org/1999/02/22-rdf-syntax-ns#type"
	rdfReifiesIRI = "http://www.w3.org/1999/02/22-rdf-syntax-ns#reifies"
)

func parseTurtleStatement(prefixes map[string]string, baseIRI string, allowQuoted bool, debugStatements bool, line string) ([]Triple, error) {
	return parseTurtleTripleLine(prefixes, baseIRI, allowQuoted, debugStatements, line)
}

func parseTurtleTripleLine(prefixes map[string]string, baseIRI string, allowQuoted bool, debugStatements bool, line string) ([]Triple, error) {
	cursor := &turtleCursor{
		input:                      line,
		prefixes:                   prefixes,
		base:                       baseIRI,
		expansionTriples:           []Triple{},
		blankNodeCounter:           0,
		allowQuotedTripleStatement: allowQuoted,
		debugStatements:            debugStatements,
	}
	subject, err := cursor.parseSubject()
	if err != nil {
		return nil, err
	}
	if _, ok := subject.(TripleTerm); ok && cursor.lastTermReified {
		return nil, cursor.errorf("reified triple term cannot be used as subject")
	}
	cursor.skipWS()
	// Allow blank node property list as a standalone triple (no predicateObjectList)
	if cursor.lastTermBlankNodeList && cursor.pos < len(cursor.input) && cursor.input[cursor.pos] == '.' {
		cursor.pos++
		if err := cursor.ensureLineEnd(); err != nil {
			return nil, err
		}
		return cursor.expansionTriples, nil
	}
	// Allow standalone quoted triple statements when enabled
	if cursor.allowQuotedTripleStatement {
		if _, ok := subject.(TripleTerm); ok && cursor.pos < len(cursor.input) && cursor.input[cursor.pos] == '.' {
			cursor.pos++
			if err := cursor.ensureLineEnd(); err != nil {
				return nil, err
			}
			return cursor.expansionTriples, nil
		}
	}

	triples, err := cursor.parsePredicateObjectList(subject)
	if err != nil {
		return nil, err
	}

	// Append expansion triples (from collections and blank node property lists)
	triples = append(triples, cursor.expansionTriples...)

	return triples, nil
}

func normalizeTriGStatement(stmt string) string {
	for {
		idx := strings.Index(stmt, `\.:`)
		if idx < 0 {
			break
		}
		after := idx + 3
		j := after
		for j < len(stmt) && (stmt[j] == ' ' || stmt[j] == '\t') {
			j++
		}
		if j < len(stmt) && stmt[j] == ':' {
			stmt = stmt[:idx+2] + stmt[idx+3:]
			continue
		}
		break
	}
	return strings.ReplaceAll(stmt, `\.:`, `\. :`)
}

type turtleCursor struct {
	input                      string
	pos                        int
	prefixes                   map[string]string
	base                       string
	expansionTriples           []Triple // Triples generated from collections and blank node property lists
	blankNodeCounter           int      // Counter for generating unique blank node IDs
	lastTermBlankNodeList      bool
	allowQuotedTripleStatement bool
	lastTermReified            bool
	debugStatements            bool
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

func (c *turtleCursor) parseQuotedSubject() (Term, error) {
	c.skipWS()
	if c.pos >= len(c.input) {
		return nil, c.errorf("unexpected end of line")
	}
	switch {
	case strings.HasPrefix(c.input[c.pos:], "<<"):
		return c.parseTripleTerm()
	case c.input[c.pos] == '[':
		return c.parseAnonBlankNode()
	case c.input[c.pos] == '<':
		return c.parseIRI()
	case strings.HasPrefix(c.input[c.pos:], "_:"):
		return c.parseBlankNode()
	default:
		return c.parsePrefixedName()
	}
}

func (c *turtleCursor) parseQuotedObject() (Term, error) {
	c.skipWS()
	if c.pos >= len(c.input) {
		return nil, c.errorf("unexpected end of line")
	}
	switch {
	case strings.HasPrefix(c.input[c.pos:], "<<"):
		return c.parseTripleTerm()
	case c.input[c.pos] == '[':
		return c.parseAnonBlankNode()
	case c.input[c.pos] == '<':
		return c.parseIRI()
	case strings.HasPrefix(c.input[c.pos:], "_:"):
		return c.parseBlankNode()
	case c.input[c.pos] == '"':
		return c.parseLiteral()
	case c.input[c.pos] == '\'':
		return c.parseLiteralSingleQuote()
	default:
		if num, ok := c.tryParseNumericLiteral(); ok {
			return num, nil
		}
		if bool, ok := c.tryParseBooleanLiteral(); ok {
			return bool, nil
		}
		return c.parsePrefixedName()
	}
}

func (c *turtleCursor) parseAnonBlankNode() (Term, error) {
	if !c.consume('[') {
		return nil, c.errorf("expected '['")
	}
	c.skipWS()
	if c.pos < len(c.input) && c.input[c.pos] == ']' {
		c.pos++
		return c.newBlankNode(), nil
	}
	return nil, c.errorf("invalid blank node syntax")
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
	// parseTerm will skip whitespace, so we don't need to do it here
	return c.parseTerm(true)
}

func (c *turtleCursor) parseObjectList(subject Term, predicate IRI) ([]Triple, bool, error) {
	var triples []Triple
	for {
		object, err := c.parseObject()
		if err != nil {
			return nil, false, err
		}
		triples = append(triples, Triple{S: subject, P: predicate, O: object})

		// Handle reifier/annotation modifiers after the object
		var reifier Term
		for {
			c.skipWS()
			handled := false
			if c.pos < len(c.input) && c.input[c.pos] == '~' {
				reifier, err = c.parseReifier(subject, predicate, object)
				if err != nil {
					return nil, false, err
				}
				handled = true
			}
			if c.pos+1 < len(c.input) && c.input[c.pos] == '{' && c.input[c.pos+1] == '|' {
				if reifier == nil {
					reifier = c.newBlankNode()
					c.addReification(reifier, subject, predicate, object)
				}
				annotationTriples, err := c.parseAnnotationSyntax(reifier, true)
				if err != nil {
					return nil, false, err
				}
				triples = append(triples, annotationTriples...)
				handled = true
			}
			if !handled {
				break
			}
		}

		c.skipWS()
		// Check for comma (more objects)
		if c.pos < len(c.input) && c.input[c.pos] == ',' {
			c.pos++
			c.skipWS()
			continue
		}
		// Check for period (end of statement) - might be after blank node property list or collection
		if c.pos < len(c.input) && c.input[c.pos] == '.' {
			// Period found - end of statement
			// Consume it here and return
			c.pos++
			if err := c.ensureLineEnd(); err != nil {
				return nil, false, err
			}
			return triples, true, nil
		}
		break
	}
	return triples, false, nil
}

func (c *turtleCursor) parsePredicateObjectList(subject Term) ([]Triple, error) {
	var triples []Triple
	for {
		// Parse predicate (verb)
		predicate, err := c.parsePredicate()
		if err != nil {
			return nil, err
		}

		// Parse objectList (comma-separated objects)
		objectTriples, ended, err := c.parseObjectList(subject, predicate)
		if err != nil {
			return nil, err
		}
		triples = append(triples, objectTriples...)
		if ended {
			return triples, nil
		}

		c.skipWS()
		// Check for semicolon (more predicates), allowing repeated semicolons
		hadSemicolon := false
		for c.pos < len(c.input) && c.input[c.pos] == ';' {
			hadSemicolon = true
			c.pos++
			c.skipWS()
		}
		if hadSemicolon {
			// If we have a period after semicolons, end the statement
			if c.pos < len(c.input) && c.input[c.pos] == '.' {
				c.pos++
				if err := c.ensureLineEnd(); err != nil {
					return nil, err
				}
				break
			}
			continue
		}

		// Check for period (end of statement)
		if c.pos < len(c.input) && c.input[c.pos] == '.' {
			c.pos++
			if err := c.ensureLineEnd(); err != nil {
				return nil, err
			}
			break
		}
		// If we get here, something is wrong
		if c.debugStatements {
			end := c.pos + 40
			if end > len(c.input) {
				end = len(c.input)
			}
			return nil, c.errorf("expected ',' or ';' or '.' near %q", c.input[c.pos:end])
		}
		return nil, c.errorf("expected ',' or ';' or '.'")
	}
	return triples, nil
}

func (c *turtleCursor) parseTerm(allowLiteral bool) (Term, error) {
	c.skipWS()
	c.lastTermBlankNodeList = false
	c.lastTermReified = false
	if c.pos >= len(c.input) {
		return nil, c.errorf("unexpected end of line")
	}
	if term, ok, err := c.tryParseTermByPrefix(allowLiteral); ok || err != nil {
		return term, err
	}
	// Check for numeric literal or boolean literal
	if allowLiteral {
		if num, ok := c.tryParseNumericLiteral(); ok {
			return num, nil
		}
		if boolVal, ok := c.tryParseBooleanLiteral(); ok {
			return boolVal, nil
		}
	}
	return c.parsePrefixedName()
}

func (c *turtleCursor) tryParseTermByPrefix(allowLiteral bool) (Term, bool, error) {
	switch {
	case strings.HasPrefix(c.input[c.pos:], "<<"):
		term, err := c.parseTripleTerm()
		return term, true, err
	case c.input[c.pos] == '<':
		term, err := c.parseIRI()
		return term, true, err
	case strings.HasPrefix(c.input[c.pos:], "_:"):
		term, err := c.parseBlankNode()
		return term, true, err
	case c.input[c.pos] == '[':
		term, err := c.parseBlankNodePropertyList()
		return term, true, err
	case c.input[c.pos] == '(':
		term, err := c.parseCollection()
		return term, true, err
	case strings.HasPrefix(c.input[c.pos:], "\"\"\""):
		if !allowLiteral {
			return nil, true, c.errorf("literal not allowed here")
		}
		term, err := c.parseLongLiteral('"')
		return term, true, err
	case strings.HasPrefix(c.input[c.pos:], "'''"):
		if !allowLiteral {
			return nil, true, c.errorf("literal not allowed here")
		}
		term, err := c.parseLongLiteral('\'')
		return term, true, err
	case c.input[c.pos] == '"':
		if !allowLiteral {
			return nil, true, c.errorf("literal not allowed here")
		}
		term, err := c.parseLiteral()
		return term, true, err
	case c.input[c.pos] == '\'':
		if !allowLiteral {
			return nil, true, c.errorf("literal not allowed here")
		}
		term, err := c.parseLiteralSingleQuote()
		return term, true, err
	default:
		return nil, false, nil
	}
}

func (c *turtleCursor) parseIRI() (Term, error) {
	if !c.consume('<') {
		return nil, c.errorf("expected IRI")
	}
	start := c.pos
	for c.pos < len(c.input) && c.input[c.pos] != '>' {
		ch := c.input[c.pos]
		// According to RDF 1.2 Turtle spec, IRIREF allows most characters
		// Invalid characters: control chars (#x00-#x20), <, >, ", {, }, |, ^, `, \
		// But \ is allowed for Unicode escapes (UCHAR)
		if ch == '\\' {
			// Handle Unicode escape in IRI
			if c.pos+1 >= len(c.input) {
				return nil, c.errorf("unterminated IRI")
			}
			next := c.input[c.pos+1]
			if next == 'u' {
				// Unicode escape \uXXXX
				if c.pos+5 >= len(c.input) {
					return nil, c.errorf("unterminated IRI")
				}
				// Validate hex digits
				for i := 2; i < 6; i++ {
					hex := c.input[c.pos+i]
					if !((hex >= '0' && hex <= '9') || (hex >= 'a' && hex <= 'f') || (hex >= 'A' && hex <= 'F')) {
						return nil, c.errorf("invalid character in IRI")
					}
				}
				codePoint := rune(0)
				for i := 2; i < 6; i++ {
					hex := c.input[c.pos+i]
					var digit int
					if hex >= '0' && hex <= '9' {
						digit = int(hex - '0')
					} else if hex >= 'a' && hex <= 'f' {
						digit = int(hex - 'a' + 10)
					} else if hex >= 'A' && hex <= 'F' {
						digit = int(hex - 'A' + 10)
					} else {
						return nil, c.errorf("invalid character in IRI")
					}
					codePoint = codePoint*16 + rune(digit)
				}
				if !isValidUnicodeCodePoint(codePoint) || isDisallowedIRIChar(codePoint) {
					return nil, c.errorf("invalid character in IRI")
				}
				c.pos += 6
				continue
			} else if next == 'U' {
				// Unicode escape \UXXXXXXXX
				if c.pos+9 >= len(c.input) {
					return nil, c.errorf("unterminated IRI")
				}
				// Validate hex digits
				for i := 2; i < 10; i++ {
					hex := c.input[c.pos+i]
					if !((hex >= '0' && hex <= '9') || (hex >= 'a' && hex <= 'f') || (hex >= 'A' && hex <= 'F')) {
						return nil, c.errorf("invalid character in IRI")
					}
				}
				codePoint := rune(0)
				for i := 2; i < 10; i++ {
					hex := c.input[c.pos+i]
					var digit int
					if hex >= '0' && hex <= '9' {
						digit = int(hex - '0')
					} else if hex >= 'a' && hex <= 'f' {
						digit = int(hex - 'a' + 10)
					} else if hex >= 'A' && hex <= 'F' {
						digit = int(hex - 'A' + 10)
					} else {
						return nil, c.errorf("invalid character in IRI")
					}
					codePoint = codePoint*16 + rune(digit)
				}
				if !isValidUnicodeCodePoint(codePoint) || isDisallowedIRIChar(codePoint) {
					return nil, c.errorf("invalid character in IRI")
				}
				c.pos += 10
				continue
			} else {
				// Backslash not followed by u or U - invalid in IRI
				return nil, c.errorf("invalid character in IRI")
			}
		}
		// Check for invalid control characters and whitespace
		if ch <= 0x20 || (ch >= 0x7F && ch <= 0x9F) {
			return nil, c.errorf("invalid character in IRI")
		}
		// Check for other invalid characters: <, >, ", {, }, |, ^, ` (but we already check for > in loop condition)
		if ch == '<' || ch == '"' || ch == '{' || ch == '}' || ch == '|' || ch == '^' || ch == '`' {
			return nil, c.errorf("invalid character in IRI")
		}
		c.pos++
	}
	if c.pos >= len(c.input) {
		return nil, c.errorf("unterminated IRI")
	}
	value := c.input[start:c.pos]
	c.pos++

	// Resolve relative IRI against base if present
	if c.base != "" {
		resolved := resolveIRI(c.base, value)
		return IRI{Value: resolved}, nil
	}
	return IRI{Value: value}, nil
}

func isDisallowedIRIChar(codePoint rune) bool {
	if codePoint <= 0x20 || (codePoint >= 0x7F && codePoint <= 0x9F) {
		return true
	}
	switch codePoint {
	case '<', '>', '"', '{', '}', '|', '^', '`', '\\':
		return true
	}
	return false
}

func (c *turtleCursor) tryParseNumericLiteral() (Literal, bool) {
	start := c.pos
	if c.pos < len(c.input) && (c.input[c.pos] == '+' || c.input[c.pos] == '-') {
		c.pos++
	}

	if c.pos >= len(c.input) {
		c.pos = start
		return Literal{}, false
	}

	hasDot := false
	hasExponent := false
	hasDigits := false

	// Handle numbers that start with '.' (e.g. .1 or +.7)
	if c.input[c.pos] == '.' {
		if c.pos+1 < len(c.input) && c.input[c.pos+1] >= '0' && c.input[c.pos+1] <= '9' {
			hasDot = true
			c.pos++
		} else {
			c.pos = start
			return Literal{}, false
		}
	}

	for c.pos < len(c.input) {
		ch := c.input[c.pos]
		if ch >= '0' && ch <= '9' {
			hasDigits = true
			c.pos++
		} else if ch == '.' && !hasDot && !hasExponent {
			next := byte(0)
			if c.pos+1 < len(c.input) {
				next = c.input[c.pos+1]
			}
			// Treat '.' as decimal point only if followed by a digit or exponent.
			if (next >= '0' && next <= '9') || next == 'e' || next == 'E' {
				hasDot = true
				c.pos++
			} else {
				break
			}
		} else if (ch == 'e' || ch == 'E') && !hasExponent && hasDigits {
			hasExponent = true
			c.pos++
			if c.pos < len(c.input) && (c.input[c.pos] == '+' || c.input[c.pos] == '-') {
				c.pos++
			}
			if c.pos >= len(c.input) || c.input[c.pos] < '0' || c.input[c.pos] > '9' {
				c.pos = start
				return Literal{}, false
			}
		} else {
			break
		}
	}

	if !hasDigits {
		c.pos = start
		return Literal{}, false
	}

	lexical := c.input[start:c.pos]
	if c.pos < len(c.input) {
		next := c.input[c.pos]
		nextNext := byte(0)
		if c.pos+1 < len(c.input) {
			nextNext = c.input[c.pos+1]
		}
		if !isTurtleTerminator(next, nextNext) {
			c.pos = start
			return Literal{}, false
		}
	}
	var datatype IRI
	if hasExponent || (hasDot && strings.Contains(lexical, "e")) || (hasDot && strings.Contains(lexical, "E")) {
		datatype = IRI{Value: "http://www.w3.org/2001/XMLSchema#double"}
	} else if hasDot {
		datatype = IRI{Value: "http://www.w3.org/2001/XMLSchema#decimal"}
	} else {
		datatype = IRI{Value: "http://www.w3.org/2001/XMLSchema#integer"}
	}

	return Literal{Lexical: lexical, Datatype: datatype}, true
}

func (c *turtleCursor) tryParseBooleanLiteral() (Literal, bool) {
	start := c.pos
	if strings.HasPrefix(c.input[c.pos:], "true") && (c.pos+4 >= len(c.input) || isTermDelimiter(c.input[c.pos+4])) {
		c.pos += 4
		return Literal{
			Lexical:  "true",
			Datatype: IRI{Value: "http://www.w3.org/2001/XMLSchema#boolean"},
		}, true
	}
	if strings.HasPrefix(c.input[c.pos:], "false") && (c.pos+5 >= len(c.input) || isTermDelimiter(c.input[c.pos+5])) {
		c.pos += 5
		return Literal{
			Lexical:  "false",
			Datatype: IRI{Value: "http://www.w3.org/2001/XMLSchema#boolean"},
		}, true
	}
	c.pos = start
	return Literal{}, false
}

func (c *turtleCursor) parsePrefixedName() (Term, error) {
	start := c.pos
	for c.pos < len(c.input) {
		ch := c.input[c.pos]
		next := byte(0)
		if c.pos+1 < len(c.input) {
			next = c.input[c.pos+1]
		}
		if c.pos > start && c.input[c.pos-1] == '\\' {
			c.pos++
			continue
		}
		if isTurtleTerminator(ch, next) {
			break
		}
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
	if local == "" {
		base, ok := c.prefixes[prefix]
		if !ok {
			return nil, c.errorf("unknown prefix %q", prefix)
		}
		return IRI{Value: base}, nil
	}
	if local[0] == '.' || local[0] == '-' {
		return nil, c.errorf("invalid token %q", token)
	}
	if strings.HasSuffix(local, ".") {
		if len(local) < 2 || local[len(local)-2] != '\\' {
			return nil, c.errorf("invalid token %q", token)
		}
	}
	for i := 0; i < len(local); i++ {
		if local[i] == '~' {
			return nil, c.errorf("invalid token %q", token)
		}
		if local[i] == '^' {
			return nil, c.errorf("invalid token %q", token)
		}
		if local[i] == '\\' {
			if i+1 >= len(local) || !isValidPNLocalEscape(local[i+1]) {
				return nil, c.errorf("invalid token %q", token)
			}
			i++
			continue
		}
		if local[i] == '%' {
			if i+2 >= len(local) || !isHexDigit(local[i+1]) || !isHexDigit(local[i+2]) {
				return nil, c.errorf("invalid token %q", token)
			}
			i += 2
		}
	}
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
	if c.pos < len(c.input) && c.input[c.pos] == ':' {
		return nil, c.errorf("invalid blank node syntax")
	}
	for c.pos < len(c.input) {
		ch := c.input[c.pos]
		next := byte(0)
		if c.pos+1 < len(c.input) {
			next = c.input[c.pos+1]
		}
		if ch == ':' {
			break
		}
		if isTurtleTerminator(ch, next) {
			break
		}
		c.pos++
	}
	if start == c.pos {
		return nil, c.errorf("blank node id missing")
	}
	if c.input[c.pos-1] == '.' {
		return nil, c.errorf("invalid blank node syntax")
	}
	return BlankNode{ID: c.input[start:c.pos]}, nil
}

func (c *turtleCursor) parseLiteral() (Term, error) {
	return c.parseLiteralWithQuote('"')
}

func (c *turtleCursor) parseLiteralSingleQuote() (Term, error) {
	return c.parseLiteralWithQuote('\'')
}

func (c *turtleCursor) parseLiteralWithQuote(quoteChar byte) (Term, error) {
	if !c.consume(quoteChar) {
		return nil, c.errorf("expected literal")
	}
	var builder strings.Builder
	for c.pos < len(c.input) {
		ch := c.input[c.pos]
		if ch == quoteChar {
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
				c.pos += 2
			case 't':
				builder.WriteByte('\t')
				c.pos += 2
			case 'r':
				builder.WriteByte('\r')
				c.pos += 2
			case 'b':
				builder.WriteByte('\b')
				c.pos += 2
			case 'f':
				builder.WriteByte('\f')
				c.pos += 2
			case '"':
				builder.WriteByte('"')
				c.pos += 2
			case '\'':
				builder.WriteByte('\'')
				c.pos += 2
			case '\\':
				builder.WriteByte('\\')
				c.pos += 2
			case 'u':
				// Unicode escape \uXXXX - validate and decode (allow surrogate pairs)
				if c.pos+5 >= len(c.input) {
					return nil, c.errorf("invalid escape sequence")
				}
				var codePoint rune
				for i := 2; i < 6; i++ {
					hex := c.input[c.pos+i]
					var digit int
					if hex >= '0' && hex <= '9' {
						digit = int(hex - '0')
					} else if hex >= 'a' && hex <= 'f' {
						digit = int(hex - 'a' + 10)
					} else if hex >= 'A' && hex <= 'F' {
						digit = int(hex - 'A' + 10)
					} else {
						return nil, c.errorf("invalid escape sequence")
					}
					codePoint = codePoint*16 + rune(digit)
				}
				if codePoint >= 0xD800 && codePoint <= 0xDBFF {
					if c.pos+11 >= len(c.input) || c.input[c.pos+6] != '\\' || c.input[c.pos+7] != 'u' {
						return nil, c.errorf("invalid escape sequence")
					}
					low := decodeUChar(c.input[c.pos+8 : c.pos+12])
					if low < 0 || low < 0xDC00 || low > 0xDFFF {
						return nil, c.errorf("invalid escape sequence")
					}
					combined := 0x10000 + ((codePoint - 0xD800) << 10) + (low - 0xDC00)
					builder.WriteRune(rune(combined))
					c.pos += 12
					continue
				}
				if codePoint >= 0xDC00 && codePoint <= 0xDFFF {
					return nil, c.errorf("invalid escape sequence")
				}
				if !isValidUnicodeCodePoint(codePoint) {
					return nil, c.errorf("invalid escape sequence")
				}
				builder.WriteRune(codePoint)
				c.pos += 6
			case 'U':
				// Unicode escape \UXXXXXXXX - validate and decode
				if c.pos+9 >= len(c.input) {
					return nil, c.errorf("invalid escape sequence")
				}
				var codePoint rune
				for i := 2; i < 10; i++ {
					hex := c.input[c.pos+i]
					var digit int
					if hex >= '0' && hex <= '9' {
						digit = int(hex - '0')
					} else if hex >= 'a' && hex <= 'f' {
						digit = int(hex - 'a' + 10)
					} else if hex >= 'A' && hex <= 'F' {
						digit = int(hex - 'A' + 10)
					} else {
						return nil, c.errorf("invalid escape sequence")
					}
					codePoint = codePoint*16 + rune(digit)
				}
				if !isValidUnicodeCodePoint(codePoint) {
					return nil, c.errorf("invalid escape sequence")
				}
				builder.WriteRune(codePoint)
				c.pos += 10
			default:
				// Invalid escape sequence
				return nil, c.errorf("invalid escape sequence")
			}
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
		for c.pos < len(c.input) {
			ch := c.input[c.pos]
			next := byte(0)
			if c.pos+1 < len(c.input) {
				next = c.input[c.pos+1]
			}
			if isTurtleTerminator(ch, next) {
				break
			}
			c.pos++
		}
		lang := c.input[start:c.pos]
		if !isValidLangTag(lang) {
			return nil, c.errorf("invalid language tag")
		}
		if strings.HasPrefix(c.input[c.pos:], "^^") {
			return nil, c.errorf("literal cannot have both language tag and datatype")
		}
		return Literal{Lexical: lexical, Lang: lang}, nil
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

// parseLongLiteral parses a long string literal using triple quotes ("""...""" or ”'...”')
func (c *turtleCursor) parseLongLiteral(quoteChar byte) (Term, error) {
	// Consume the three opening quotes
	c.pos += 3

	var builder strings.Builder
	for c.pos < len(c.input) {
		// Check for closing triple quotes
		if c.pos+2 < len(c.input) &&
			c.input[c.pos] == quoteChar &&
			c.input[c.pos+1] == quoteChar &&
			c.input[c.pos+2] == quoteChar {
			// Found closing triple quotes
			c.pos += 3
			break
		}

		ch := c.input[c.pos]

		// In triple-quoted strings, we only need to escape the triple quote itself
		// Single quotes in """ strings and double quotes in ''' strings don't need escaping
		if ch == '\\' {
			if c.pos+1 >= len(c.input) {
				return nil, c.errorf("unterminated escape")
			}
			next := c.input[c.pos+1]
			// Check if it's escaping the triple quote
			if next == quoteChar && c.pos+3 < len(c.input) &&
				c.input[c.pos+2] == quoteChar && c.input[c.pos+3] == quoteChar {
				// Escaped triple quote - write one quote and skip the escape
				builder.WriteByte(quoteChar)
				c.pos += 2 // Skip \ and first quote, next iteration will handle the other two
				continue
			}
			// Handle other escape sequences
			switch next {
			case 'n':
				builder.WriteByte('\n')
				c.pos += 2
			case 't':
				builder.WriteByte('\t')
				c.pos += 2
			case 'r':
				builder.WriteByte('\r')
				c.pos += 2
			case 'b':
				builder.WriteByte('\b')
				c.pos += 2
			case 'f':
				builder.WriteByte('\f')
				c.pos += 2
			case '"':
				builder.WriteByte('"')
				c.pos += 2
			case '\'':
				builder.WriteByte('\'')
				c.pos += 2
			case '\\':
				builder.WriteByte('\\')
				c.pos += 2
			case 'u':
				// Unicode escape \uXXXX (allow surrogate pairs)
				if c.pos+5 >= len(c.input) {
					return nil, c.errorf("invalid escape sequence")
				}
				var codePoint rune
				for i := 2; i < 6; i++ {
					hex := c.input[c.pos+i]
					var digit int
					if hex >= '0' && hex <= '9' {
						digit = int(hex - '0')
					} else if hex >= 'a' && hex <= 'f' {
						digit = int(hex - 'a' + 10)
					} else if hex >= 'A' && hex <= 'F' {
						digit = int(hex - 'A' + 10)
					} else {
						return nil, c.errorf("invalid escape sequence")
					}
					codePoint = codePoint*16 + rune(digit)
				}
				if codePoint >= 0xD800 && codePoint <= 0xDBFF {
					if c.pos+11 >= len(c.input) || c.input[c.pos+6] != '\\' || c.input[c.pos+7] != 'u' {
						return nil, c.errorf("invalid escape sequence")
					}
					low := decodeUChar(c.input[c.pos+8 : c.pos+12])
					if low < 0 || low < 0xDC00 || low > 0xDFFF {
						return nil, c.errorf("invalid escape sequence")
					}
					combined := 0x10000 + ((codePoint - 0xD800) << 10) + (low - 0xDC00)
					builder.WriteRune(rune(combined))
					c.pos += 12
					continue
				}
				if codePoint >= 0xDC00 && codePoint <= 0xDFFF {
					return nil, c.errorf("invalid escape sequence")
				}
				if !isValidUnicodeCodePoint(codePoint) {
					return nil, c.errorf("invalid escape sequence")
				}
				builder.WriteRune(codePoint)
				c.pos += 6
			case 'U':
				// Unicode escape \UXXXXXXXX
				if c.pos+9 >= len(c.input) {
					return nil, c.errorf("invalid escape sequence")
				}
				var codePoint rune
				for i := 2; i < 10; i++ {
					hex := c.input[c.pos+i]
					var digit int
					if hex >= '0' && hex <= '9' {
						digit = int(hex - '0')
					} else if hex >= 'a' && hex <= 'f' {
						digit = int(hex - 'a' + 10)
					} else if hex >= 'A' && hex <= 'F' {
						digit = int(hex - 'A' + 10)
					} else {
						return nil, c.errorf("invalid escape sequence")
					}
					codePoint = codePoint*16 + rune(digit)
				}
				if !isValidUnicodeCodePoint(codePoint) {
					return nil, c.errorf("invalid escape sequence")
				}
				builder.WriteRune(codePoint)
				c.pos += 10
			default:
				// Invalid escape sequence
				return nil, c.errorf("invalid escape sequence")
			}
			continue
		}

		builder.WriteByte(ch)
		c.pos++
	}

	if c.pos >= len(c.input) {
		return nil, c.errorf("unterminated long string literal")
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

	// Reified triple terms use the form <<( ... )>> with no whitespace between << and (
	hasParens := false
	if c.pos < len(c.input) && c.input[c.pos] == '(' {
		hasParens = true
		c.pos++
		c.skipWS()
	} else {
		c.skipWS()
		if c.pos < len(c.input) && c.input[c.pos] == '(' {
			return nil, c.errorf("unexpected '(' after '<<'")
		}
	}

	// Parse subject (nested triple terms are allowed)
	subject, err := c.parseQuotedSubject()
	if err != nil {
		return nil, err
	}

	// Predicate must be IRI
	predicate, err := c.parsePredicate()
	if err != nil {
		return nil, err
	}

	// Parse object (nested triple terms are allowed)
	object, err := c.parseQuotedObject()
	if err != nil {
		return nil, err
	}

	c.skipWS()
	if hasParens {
		if !c.consume(')') {
			return nil, c.errorf("expected ')'")
		}
		c.skipWS()
	}
	c.lastTermReified = hasParens
	// Handle optional reifier syntax: << :s :p :o ~ :i >>
	if c.pos < len(c.input) && c.input[c.pos] == '~' {
		reifier, err := c.parseReifier(subject, predicate, object)
		if err != nil {
			return nil, err
		}
		c.skipWS()
		if !strings.HasPrefix(c.input[c.pos:], ">>") {
			return nil, c.errorf("expected '>>'")
		}
		c.pos += 2
		return reifier, nil
	}
	if !strings.HasPrefix(c.input[c.pos:], ">>") {
		return nil, c.errorf("expected '>>'")
	}
	c.pos += 2
	return TripleTerm{S: subject, P: predicate, O: object}, nil
}

// parseAnnotationSyntax parses annotation syntax {| predicateObjectList |}
// and returns the annotation triples using the provided subject.
// When useReifier is true, nested annotations create reifier nodes and rdf:reifies triples.
func (c *turtleCursor) parseAnnotationSyntax(annotationSubject Term, useReifier bool) ([]Triple, error) {
	// Consume {|
	if !strings.HasPrefix(c.input[c.pos:], "{|") {
		return nil, c.errorf("expected '{|'")
	}
	c.pos += 2
	c.skipWS()

	var annotationTriples []Triple

	// Parse predicateObjectList
	for {
		// Parse predicate
		pred, err := c.parsePredicate()
		if err != nil {
			return nil, err
		}

		// Parse objectList (comma-separated objects)
		for {
			obj, err := c.parseObject()
			if err != nil {
				return nil, err
			}
			// Create annotation triple
			annotationTriples = append(annotationTriples, Triple{
				S: annotationSubject,
				P: pred,
				O: obj,
			})

			c.skipWS()
			// Handle nested annotations on this annotation triple
			if c.pos+1 < len(c.input) && c.input[c.pos] == '{' && c.input[c.pos+1] == '|' {
				var nestedSubject Term
				if useReifier {
					nestedSubject = c.newBlankNode()
					c.addReification(nestedSubject, annotationSubject, pred, obj)
				} else {
					nestedSubject = TripleTerm{S: annotationSubject, P: pred, O: obj}
				}
				nestedTriples, err := c.parseAnnotationSyntax(nestedSubject, useReifier)
				if err != nil {
					return nil, err
				}
				annotationTriples = append(annotationTriples, nestedTriples...)
				c.skipWS()
			}
			// Check for comma (more objects)
			if c.pos < len(c.input) && c.input[c.pos] == ',' {
				c.pos++
				c.skipWS()
				continue
			}
			break
		}

		c.skipWS()
		// Check for semicolon (more predicates), allowing repeated semicolons
		hadSemicolon := false
		for c.pos < len(c.input) && c.input[c.pos] == ';' {
			hadSemicolon = true
			c.pos++
			c.skipWS()
		}
		if hadSemicolon {
			if c.pos+1 < len(c.input) && c.input[c.pos] == '|' && c.input[c.pos+1] == '}' {
				c.pos += 2
				break
			}
			continue
		}
		// Check for closing |}
		if c.pos+1 < len(c.input) && c.input[c.pos] == '|' && c.input[c.pos+1] == '}' {
			c.pos += 2
			break
		}
		// If we get here, something is wrong
		return nil, c.errorf("expected ',' or ';' or '|}'")
	}

	return annotationTriples, nil
}

func (c *turtleCursor) parseReifier(subject Term, predicate IRI, object Term) (Term, error) {
	if c.pos >= len(c.input) || c.input[c.pos] != '~' {
		return nil, c.errorf("expected '~'")
	}
	c.pos++
	c.skipWS()
	var reifier Term
	if strings.HasPrefix(c.input[c.pos:], ">>") ||
		(c.pos < len(c.input) && (c.input[c.pos] == '{' || c.input[c.pos] == ',' || c.input[c.pos] == ';' || c.input[c.pos] == '.')) {
		reifier = c.newBlankNode()
	} else {
		term, err := c.parseTerm(false)
		if err != nil {
			return nil, err
		}
		switch term.(type) {
		case IRI, BlankNode:
			reifier = term
		default:
			return nil, c.errorf("reifier must be IRI or blank node")
		}
	}
	c.addReification(reifier, subject, predicate, object)
	return reifier, nil
}

func (c *turtleCursor) addReification(reifier Term, subject Term, predicate IRI, object Term) {
	c.expansionTriples = append(c.expansionTriples, Triple{
		S: reifier,
		P: IRI{Value: rdfReifiesIRI},
		O: TripleTerm{S: subject, P: predicate, O: object},
	})
}

func (c *turtleCursor) newBlankNode() BlankNode {
	c.blankNodeCounter++
	return BlankNode{ID: fmt.Sprintf("b%d", c.blankNodeCounter)}
}

// parseCollection parses a collection (object*) and returns the head blank node.
// It also generates rdf:first/rdf:rest triples and stores them in expansionTriples.
func (c *turtleCursor) parseCollection() (Term, error) {
	if !c.consume('(') {
		return nil, c.errorf("expected '('")
	}
	c.skipWS()

	// Empty collection
	if c.pos < len(c.input) && c.input[c.pos] == ')' {
		c.pos++
		return IRI{Value: "http://www.w3.org/1999/02/22-rdf-syntax-ns#nil"}, nil
	}

	var objects []Term
	for {
		c.skipWS()
		if c.pos >= len(c.input) {
			return nil, c.errorf("unterminated collection")
		}
		if c.input[c.pos] == ')' {
			c.pos++
			break
		}
		obj, err := c.parseTerm(true)
		if err != nil {
			return nil, err
		}
		objects = append(objects, obj)
		c.skipWS()
		if c.pos < len(c.input) && c.input[c.pos] == ')' {
			c.pos++
			break
		}
	}

	if len(objects) == 0 {
		return IRI{Value: "http://www.w3.org/1999/02/22-rdf-syntax-ns#nil"}, nil
	}

	// Generate rdf:first/rdf:rest triples
	head := c.newBlankNode()
	rdfFirst := IRI{Value: "http://www.w3.org/1999/02/22-rdf-syntax-ns#first"}
	rdfRest := IRI{Value: "http://www.w3.org/1999/02/22-rdf-syntax-ns#rest"}
	rdfNil := IRI{Value: "http://www.w3.org/1999/02/22-rdf-syntax-ns#nil"}

	current := head
	for i, obj := range objects {
		// rdf:first triple
		c.expansionTriples = append(c.expansionTriples, Triple{
			S: current,
			P: rdfFirst,
			O: obj,
		})

		// rdf:rest triple
		var rest Term
		if i == len(objects)-1 {
			rest = rdfNil
		} else {
			rest = c.newBlankNode()
		}
		c.expansionTriples = append(c.expansionTriples, Triple{
			S: current,
			P: rdfRest,
			O: rest,
		})

		if bn, ok := rest.(BlankNode); ok {
			current = bn
		}
	}

	return head, nil
}

// parseBlankNodePropertyList parses [predicateObjectList] and returns a blank node.
// It also generates triples from the predicateObjectList and stores them in expansionTriples.
func (c *turtleCursor) parseBlankNodePropertyList() (Term, error) {
	if !c.consume('[') {
		return nil, c.errorf("expected '['")
	}
	c.skipWS()

	// Empty blank node property list []
	if c.pos < len(c.input) && c.input[c.pos] == ']' {
		c.pos++
		c.lastTermBlankNodeList = true
		return c.newBlankNode(), nil
	}

	bn := c.newBlankNode()

	// Parse predicateObjectList
	for {
		// Parse predicate (verb)
		predicate, err := c.parsePredicate()
		if err != nil {
			return nil, err
		}

		// Parse objectList (comma-separated objects)
		for {
			object, err := c.parseObject()
			if err != nil {
				return nil, err
			}
			c.expansionTriples = append(c.expansionTriples, Triple{
				S: bn,
				P: predicate,
				O: object,
			})

			c.skipWS()
			// Check for closing bracket first (before comma)
			if c.pos < len(c.input) && c.input[c.pos] == ']' {
				c.pos++ // Consume ']'
				// Skip whitespace after ']' so the caller sees the next token
				c.skipWS()
				c.lastTermBlankNodeList = true
				return bn, nil
			}
			// Check for comma (more objects)
			if c.pos < len(c.input) && c.input[c.pos] == ',' {
				c.pos++
				c.skipWS()
				continue
			}
			c.lastTermBlankNodeList = true
			break
		}

		c.skipWS()
		// Check for closing bracket (before semicolon)
		if c.pos < len(c.input) && c.input[c.pos] == ']' {
			c.pos++ // Consume ']'
			// Skip whitespace after ']' so the caller sees the next token
			c.skipWS()
			break
		}
		// Check for semicolon (more predicates), allowing trailing semicolons
		hadSemicolon := false
		for c.pos < len(c.input) && c.input[c.pos] == ';' {
			hadSemicolon = true
			c.pos++
			c.skipWS()
		}
		if hadSemicolon {
			// Allow closing |} after trailing semicolon(s)
			if c.pos+1 < len(c.input) && c.input[c.pos] == '|' && c.input[c.pos+1] == '}' {
				c.pos += 2
				break
			}
			continue
		}
		// If we get here, something is wrong
		return nil, c.errorf("expected ',' or ';' or ']'")
	}

	return bn, nil
}

func isTurtleTerminator(ch byte, next byte) bool {
	switch ch {
	case ' ', '\t', '\r', '\n', ';', ',', '(', ')', '[', ']', '}', '>', '"', '\'':
		return true
	case '<':
		return true
	case '.':
		// Dot is a terminator only if followed by whitespace or a list/statement delimiter.
		if next == 0 {
			return true
		}
		switch next {
		case ' ', '\t', '\r', '\n', ';', ',', ')', ']', '}':
			return true
		default:
			return false
		}
	default:
		return false
	}
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
