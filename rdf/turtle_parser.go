package rdf

import (
	"fmt"
	"io"
	"os"
	"strings"
)

type turtleParser struct {
	lexer                      *turtleLexer
	opts                       DecodeOptions
	prefixes                   map[string]string
	baseIRI                    string
	allowQuotedTripleStatement bool
	pending                    []Triple
	expansionTriples           []Triple // Triples from collections and blank node lists
	blankNodeCounter           int
	err                        error
}

func newTurtleParser(r io.Reader, opts DecodeOptions) *turtleParser {
	if opts.AllowEnvOverrides && os.Getenv("TURTLE_ALLOW_QT_STMT") != "" {
		opts.AllowQuotedTripleStatement = true
	}
	return &turtleParser{
		lexer:                      newTurtleLexer(r, opts),
		opts:                       normalizeDecodeOptions(opts),
		prefixes:                   map[string]string{},
		allowQuotedTripleStatement: opts.AllowQuotedTripleStatement,
		blankNodeCounter:           0,
	}
}

func (p *turtleParser) newBlankNode() BlankNode {
	p.blankNodeCounter++
	return BlankNode{ID: fmt.Sprintf("b%d", p.blankNodeCounter)}
}

func (p *turtleParser) NextTriple() (Triple, error) {
	if len(p.pending) > 0 {
		next := p.pending[0]
		p.pending = p.pending[1:]
		return next, nil
	}
	for {
		if err := checkDecodeContext(p.opts.Context); err != nil {
			p.err = err
			return Triple{}, err
		}
		var statement strings.Builder
		for {
			if err := checkDecodeContext(p.opts.Context); err != nil {
				p.err = err
				return Triple{}, err
			}
			token := p.lexer.Next()
			switch token.Kind {
			case TokEOF:
				if statement.Len() == 0 {
					return Triple{}, io.EOF
				}
				// Parse what we have so far
				line := strings.TrimSpace(statement.String())
				if line == "" {
					return Triple{}, io.EOF
				}
				triples, err := p.parseStatement(line)
				if err != nil {
					p.err = err
					return Triple{}, err
				}
				if len(triples) == 0 {
					return Triple{}, io.EOF
				}
				if len(triples) > 1 {
					p.pending = triples[1:]
				}
				return triples[0], nil
			case TokError:
				p.err = token.Err
				return Triple{}, token.Err
			case TokLine:
				if statement.Len() == 0 && p.handleDirective(token.Lexeme) {
					continue
				}
				if err := p.appendStatementPart(&statement, token.Lexeme); err != nil {
					p.err = err
					return Triple{}, err
				}
				stmt := strings.TrimSpace(statement.String())
				if stmt != "" && isStatementComplete(stmt) {
					triples, err := p.parseStatement(stmt)
					if err != nil {
						p.err = err
						return Triple{}, err
					}
					if len(triples) == 0 {
						statement.Reset()
						continue
					}
					if len(triples) > 1 {
						p.pending = triples[1:]
					}
					return triples[0], nil
				}
			}
		}
	}
}

func (p *turtleParser) parseStatement(line string) ([]Triple, error) {
	tokens, err := tokenizeTurtleLine(line)
	if err != nil {
		return nil, err
	}
	if handled, err := p.parseDirectiveTokens(tokens); err != nil {
		return nil, err
	} else if handled {
		return nil, nil
	}
	return p.parseTriplesTokens(tokens, line)
}

func (p *turtleParser) Err() error { return p.err }

func (p *turtleParser) appendStatementPart(builder *strings.Builder, part string) error {
	if builder.Len() > 0 {
		builder.WriteString(" ")
	}
	builder.WriteString(part)
	if p.opts.MaxStatementBytes > 0 && builder.Len() > p.opts.MaxStatementBytes {
		return ErrStatementTooLong
	}
	return nil
}

func (p *turtleParser) wrapParseError(statement string, err error) error {
	if p.opts.DebugStatements || (p.opts.AllowEnvOverrides && os.Getenv("TURTLE_DEBUG_STATEMENT") != "") {
		return WrapParseError("turtle", statement, -1, err)
	}
	return WrapParseError("turtle", "", -1, err)
}

