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
	tripleCount                int64 // Number of triples processed
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
		// blankNodeCounter uses zero value (0)
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

	triple, err := p.readNextTriple()
	if err != nil {
		p.err = err
	}
	return triple, err
}

// readNextTriple reads and parses the next triple from the input stream.
func (p *turtleParser) readNextTriple() (Triple, error) {
	for {
		if err := checkDecodeContext(p.opts.Context); err != nil {
			return Triple{}, err
		}

		statement, err := p.readStatementLines()
		if err != nil {
			return Triple{}, err
		}

		if statement == "" {
			return Triple{}, io.EOF
		}

		triples, err := p.parseStatement(statement)
		if err != nil {
			return Triple{}, err
		}

		if len(triples) == 0 {
			continue
		}

		if len(triples) > 1 {
			p.pending = triples[1:]
		}
		return triples[0], nil
	}
}

// readStatementLines reads lines from the lexer and builds a complete statement.
func (p *turtleParser) readStatementLines() (string, error) {
	var statement strings.Builder
	for {
		if err := checkDecodeContext(p.opts.Context); err != nil {
			return "", err
		}

		token := p.lexer.Next()
		switch token.Kind {
		case TokEOF:
			if statement.Len() == 0 {
				return "", io.EOF
			}
			return strings.TrimSpace(statement.String()), nil

		case TokError:
			return "", token.Err

		case TokLine:
			// Quick check: if line looks like a directive, tokenize and parse it
			if statement.Len() == 0 && p.isLikelyDirective(token.Lexeme) {
				tokens, err := tokenizeTurtleLine(token.Lexeme)
				if err == nil {
					if handled, _ := p.parseDirectiveTokens(tokens); handled {
						continue
					}
				}
			}

			if err := p.appendStatementPart(&statement, token.Lexeme); err != nil {
				return "", err
			}

			stmt := strings.TrimSpace(statement.String())
			if stmt != "" && isStatementComplete(stmt) {
				return stmt, nil
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

// isLikelyDirective performs a quick string-based check to see if a line might be a directive.
// This is used for early detection before tokenization. The actual parsing is done by parseDirectiveTokens.
func (p *turtleParser) isLikelyDirective(line string) bool {
	line = strings.TrimSpace(line)
	if len(line) == 0 {
		return false
	}
	// Check for @-prefixed directives first (most common)
	if line[0] == '@' {
		return strings.HasPrefix(line, directiveAtPrefix) ||
			strings.HasPrefix(line, directiveAtBase) ||
			strings.HasPrefix(line, directiveAtVersion)
	}
	// Check for bare directives (case-insensitive)
	upper := strings.ToUpper(line)
	return strings.HasPrefix(upper, directivePrefix) ||
		strings.HasPrefix(upper, directiveBase) ||
		strings.HasPrefix(upper, directiveVersion)
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
	// Reset for next statement - if capacity is large, release it to prevent memory bloat
	const maxExpansionTriplesCapacity = 1024
	if cap(p.expansionTriples) > maxExpansionTriplesCapacity {
		p.expansionTriples = nil
	} else {
		p.expansionTriples = p.expansionTriples[:0]
	}
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

		next := stream.peek().Kind
		if next == TokSemicolon {
			p.skipSemicolons(stream)
			if stream.peek().Kind == TokDot || stream.peek().Kind == TokEOF {
				return triples, nil
			}
			continue
		}
		return triples, nil
	}
}

// skipSemicolons consumes one or more semicolon tokens.
func (p *turtleParser) skipSemicolons(stream *turtleTokenStream) {
	for stream.peek().Kind == TokSemicolon {
		stream.next()
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
		return IRI{}, p.wrapParseError("", fmt.Errorf("predicate must be IRI, got %T", term))
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
	return p.parseTermTokensWithDepth(stream, allowLiteral, 0)
}

func (p *turtleParser) parseTermTokensWithDepth(stream *turtleTokenStream, allowLiteral bool, depth int) (Term, error) {
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
			return nil, p.wrapParseError("", fmt.Errorf("undefined prefix: %s", prefix))
		}
		return IRI{Value: base}, nil
	case TokPNAMELN:
		stream.next()
		parts := strings.SplitN(tok.Lexeme, ":", 2)
		if len(parts) != 2 {
			return nil, p.wrapParseError("", fmt.Errorf("invalid prefixed name: %s", tok.Lexeme))
		}
		base, ok := p.prefixes[parts[0]]
		if !ok {
			return nil, p.wrapParseError("", fmt.Errorf("undefined prefix: %s", parts[0]))
		}
		return IRI{Value: base + parts[1]}, nil
	case TokBlankNode:
		stream.next()
		return BlankNode{ID: tok.Lexeme[2:]}, nil // Skip "_:"
	case TokString, TokStringLong:
		return p.parseLiteralTokens(stream, allowLiteral)
	case TokInteger, TokDecimal, TokDouble:
		stream.next()
		return p.parseNumericLiteralToken(tok)
	case TokBoolean:
		stream.next()
		return p.parseBooleanLiteralToken(tok)
	case TokLBracket:
		return p.parseBlankNodePropertyListTokens(stream, depth)
	case TokLParen:
		return p.parseCollectionTokens(stream, depth)
	case TokLDoubleAngle:
		return p.parseTripleTermTokens(stream)
	case TokLangTag:
		// Language tags should only appear after string literals, not as standalone tokens
		return nil, p.wrapParseError("", fmt.Errorf("unexpected language tag (must follow a string literal)"))
	case TokDatatypePrefix:
		// Datatype prefix should only appear after string literals, not as standalone tokens
		return nil, p.wrapParseError("", fmt.Errorf("unexpected datatype prefix (must follow a string literal)"))
	default:
		return nil, p.wrapParseError("", fmt.Errorf("unexpected token: %v", tok.Kind))
	}
}

func (p *turtleParser) parseLiteralTokens(stream *turtleTokenStream, allowLiteral bool) (Term, error) {
	if !allowLiteral {
		return nil, p.wrapParseError("", fmt.Errorf("literal not allowed here"))
	}
	tok := stream.next()
	lexeme := tok.Lexeme

	// Parse the string value (remove quotes)
	var lexical string
	if tok.Kind == TokStringLong {
		// Long string: """...""" or '''...'''
		if len(lexeme) < minLongStringLength {
			return nil, p.wrapParseError("", fmt.Errorf("invalid long string"))
		}
		lexical = lexeme[3 : len(lexeme)-3] // Remove triple quotes
		// Unescape the string
		var err error
		lexical, err = UnescapeString(lexical)
		if err != nil {
			return nil, p.wrapParseError("", err)
		}
	} else {
		// Regular string: "..." or '...'
		if len(lexeme) < minStringLength {
			return nil, p.wrapParseError("", fmt.Errorf("invalid string"))
		}
		lexical = lexeme[1 : len(lexeme)-1] // Remove quotes
		// Unescape the string
		var err error
		lexical, err = UnescapeString(lexical)
		if err != nil {
			return nil, p.wrapParseError("", err)
		}
	}

	// Check for language tag or datatype
	next := stream.peek()
	if next.Kind == TokLangTag {
		stream.next()
		lang := next.Lexeme
		if !isValidLangTag(lang) {
			return nil, p.wrapParseError("", fmt.Errorf("invalid language tag: %s", lang))
		}
		// Check that there's no datatype after lang tag
		if stream.peek().Kind == TokDatatypePrefix {
			return nil, p.wrapParseError("", fmt.Errorf("literal cannot have both language tag and datatype"))
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
			return nil, p.wrapParseError("", fmt.Errorf("datatype must be IRI"))
		}
		return Literal{Lexical: lexical, Datatype: iri}, nil
	}
	return Literal{Lexical: lexical}, nil
}

func (p *turtleParser) parseCollectionTokens(stream *turtleTokenStream, depth int) (Term, error) {
	// Check depth limit
	if p.opts.MaxDepth > 0 && depth >= p.opts.MaxDepth {
		return nil, p.wrapParseError("", ErrDepthExceeded)
	}
	// Consume LParen
	if stream.next().Kind != TokLParen {
		return nil, p.wrapParseError("", fmt.Errorf("expected '('"))
	}

	// Check for empty collection
	if stream.peek().Kind == TokRParen {
		stream.next()
		return IRI{Value: rdfNilIRI}, nil
	}

	var objects []Term
	for {
		if stream.peek().Kind == TokRParen {
			stream.next()
			break
		}
		obj, err := p.parseTermTokensWithDepth(stream, true, depth+1)
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
		return IRI{Value: rdfNilIRI}, nil
	}

	// Generate rdf:first/rdf:rest triples
	head := generateCollectionTriples(objects, &p.expansionTriples, p.newBlankNode)
	return head, nil
}

func (p *turtleParser) parseBlankNodePropertyListTokens(stream *turtleTokenStream, depth int) (Term, error) {
	// Check depth limit
	if p.opts.MaxDepth > 0 && depth >= p.opts.MaxDepth {
		return nil, p.wrapParseError("", ErrDepthExceeded)
	}
	// Consume LBracket
	if stream.next().Kind != TokLBracket {
		return nil, p.wrapParseError("", fmt.Errorf("expected '['"))
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
			object, err := p.parseTermTokensWithDepth(stream, true, depth+1)
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
		next := stream.peek().Kind
		if next == TokRBracket {
			stream.next()
			return bn, nil
		}
		if next == TokSemicolon {
			p.skipSemicolons(stream)
			if stream.peek().Kind == TokRBracket {
				stream.next()
				return bn, nil
			}
			continue
		}
		return nil, p.wrapParseError("", fmt.Errorf("expected ',' or ';' or ']'"))
	}
}

func (p *turtleParser) parseTripleTermTokens(stream *turtleTokenStream) (Term, error) {
	// Consume LDoubleAngle
	if stream.next().Kind != TokLDoubleAngle {
		return nil, p.wrapParseError("", fmt.Errorf("expected '<<'"))
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
			return nil, p.wrapParseError("", fmt.Errorf("expected ')'"))
		}
		stream.next()
	}

	// Handle optional reifier: << ... ~ ... >>
	// Note: "~" is not tokenized by the lexer, so we need to check the raw input
	// For now, we'll skip reifier support in token-based parsing and fall back to cursor
	// This is a limitation that can be addressed by extending the lexer

	if stream.peek().Kind != TokRDoubleAngle {
		return nil, p.wrapParseError("", fmt.Errorf("expected '>>'"))
	}
	stream.next()
	return TripleTerm{S: subject, P: predicate, O: object}, nil
}

func (p *turtleParser) parseNumericLiteralToken(tok turtleToken) (Term, error) {
	lexical := tok.Lexeme
	var datatype IRI

	// Determine datatype based on token kind
	switch tok.Kind {
	case TokDouble:
		datatype = IRI{Value: "http://www.w3.org/2001/XMLSchema#double"}
	case TokDecimal:
		datatype = IRI{Value: "http://www.w3.org/2001/XMLSchema#decimal"}
	case TokInteger:
		datatype = IRI{Value: "http://www.w3.org/2001/XMLSchema#integer"}
	default:
		// Fallback: determine from lexical form
		if strings.Contains(lexical, "e") || strings.Contains(lexical, "E") {
			datatype = IRI{Value: "http://www.w3.org/2001/XMLSchema#double"}
		} else if strings.Contains(lexical, ".") {
			datatype = IRI{Value: "http://www.w3.org/2001/XMLSchema#decimal"}
		} else {
			datatype = IRI{Value: "http://www.w3.org/2001/XMLSchema#integer"}
		}
	}

	return Literal{Lexical: lexical, Datatype: datatype}, nil
}

func (p *turtleParser) parseBooleanLiteralToken(tok turtleToken) (Term, error) {
	lexical := tok.Lexeme
	return Literal{
		Lexical:  lexical,
		Datatype: IRI{Value: "http://www.w3.org/2001/XMLSchema#boolean"},
	}, nil
}

func (p *turtleParser) parseAnnotationTokens(stream *turtleTokenStream, annotationSubject Term) ([]Triple, error) {
	// Consume AnnotationL {|
	if stream.next().Kind != TokAnnotationL {
		return nil, p.wrapParseError("", fmt.Errorf("expected '{|'"))
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
		next := stream.peek().Kind
		if next == TokSemicolon {
			p.skipSemicolons(stream)
			if stream.peek().Kind == TokAnnotationR {
				stream.next()
				break
			}
			continue
		}
		if next == TokAnnotationR {
			stream.next()
			break
		}
		return nil, p.wrapParseError("", fmt.Errorf("expected ',' or ';' or '|}'"))
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
