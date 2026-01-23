package rdf

import (
	"bufio"
	"io"
	"strings"
)

type turtleTokenKind int

const (
	TokLine turtleTokenKind = iota
	TokEOF
	TokError
	// Future token kinds for full lexical scanning.
	TokIRIRef
	TokPNAMENS
	TokPNAMELN
	TokBlankNode
	TokString
	TokStringLong
	TokInteger
	TokDecimal
	TokDouble
	TokBoolean
	TokPrefix
	TokBase
	TokVersion
	TokDot
	TokComma
	TokSemicolon
	TokLBracket
	TokRBracket
	TokLParen
	TokRParen
	TokLBrace
	TokRBrace
	TokLDoubleAngle
	TokRDoubleAngle
	TokAnnotationL
	TokAnnotationR
	TokA
	TokLangTag
	TokDatatypePrefix
)

const (
	lexLDoubleAngle = "<<"
	lexRDoubleAngle = ">>"
	lexAnnotationL  = "{|"
	lexAnnotationR  = "|}"
	lexDot          = "."
	lexComma        = ","
	lexSemicolon    = ";"
	lexLBracket     = "["
	lexRBracket     = "]"
	lexLParen       = "("
	lexRParen       = ")"
	lexLBrace       = "{"
	lexRBrace       = "}"
	lexIRIStart     = "<"
	lexIRIEnd       = ">"
	lexBlankNode    = "_:"
	lexQuote        = "\""
	lexApos         = "'"
	lexPrefix       = "@prefix"
	lexBase         = "@base"
	lexVersion      = "@version"
	lexPrefixBare   = "prefix"
	lexBaseBare     = "base"
	lexVersionBare  = "version"
)

func (k turtleTokenKind) String() string {
	switch k {
	case TokLine:
		return "TokLine"
	case TokEOF:
		return "TokEOF"
	case TokError:
		return "TokError"
	case TokIRIRef:
		return "TokIRIRef"
	case TokPNAMENS:
		return "TokPNAMENS"
	case TokPNAMELN:
		return "TokPNAMELN"
	case TokBlankNode:
		return "TokBlankNode"
	case TokString:
		return "TokString"
	case TokStringLong:
		return "TokStringLong"
	case TokInteger:
		return "TokInteger"
	case TokDecimal:
		return "TokDecimal"
	case TokDouble:
		return "TokDouble"
	case TokBoolean:
		return "TokBoolean"
	case TokPrefix:
		return "TokPrefix"
	case TokBase:
		return "TokBase"
	case TokVersion:
		return "TokVersion"
	case TokDot:
		return "TokDot"
	case TokComma:
		return "TokComma"
	case TokSemicolon:
		return "TokSemicolon"
	case TokLBracket:
		return "TokLBracket"
	case TokRBracket:
		return "TokRBracket"
	case TokLParen:
		return "TokLParen"
	case TokRParen:
		return "TokRParen"
	case TokLBrace:
		return "TokLBrace"
	case TokRBrace:
		return "TokRBrace"
	case TokLDoubleAngle:
		return "TokLDoubleAngle"
	case TokRDoubleAngle:
		return "TokRDoubleAngle"
	case TokAnnotationL:
		return "TokAnnotationL"
	case TokAnnotationR:
		return "TokAnnotationR"
	case TokA:
		return "TokA"
	case TokLangTag:
		return "TokLangTag"
	case TokDatatypePrefix:
		return "TokDatatypePrefix"
	default:
		return "TokUnknown"
	}
}

type turtleToken struct {
	Kind   turtleTokenKind
	Lexeme string
	Err    error
}

type turtleLexer struct {
	reader *bufio.Reader
	opts   DecodeOptions
}

func newTurtleLexer(r io.Reader, opts DecodeOptions) *turtleLexer {
	return &turtleLexer{
		reader: bufio.NewReader(r),
		opts:   normalizeDecodeOptions(opts),
	}
}

func (l *turtleLexer) Next() turtleToken {
	for {
		line, err := readLineWithLimit(l.reader, l.opts.MaxLineBytes)
		if err != nil {
			if err == io.EOF && line == "" {
				return turtleToken{Kind: TokEOF}
			}
			if err == io.EOF && line != "" {
				trimmed := strings.TrimSpace(stripComment(line))
				if trimmed == "" {
					return turtleToken{Kind: TokEOF}
				}
				return turtleToken{Kind: TokLine, Lexeme: trimmed}
			}
			return turtleToken{Kind: TokError, Err: err}
		}
		trimmed := strings.TrimSpace(stripComment(line))
		if trimmed == "" {
			continue
		}
		return turtleToken{Kind: TokLine, Lexeme: trimmed}
	}
}