func (p *turtleParser) parseTripleLine(line string) ([]Triple, error) {
	debugStatements := p.opts.DebugStatements || (p.opts.AllowEnvOverrides && os.Getenv("TURTLE_DEBUG_STATEMENT") != "")
	triples, err := parseTurtleTripleLine(p.prefixes, p.baseIRI, p.allowQuotedTripleStatement, debugStatements, line)
	if err != nil {
		return nil, p.wrapParseError(line, err)
	}
	return triples, nil
}

func (p *turtleParser) handleDirective(line string) bool {
	if prefix, iri, ok := parseAtPrefixDirective(line, true); ok {
		p.prefixes[prefix] = iri
		return true
	}
	if prefix, iri, ok := parseBarePrefixDirective(line); ok {
		p.prefixes[prefix] = iri
		return true
	}
	if parseVersionDirective(line) {
		p.allowQuotedTripleStatement = true
		return true
	}
	if iri, ok := parseAtBaseDirective(line); ok {
		p.baseIRI = iri
		return true
	}
	if iri, ok := parseBaseDirective(line); ok {
		p.baseIRI = iri
		return true
	}
	return false
}

func (p *turtleParser) parseDirectiveTokens(tokens []turtleToken) (bool, error) {
	if len(tokens) == 0 {
		return false, nil
	}
	switch tokens[0].Kind {
	case TokPrefix:
		if tokens[0].Lexeme != lexPrefix && !strings.EqualFold(tokens[0].Lexeme, lexPrefixBare) {
			return false, nil
		}
		if len(tokens) < 3 {
			return false, nil
		}
		prefixTok := tokens[1]
		iriTok := tokens[2]
		if prefixTok.Kind != TokPNAMENS || iriTok.Kind != TokIRIRef {
			return false, nil
		}
		prefix := strings.TrimSuffix(prefixTok.Lexeme, ":")
		iri := strings.Trim(iriTok.Lexeme, "<>")
		p.prefixes[prefix] = iri
		return true, nil
	case TokBase:
		if tokens[0].Lexeme != lexBase && !strings.EqualFold(tokens[0].Lexeme, lexBaseBare) {
			return false, nil
		}
		if len(tokens) < 2 {
			return false, nil
		}
		iriTok := tokens[1]
		if iriTok.Kind != TokIRIRef {
			return false, nil
		}
		p.baseIRI = strings.Trim(iriTok.Lexeme, "<>")
		return true, nil
	case TokVersion:
		if tokens[0].Lexeme != lexVersion && !strings.EqualFold(tokens[0].Lexeme, lexVersionBare) {
			return false, nil
		}
		p.allowQuotedTripleStatement = true
		return true, nil
	default:
		return false, nil
	}
}

func (p *turtleParser) parseTriplesTokens(tokens []turtleToken, line string) ([]Triple, error) {
	stream := &turtleTokenStream{tokens: tokens}
	subject, err := p.parseSubjectTokens(stream)
	if err != nil {
		return nil, p.wrapParseError(line, err)
	}
	triples, err := p.parsePredicateObjectListTokens(stream, subject)
	if err != nil {
		return nil, p.wrapParseError(line, err)
	}
	// Add expansion triples (from collections and blank node lists)
	triples = append(triples, p.expansionTriples...)
	p.expansionTriples = p.expansionTriples[:0] // Reset for next statement
	if stream.peek().Kind == TokDot {
		stream.next()
	}
	if stream.peek().Kind != TokEOF {
		return nil, p.wrapParseError(line, fmt.Errorf("unexpected token after statement: %v", stream.peek().Kind))
	}
	return triples, nil
}

func (p *turtleParser) parseSubjectTokens(stream *turtleTokenStream) (Term, error) {
	return p.parseTermTokens(stream, false)
}

func (p *turtleParser) parsePredicateObjectListTokens(stream *turtleTokenStream, subject Term) ([]Triple, error) {
	var triples []Triple
	for {
		predicate, err := p.parseVerbTokens(stream)
		if err != nil {
			return nil, err
		}
		objectTriples, err := p.parseObjectListTokens(stream, subject, predicate)
		if err != nil {
			return nil, err
		}
		triples = append(triples, objectTriples...)
		if stream.peek().Kind == TokSemicolon {
			for stream.peek().Kind == TokSemicolon {
				stream.next()
			}
			if stream.peek().Kind == TokDot || stream.peek().Kind == TokEOF {
				return triples, nil
			}
			continue
		}
		return triples, nil
	}
}

