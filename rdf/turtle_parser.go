package rdf

import (
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
	}
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
				// try to parse what we have so far
				goto parseStatement
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
					goto parseStatement
				}
			}
		}
	parseStatement:
		line := strings.TrimSpace(statement.String())
		if line == "" {
			continue
		}
		tokens, err := tokenizeTurtleLine(line)
		if err != nil {
			p.err = err
			return Triple{}, err
		}
		if handled, err := p.parseDirectiveTokens(tokens); err != nil {
			p.err = err
			return Triple{}, err
		} else if handled {
			continue
		}
		triples, err := p.parseTriplesTokens(tokens, line)
		if err != nil {
			p.err = err
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
	if p.needsCursorFallback(tokens) {
		return p.parseTripleLine(line)
	}
	stream := &turtleTokenStream{tokens: tokens}
	subject, err := p.parseSubjectTokens(stream)
	if err != nil {
		return nil, err
	}
	triples, err := p.parsePredicateObjectListTokens(stream, subject)
	if err != nil {
		return nil, err
	}
	if stream.peek().Kind == TokDot {
		stream.next()
	}
	if stream.peek().Kind != TokEOF {
		return nil, WrapParseError("turtle", line, -1, io.ErrUnexpectedEOF)
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
		return IRI{}, WrapParseError("turtle", "", -1, io.ErrUnexpectedEOF)
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
	case TokIRIRef, TokPNAMENS, TokPNAMELN, TokBlankNode, TokString, TokStringLong, TokInteger, TokDecimal, TokDouble, TokBoolean:
		stream.next()
		return p.parseTermFromLexeme(tok.Lexeme, allowLiteral)
	default:
		return nil, WrapParseError("turtle", "", -1, io.ErrUnexpectedEOF)
	}
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

func (p *turtleParser) needsCursorFallback(tokens []turtleToken) bool {
	for _, tok := range tokens {
		switch tok.Kind {
		case TokLBracket, TokLParen, TokLDoubleAngle, TokAnnotationL, TokAnnotationR, TokLBrace, TokRBrace, TokError:
			return true
		}
		if strings.HasPrefix(tok.Lexeme, "@") || strings.HasPrefix(tok.Lexeme, "^^") {
			return true
		}
	}
	return false
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