// tokenizeTurtleLine splits a single Turtle statement line into tokens.
// It is intentionally conservative and defers semantic validation to the parser.
func tokenizeTurtleLine(line string) ([]turtleToken, error) {
	scanner := &turtleScanner{input: line}
	var tokens []turtleToken
	for {
		tok, err := scanner.nextToken()
		if err != nil {
			return nil, err
		}
		if tok.Kind == TokEOF {
			return tokens, nil
		}
		tokens = append(tokens, tok)
	}
}

type turtleScanner struct {
	input string
	pos   int
}

func (s *turtleScanner) nextToken() (turtleToken, error) {
	s.skipWS()
	if s.pos >= len(s.input) {
		return turtleToken{Kind: TokEOF}, nil
	}
	if s.match(lexLDoubleAngle) {
		s.pos += 2
		return turtleToken{Kind: TokLDoubleAngle, Lexeme: lexLDoubleAngle}, nil
	}
	if s.match(lexRDoubleAngle) {
		s.pos += 2
		return turtleToken{Kind: TokRDoubleAngle, Lexeme: lexRDoubleAngle}, nil
	}
	if s.match(lexAnnotationL) {
		s.pos += 2
		return turtleToken{Kind: TokAnnotationL, Lexeme: lexAnnotationL}, nil
	}
	if s.match(lexAnnotationR) {
		s.pos += 2
		return turtleToken{Kind: TokAnnotationR, Lexeme: lexAnnotationR}, nil
	}
	if s.pos+1 < len(s.input) && s.input[s.pos] == '^' && s.input[s.pos+1] == '^' {
		s.pos += 2
		return turtleToken{Kind: TokDatatypePrefix, Lexeme: "^^"}, nil
	}
	// Check for @ but only scan as language tag if it's not a directive
	if s.input[s.pos] == '@' {
		// Check if it's a directive (@prefix, @base, @version) - these are handled by scanWord
		remaining := s.input[s.pos:]
		if strings.HasPrefix(remaining, "@prefix") || strings.HasPrefix(remaining, "@base") || strings.HasPrefix(remaining, "@version") {
			// Let scanWord handle it
		} else {
			// It's likely a language tag
			return s.scanLangTag()
		}
	}
	switch s.input[s.pos] {
	case lexDot[0]:
		s.pos++
		return turtleToken{Kind: TokDot, Lexeme: lexDot}, nil
	case lexComma[0]:
		s.pos++
		return turtleToken{Kind: TokComma, Lexeme: lexComma}, nil
	case lexSemicolon[0]:
		s.pos++
		return turtleToken{Kind: TokSemicolon, Lexeme: lexSemicolon}, nil
	case lexLBracket[0]:
		s.pos++
		return turtleToken{Kind: TokLBracket, Lexeme: lexLBracket}, nil
	case lexRBracket[0]:
		s.pos++
		return turtleToken{Kind: TokRBracket, Lexeme: lexRBracket}, nil
	case lexLParen[0]:
		s.pos++
		return turtleToken{Kind: TokLParen, Lexeme: lexLParen}, nil
	case lexRParen[0]:
		s.pos++
		return turtleToken{Kind: TokRParen, Lexeme: lexRParen}, nil
	case lexLBrace[0]:
		s.pos++
		return turtleToken{Kind: TokLBrace, Lexeme: lexLBrace}, nil
	case lexRBrace[0]:
		s.pos++
		return turtleToken{Kind: TokRBrace, Lexeme: lexRBrace}, nil
	case lexIRIStart[0]:
		return s.scanIRIRef()
	case lexQuote[0], lexApos[0]:
		return s.scanString()
	}
	if s.match(lexBlankNode) {
		return s.scanBlankNode()
	}
	return s.scanWord()
}

func (s *turtleScanner) scanIRIRef() (turtleToken, error) {
	start := s.pos
	s.pos++ // consume '<'
	for s.pos < len(s.input) && s.input[s.pos] != '>' {
		if s.input[s.pos] == '\\' {
			// Skip escape sequence, parser will validate if needed.
			if s.pos+1 < len(s.input) && (s.input[s.pos+1] == 'u' || s.input[s.pos+1] == 'U') {
				s.pos += 2
				continue
			}
		}
		s.pos++
	}
	if s.pos >= len(s.input) {
		return turtleToken{}, ErrStatementTooLong
	}
	s.pos++ // consume '>'
	return turtleToken{Kind: TokIRIRef, Lexeme: s.input[start:s.pos]}, nil
}