func (p *turtleParser) parseVerbTokens(stream *turtleTokenStream) (IRI, error) {
	tok := stream.peek()
	if tok.Kind == TokA {
		stream.next()
		return IRI{Value: rdfTypeIRI}, nil
	}
	term, err := p.parseTermTokens(stream, false)
	if err != nil {
		return IRI{}, err
	}
	iri, ok := term.(IRI)
	if !ok {
		return IRI{}, WrapParseError("turtle", "", -1, fmt.Errorf("predicate must be IRI, got %T", term))
	}
	return iri, nil
}

func (p *turtleParser) parseObjectListTokens(stream *turtleTokenStream, subject Term, predicate IRI) ([]Triple, error) {
	var triples []Triple
	for {
		obj, err := p.parseTermTokens(stream, true)
		if err != nil {
			return nil, err
		}
		triples = append(triples, Triple{S: subject, P: predicate, O: obj})

		// Handle annotations after object: {| ... |}
		if stream.peek().Kind == TokAnnotationL {
			annotationTriples, err := p.parseAnnotationTokens(stream, obj)
			if err != nil {
				return nil, err
			}
			triples = append(triples, annotationTriples...)
		}

		if stream.peek().Kind == TokComma {
			stream.next()
			continue
		}
		return triples, nil
	}
}

func (p *turtleParser) parseTermTokens(stream *turtleTokenStream, allowLiteral bool) (Term, error) {
	tok := stream.peek()
	switch tok.Kind {
	case TokIRIRef:
		stream.next()
		iri := strings.Trim(tok.Lexeme, "<>")
		if p.baseIRI != "" {
			iri = resolveIRI(p.baseIRI, iri)
		}
		return IRI{Value: iri}, nil
	case TokPNAMENS:
		stream.next()
		prefix := strings.TrimSuffix(tok.Lexeme, ":")
		base, ok := p.prefixes[prefix]
		if !ok {
			return nil, WrapParseError("turtle", "", -1, fmt.Errorf("undefined prefix: %s", prefix))
		}
		return IRI{Value: base}, nil
	case TokPNAMELN:
		stream.next()
		parts := strings.SplitN(tok.Lexeme, ":", 2)
		if len(parts) != 2 {
			return nil, WrapParseError("turtle", "", -1, fmt.Errorf("invalid prefixed name: %s", tok.Lexeme))
		}
		base, ok := p.prefixes[parts[0]]
		if !ok {
			return nil, WrapParseError("turtle", "", -1, fmt.Errorf("undefined prefix: %s", parts[0]))
		}
		return IRI{Value: base + parts[1]}, nil
	case TokBlankNode:
		stream.next()
		return BlankNode{ID: tok.Lexeme[2:]}, nil // Skip "_:"
	case TokString, TokStringLong:
		return p.parseLiteralTokens(stream, allowLiteral)
	case TokInteger, TokDecimal, TokDouble, TokBoolean:
		stream.next()
		return p.parseTermFromLexeme(tok.Lexeme, allowLiteral)
	case TokLBracket:
		return p.parseBlankNodePropertyListTokens(stream)
	case TokLParen:
		return p.parseCollectionTokens(stream)
	case TokLDoubleAngle:
		return p.parseTripleTermTokens(stream)
	default:
		return nil, WrapParseError("turtle", "", -1, fmt.Errorf("unexpected token: %v", tok.Kind))
	}
}

func (p *turtleParser) parseLiteralTokens(stream *turtleTokenStream, allowLiteral bool) (Term, error) {
	if !allowLiteral {
		return nil, WrapParseError("turtle", "", -1, fmt.Errorf("literal not allowed here"))
	}
	tok := stream.next()
	lexeme := tok.Lexeme

	// Parse the string value (remove quotes)
	var lexical string
	if tok.Kind == TokStringLong {
		// Long string: """...""" or '''...'''
		if len(lexeme) < 6 {
			return nil, WrapParseError("turtle", "", -1, fmt.Errorf("invalid long string"))
		}
		quote := lexeme[0]
		lexical = lexeme[3 : len(lexeme)-3] // Remove triple quotes
		// Unescape the string
		var err error
		lexical, err = p.unescapeString(lexical, quote)
		if err != nil {
			return nil, WrapParseError("turtle", "", -1, err)
		}
	} else {
		// Regular string: "..." or '...'
		if len(lexeme) < 2 {
			return nil, WrapParseError("turtle", "", -1, fmt.Errorf("invalid string"))
		}
		quote := lexeme[0]
		lexical = lexeme[1 : len(lexeme)-1] // Remove quotes
		// Unescape the string
		var err error
		lexical, err = p.unescapeString(lexical, quote)
		if err != nil {
			return nil, WrapParseError("turtle", "", -1, err)
		}
	}

	// Check for language tag or datatype
	next := stream.peek()
	if next.Kind == TokLangTag {
		stream.next()
		lang := next.Lexeme
		if !isValidLangTag(lang) {
			return nil, WrapParseError("turtle", "", -1, fmt.Errorf("invalid language tag: %s", lang))
		}
		// Check that there's no datatype after lang tag
		if stream.peek().Kind == TokDatatypePrefix {
			return nil, WrapParseError("turtle", "", -1, fmt.Errorf("literal cannot have both language tag and datatype"))
		}
		return Literal{Lexical: lexical, Lang: lang}, nil
	}
	if next.Kind == TokDatatypePrefix {
		stream.next()
		// Parse the datatype (should be an IRI or prefixed name)
		dtTerm, err := p.parseTermTokens(stream, false)
		if err != nil {
			return nil, err
		}
		iri, ok := dtTerm.(IRI)
		if !ok {
			return nil, WrapParseError("turtle", "", -1, fmt.Errorf("datatype must be IRI"))
		}
		return Literal{Lexical: lexical, Datatype: iri}, nil
	}
	return Literal{Lexical: lexical}, nil
}