func (s *turtleScanner) scanString() (turtleToken, error) {
	quote := s.input[s.pos]
	if s.pos+2 < len(s.input) && s.input[s.pos+1] == quote && s.input[s.pos+2] == quote {
		return s.scanLongString(quote)
	}
	start := s.pos
	s.pos++
	for s.pos < len(s.input) {
		ch := s.input[s.pos]
		if ch == '\\' {
			s.pos += 2
			continue
		}
		if ch == quote {
			s.pos++
			return turtleToken{Kind: TokString, Lexeme: s.input[start:s.pos]}, nil
		}
		s.pos++
	}
	return turtleToken{}, ErrStatementTooLong
}

func (s *turtleScanner) scanLangTag() (turtleToken, error) {
	start := s.pos
	s.pos++ // consume '@'
	for s.pos < len(s.input) {
		ch := s.input[s.pos]
		next := byte(0)
		if s.pos+1 < len(s.input) {
			next = s.input[s.pos+1]
		}
		if isTurtleTerminator(ch, next) {
			break
		}
		s.pos++
	}
	lang := s.input[start+1 : s.pos] // skip '@'
	return turtleToken{Kind: TokLangTag, Lexeme: lang}, nil
}

func (s *turtleScanner) scanLongString(quote byte) (turtleToken, error) {
	start := s.pos
	s.pos += 3
	for s.pos+2 < len(s.input) {
		ch := s.input[s.pos]
		if ch == '\\' {
			s.pos += 2
			continue
		}
		if s.input[s.pos] == quote && s.input[s.pos+1] == quote && s.input[s.pos+2] == quote {
			s.pos += 3
			return turtleToken{Kind: TokStringLong, Lexeme: s.input[start:s.pos]}, nil
		}
		s.pos++
	}
	return turtleToken{}, ErrStatementTooLong
}

func (s *turtleScanner) scanBlankNode() (turtleToken, error) {
	start := s.pos
	s.pos += 2
	for s.pos < len(s.input) {
		ch := s.input[s.pos]
		if isTurtleTerminator(ch, 0) {
			break
		}
		s.pos++
	}
	return turtleToken{Kind: TokBlankNode, Lexeme: s.input[start:s.pos]}, nil
}

func (s *turtleScanner) scanWord() (turtleToken, error) {
	start := s.pos
	for s.pos < len(s.input) {
		ch := s.input[s.pos]
		next := byte(0)
		if s.pos+1 < len(s.input) {
			next = s.input[s.pos+1]
		}
		if isTurtleTerminator(ch, next) {
			break
		}
		s.pos++
	}
	lexeme := s.input[start:s.pos]
	switch {
	case lexeme == lexPrefix || strings.EqualFold(lexeme, lexPrefixBare):
		return turtleToken{Kind: TokPrefix, Lexeme: lexeme}, nil
	case lexeme == lexBase || strings.EqualFold(lexeme, lexBaseBare):
		return turtleToken{Kind: TokBase, Lexeme: lexeme}, nil
	case lexeme == lexVersion || strings.EqualFold(lexeme, lexVersionBare):
		return turtleToken{Kind: TokVersion, Lexeme: lexeme}, nil
	}
	if lexeme == "a" {
		return turtleToken{Kind: TokA, Lexeme: lexeme}, nil
	}
	if lexeme == "true" || lexeme == "false" {
		return turtleToken{Kind: TokBoolean, Lexeme: lexeme}, nil
	}
	if strings.Contains(lexeme, ":") {
		if strings.HasSuffix(lexeme, ":") {
			return turtleToken{Kind: TokPNAMENS, Lexeme: lexeme}, nil
		}
		return turtleToken{Kind: TokPNAMELN, Lexeme: lexeme}, nil
	}
	if isNumericLiteral(lexeme) {
		return turtleToken{Kind: TokDecimal, Lexeme: lexeme}, nil
	}
	return turtleToken{Kind: TokError, Lexeme: lexeme, Err: ErrUnsupportedFormat}, nil
}

func (s *turtleScanner) skipWS() {
	for s.pos < len(s.input) {
		switch s.input[s.pos] {
		case ' ', '\t', '\r', '\n':
			s.pos++
		default:
			return
		}
	}
}

func (s *turtleScanner) match(prefix string) bool {
	return strings.HasPrefix(s.input[s.pos:], prefix)
}

func isNumericLiteral(value string) bool {
	if value == "" {
		return false
	}
	for i := 0; i < len(value); i++ {
		ch := value[i]
		if (ch >= '0' && ch <= '9') || ch == '.' || ch == '+' || ch == '-' || ch == 'e' || ch == 'E' {
			continue
		}
		return false
	}
	return true
}