func (p *turtleParser) parseCollectionTokens(stream *turtleTokenStream) (Term, error) {
	// Consume LParen
	if stream.next().Kind != TokLParen {
		return nil, WrapParseError("turtle", "", -1, fmt.Errorf("expected '('"))
	}

	// Check for empty collection
	if stream.peek().Kind == TokRParen {
		stream.next()
		return IRI{Value: "http://www.w3.org/1999/02/22-rdf-syntax-ns#nil"}, nil
	}

	var objects []Term
	for {
		if stream.peek().Kind == TokRParen {
			stream.next()
			break
		}
		obj, err := p.parseTermTokens(stream, true)
		if err != nil {
			return nil, err
		}
		objects = append(objects, obj)
		if stream.peek().Kind == TokRParen {
			stream.next()
			break
		}
	}

	if len(objects) == 0 {
		return IRI{Value: "http://www.w3.org/1999/02/22-rdf-syntax-ns#nil"}, nil
	}

	// Generate rdf:first/rdf:rest triples
	head := p.newBlankNode()
	rdfFirst := IRI{Value: "http://www.w3.org/1999/02/22-rdf-syntax-ns#first"}
	rdfRest := IRI{Value: "http://www.w3.org/1999/02/22-rdf-syntax-ns#rest"}
	rdfNil := IRI{Value: "http://www.w3.org/1999/02/22-rdf-syntax-ns#nil"}

	current := head
	for i, obj := range objects {
		// rdf:first triple
		p.expansionTriples = append(p.expansionTriples, Triple{
			S: current,
			P: rdfFirst,
			O: obj,
		})

		// rdf:rest triple
		var rest Term
		if i == len(objects)-1 {
			rest = rdfNil
		} else {
			rest = p.newBlankNode()
		}
		p.expansionTriples = append(p.expansionTriples, Triple{
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

func (p *turtleParser) parseBlankNodePropertyListTokens(stream *turtleTokenStream) (Term, error) {
	// Consume LBracket
	if stream.next().Kind != TokLBracket {
		return nil, WrapParseError("turtle", "", -1, fmt.Errorf("expected '['"))
	}

	// Check for empty blank node property list
	if stream.peek().Kind == TokRBracket {
		stream.next()
		return p.newBlankNode(), nil
	}

	bn := p.newBlankNode()

	// Parse predicateObjectList
	for {
		// Parse predicate
		predicate, err := p.parseVerbTokens(stream)
		if err != nil {
			return nil, err
		}

		// Parse objectList
		for {
			object, err := p.parseTermTokens(stream, true)
			if err != nil {
				return nil, err
			}
			p.expansionTriples = append(p.expansionTriples, Triple{
				S: bn,
				P: predicate,
				O: object,
			})

			if stream.peek().Kind == TokComma {
				stream.next()
				continue
			}
			if stream.peek().Kind == TokRBracket {
				stream.next()
				return bn, nil
			}
			break
		}

		// Check for semicolon or closing bracket
		if stream.peek().Kind == TokRBracket {
			stream.next()
			return bn, nil
		}
		if stream.peek().Kind == TokSemicolon {
			for stream.peek().Kind == TokSemicolon {
				stream.next()
			}
			if stream.peek().Kind == TokRBracket {
				stream.next()
				return bn, nil
			}
			continue
		}
		return nil, WrapParseError("turtle", "", -1, fmt.Errorf("expected ',' or ';' or ']'"))
	}
}

func (p *turtleParser) parseTripleTermTokens(stream *turtleTokenStream) (Term, error) {
	// Consume LDoubleAngle
	if stream.next().Kind != TokLDoubleAngle {
		return nil, WrapParseError("turtle", "", -1, fmt.Errorf("expected '<<'"))
	}

	// Check for optional parens: <<( ... )>>
	hasParens := false
	if stream.peek().Kind == TokLParen {
		hasParens = true
		stream.next()
	}

	// Parse subject
	subject, err := p.parseTermTokens(stream, false)
	if err != nil {
		return nil, err
	}

	// Parse predicate
	predicate, err := p.parseVerbTokens(stream)
	if err != nil {
		return nil, err
	}

	// Parse object
	object, err := p.parseTermTokens(stream, true)
	if err != nil {
		return nil, err
	}

	if hasParens {
		if stream.peek().Kind != TokRParen {
			return nil, WrapParseError("turtle", "", -1, fmt.Errorf("expected ')'"))
		}
		stream.next()
	}

	// Handle optional reifier: << ... ~ ... >>
	// Note: "~" is not tokenized by the lexer, so we need to check the raw input
	// For now, we'll skip reifier support in token-based parsing and fall back to cursor
	// This is a limitation that can be addressed by extending the lexer

	if stream.peek().Kind != TokRDoubleAngle {
		return nil, WrapParseError("turtle", "", -1, fmt.Errorf("expected '>>'"))
	}
	stream.next()
	return TripleTerm{S: subject, P: predicate, O: object}, nil
}

func (p *turtleParser) unescapeString(s string, quoteChar byte) (string, error) {
	var builder strings.Builder
	pos := 0
	for pos < len(s) {
		ch := s[pos]
		if ch == '\\' {
			if pos+1 >= len(s) {
				return "", fmt.Errorf("unterminated escape")
			}
			next := s[pos+1]
			switch next {
			case 'n':
				builder.WriteByte('\n')
				pos += 2
			case 't':
				builder.WriteByte('\t')
				pos += 2
			case 'r':
				builder.WriteByte('\r')
				pos += 2
			case 'b':
				builder.WriteByte('\b')
				pos += 2
			case 'f':
				builder.WriteByte('\f')
				pos += 2
			case '"':
				builder.WriteByte('"')
				pos += 2
			case '\'':
				builder.WriteByte('\'')
				pos += 2
			case '\\':
				builder.WriteByte('\\')
				pos += 2
			case 'u':
				// Unicode escape \uXXXX
				if pos+5 >= len(s) {
					return "", fmt.Errorf("invalid escape sequence")
				}
				codePoint := decodeUChar(s[pos+2 : pos+6])
				if codePoint < 0 {
					return "", fmt.Errorf("invalid escape sequence")
				}
				if codePoint >= 0xD800 && codePoint <= 0xDBFF {
					// Surrogate pair
					if pos+11 >= len(s) || s[pos+6] != '\\' || s[pos+7] != 'u' {
						return "", fmt.Errorf("invalid escape sequence")
					}
					low := decodeUChar(s[pos+8 : pos+12])
					if low < 0 || low < 0xDC00 || low > 0xDFFF {
						return "", fmt.Errorf("invalid escape sequence")
					}
					combined := 0x10000 + ((codePoint - 0xD800) << 10) + (low - 0xDC00)
					if !isValidUnicodeCodePoint(rune(combined)) {
						return "", fmt.Errorf("invalid escape sequence")
					}
					builder.WriteRune(rune(combined))
					pos += 12
					continue
				}
				if codePoint >= 0xDC00 && codePoint <= 0xDFFF {
					return "", fmt.Errorf("invalid escape sequence")
				}
				if !isValidUnicodeCodePoint(codePoint) {
					return "", fmt.Errorf("invalid escape sequence")
				}
				builder.WriteRune(codePoint)
				pos += 6
			case 'U':
				// Unicode escape \UXXXXXXXX
				if pos+9 >= len(s) {
					return "", fmt.Errorf("invalid escape sequence")
				}
				var codePoint rune
				for i := 2; i < 10; i++ {
					hex := s[pos+i]
					var digit int
					if hex >= '0' && hex <= '9' {
						digit = int(hex - '0')
					} else if hex >= 'a' && hex <= 'f' {
						digit = int(hex - 'a' + 10)
					} else if hex >= 'A' && hex <= 'F' {
						digit = int(hex - 'A' + 10)
					} else {
						return "", fmt.Errorf("invalid escape sequence")
					}
					codePoint = codePoint*16 + rune(digit)
				}
				if !isValidUnicodeCodePoint(codePoint) {
					return "", fmt.Errorf("invalid escape sequence")
				}
				builder.WriteRune(codePoint)
				pos += 10
			default:
				return "", fmt.Errorf("invalid escape sequence")
			}
			continue
		}
		builder.WriteByte(ch)
		pos++
	}
	return builder.String(), nil
}

func (p *turtleParser) parseTermFromLexeme(lexeme string, allowLiteral bool) (Term, error) {
	cursor := &turtleCursor{
		input:                      lexeme,
		prefixes:                   p.prefixes,
		base:                       p.baseIRI,
		allowQuotedTripleStatement: p.allowQuotedTripleStatement,
	}
	term, err := cursor.parseTerm(allowLiteral)
	if err != nil {
		return nil, err
	}
	cursor.skipWS()
	if cursor.pos != len(cursor.input) {
		return nil, cursor.errorf("unexpected trailing input")
	}
	return term, nil
}

func (p *turtleParser) parseAnnotationTokens(stream *turtleTokenStream, annotationSubject Term) ([]Triple, error) {
	// Consume AnnotationL {|
	if stream.next().Kind != TokAnnotationL {
		return nil, WrapParseError("turtle", "", -1, fmt.Errorf("expected '{|'"))
	}

	var annotationTriples []Triple

	// Parse predicateObjectList
	for {
		// Parse predicate
		pred, err := p.parseVerbTokens(stream)
		if err != nil {
			return nil, err
		}

		// Parse objectList
		for {
			obj, err := p.parseTermTokens(stream, true)
			if err != nil {
				return nil, err
			}
			annotationTriples = append(annotationTriples, Triple{
				S: annotationSubject,
				P: pred,
				O: obj,
			})

			// Handle nested annotations
			if stream.peek().Kind == TokAnnotationL {
				nestedTriples, err := p.parseAnnotationTokens(stream, obj)
				if err != nil {
					return nil, err
				}
				annotationTriples = append(annotationTriples, nestedTriples...)
			}

			if stream.peek().Kind == TokComma {
				stream.next()
				continue
			}
			break
		}

		// Check for semicolon or closing annotation
		if stream.peek().Kind == TokSemicolon {
			for stream.peek().Kind == TokSemicolon {
				stream.next()
			}
			if stream.peek().Kind == TokAnnotationR {
				stream.next()
				break
			}
			continue
		}
		if stream.peek().Kind == TokAnnotationR {
			stream.next()
			break
		}
		return nil, WrapParseError("turtle", "", -1, fmt.Errorf("expected ',' or ';' or '|}'"))
	}

	return annotationTriples, nil
}

type turtleTokenStream struct {
	tokens []turtleToken
	pos    int
}

func (s *turtleTokenStream) peek() turtleToken {
	if s.pos >= len(s.tokens) {
		return turtleToken{Kind: TokEOF}
	}
	return s.tokens[s.pos]
}

func (s *turtleTokenStream) next() turtleToken {
	tok := s.peek()
	if s.pos < len(s.tokens) {
		s.pos++
	}
	return tok
}
