package rdf

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
)

// Unicode surrogate pair constants
const (
	unicodeSurrogateHighStart = 0xD800
	unicodeSurrogateHighEnd   = 0xDBFF
	unicodeSurrogateLowStart  = 0xDC00
	unicodeSurrogateLowEnd    = 0xDFFF
	unicodeSurrogateBase      = 0x10000
)

// Directive keywords for Turtle/TriG
const (
	directiveAtPrefix  = "@prefix"
	directivePrefix    = "PREFIX"
	directiveAtBase    = "@base"
	directiveBase      = "BASE"
	directiveAtVersion = "@version"
	directiveVersion   = "VERSION"
	directiveGraph     = "GRAPH"
)

// String literal length constants
const (
	minLongStringLength     = 6  // Minimum length for """...""" or '''...'''
	minStringLength         = 2  // Minimum length for "..." or '...'
	unicodeEscapeLength     = 6  // Length of \uXXXX escape sequence
	unicodeLongEscapeLength = 10 // Length of \UXXXXXXXX escape sequence
)

func isHexDigit(ch byte) bool {
	return (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}

func isValidPNLocalEscape(ch byte) bool {
	switch ch {
	case '_', '~', '.', '-', '!', '$', '&', '\'', '(', ')', '*', '+', ',', ';', '=', '/', '?', '#', '@', '%':
		return true
	default:
		return false
	}
}

func isValidLangTag(tag string) bool {
	if tag == "" {
		return false
	}

	hasDir := strings.Contains(tag, "--")
	if hasDir {
		if strings.Count(tag, "--") > 1 {
			return false
		}
		if strings.HasSuffix(tag, "--ltr") {
			tag = strings.TrimSuffix(tag, "--ltr")
		} else if strings.HasSuffix(tag, "--rtl") {
			tag = strings.TrimSuffix(tag, "--rtl")
		} else {
			return false
		}
	}

	parts := strings.Split(tag, "-")
	if len(parts) == 0 {
		return false
	}
	if len(parts[0]) < 1 || len(parts[0]) > 8 {
		return false
	}
	for i, part := range parts {
		if part == "" {
			return false
		}
		for j := 0; j < len(part); j++ {
			ch := part[j]
			if i == 0 {
				if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')) {
					return false
				}
			} else {
				if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')) {
					return false
				}
			}
		}
	}
	return true
}

func isValidUnicodeCodePoint(codePoint rune) bool {
	if codePoint > 0x10FFFF {
		return false
	}
	if codePoint >= 0xD800 && codePoint <= 0xDFFF {
		return false
	}
	return true
}

// parseHexDigit converts a single hex digit byte to its integer value.
// Returns the digit value and true if valid, or 0 and false if invalid.
func parseHexDigit(hex byte) (int, bool) {
	switch {
	case hex >= '0' && hex <= '9':
		return int(hex - '0'), true
	case hex >= 'a' && hex <= 'f':
		return int(hex-'a') + 10, true
	case hex >= 'A' && hex <= 'F':
		return int(hex-'A') + 10, true
	default:
		return 0, false
	}
}

func decodeUChar(hexStr string) rune {
	if len(hexStr) != 4 && len(hexStr) != 8 {
		return -1
	}
	var codePoint rune
	for i := 0; i < len(hexStr); i++ {
		digit, ok := parseHexDigit(hexStr[i])
		if !ok {
			return -1
		}
		codePoint = codePoint*16 + rune(digit)
	}
	return codePoint
}

func isValidPrefixName(prefix string) bool {
	if prefix == "" {
		return true
	}
	if prefix[0] == '.' || prefix[len(prefix)-1] == '.' {
		return false
	}
	first := prefix[0]
	if !((first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z') || first == '_' || first >= 0x80) {
		return false
	}
	for i := 1; i < len(prefix); i++ {
		ch := prefix[i]
		if ch == '.' {
			continue
		}
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '-' || ch >= 0x80 {
			continue
		}
		return false
	}
	return true
}

func readLineWithLimit(reader *bufio.Reader, maxBytes int) (string, error) {
	if maxBytes < 0 {
		maxBytes = 0
	}
	if maxBytes == 0 {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF && len(line) > 0 {
				return line, nil
			}
			return "", err
		}
		return line, nil
	}

	var buffer []byte
	for {
		part, err := reader.ReadSlice('\n')
		buffer = append(buffer, part...)
		if len(buffer) > maxBytes {
			discardLine(reader)
			return "", ErrLineTooLong
		}
		if err == nil {
			return string(buffer), nil
		}
		if err == bufio.ErrBufferFull {
			continue
		}
		if err == io.EOF && len(buffer) > 0 {
			return string(buffer), nil
		}
		return "", err
	}
}

func discardLine(reader *bufio.Reader) {
	for {
		_, err := reader.ReadSlice('\n')
		if err == nil {
			return
		}
		if err != bufio.ErrBufferFull {
			return
		}
	}
}

type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (c *contextReader) Read(p []byte) (int, error) {
	select {
	case <-c.ctx.Done():
		return 0, c.ctx.Err()
	default:
		return c.r.Read(p)
	}
}

func checkDecodeContext(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func parseAtPrefixDirective(line string, requireTerminator bool) (string, string, bool) {
	if !strings.HasPrefix(line, directiveAtPrefix) {
		return "", "", false
	}
	parts := strings.Fields(line)
	if len(parts) < 3 {
		return "", "", false
	}
	if !strings.HasSuffix(parts[1], ":") {
		return "", "", false
	}
	prefix := strings.TrimSuffix(parts[1], ":")
	if !isValidPrefixName(prefix) {
		return "", "", false
	}
	iriPart := parts[2]
	if !strings.HasPrefix(iriPart, "<") {
		return "", "", false
	}
	closeIdx := strings.Index(iriPart, ">")
	if closeIdx <= 0 {
		return "", "", false
	}
	if requireTerminator {
		if closeIdx+1 < len(iriPart) && iriPart[closeIdx+1] == '.' {
			// Period is part of the statement terminator, already handled.
		} else if len(parts) > 3 && parts[3] == "." {
			// Period is separate token.
		} else if !strings.HasSuffix(line, ".") {
			return "", "", false
		}
	}
	return prefix, iriPart[1:closeIdx], true
}

func parseBarePrefixDirective(line string) (string, string, bool) {
	if strings.HasPrefix(line, "@") || !strings.HasPrefix(strings.ToUpper(line), directivePrefix) {
		return "", "", false
	}
	parts := strings.Fields(line)
	if len(parts) < 3 {
		return "", "", false
	}
	if !strings.HasSuffix(parts[1], ":") {
		return "", "", false
	}
	prefix := strings.TrimSuffix(parts[1], ":")
	if !isValidPrefixName(prefix) {
		return "", "", false
	}
	iriPart := parts[2]
	if !strings.HasPrefix(iriPart, "<") {
		return "", "", false
	}
	closeIdx := strings.Index(iriPart, ">")
	if closeIdx <= 0 {
		return "", "", false
	}
	return prefix, iriPart[1:closeIdx], true
}

func parseVersionDirective(line string) bool {
	if !strings.HasPrefix(strings.ToLower(line), strings.ToLower(directiveVersion)) && !strings.HasPrefix(line, directiveAtVersion) {
		return false
	}
	rest := strings.TrimSpace(line)
	if strings.HasPrefix(rest, directiveAtVersion) {
		rest = strings.TrimSpace(rest[len(directiveAtVersion):])
	} else {
		rest = strings.TrimSpace(rest[len(directiveVersion):])
	}
	rest = strings.TrimSpace(strings.TrimSuffix(rest, "."))
	if rest == "" {
		return false
	}
	quote := rest[0]
	if quote != '"' && quote != '\'' {
		return false
	}
	if len(rest) >= 3 && rest[1] == quote && rest[2] == quote {
		return false
	}
	end := strings.IndexByte(rest[1:], quote)
	return end >= 0
}

func parseAtBaseDirective(line string) (string, bool) {
	if !strings.HasPrefix(line, directiveAtBase) {
		return "", false
	}
	rest := strings.TrimSpace(line[len(directiveAtBase):])
	if !strings.HasPrefix(rest, "<") {
		return "", false
	}
	closeIdx := strings.Index(rest, ">")
	if closeIdx <= 0 {
		return "", false
	}
	return rest[1:closeIdx], true
}

func parseBaseDirective(line string) (string, bool) {
	if strings.HasPrefix(line, "@") || !strings.HasPrefix(strings.ToUpper(line), directiveBase) {
		return "", false
	}
	if strings.HasSuffix(strings.TrimSpace(line), ".") {
		return "", false
	}
	rest := strings.TrimSpace(line[len(directiveBase):])
	if !strings.HasPrefix(rest, "<") {
		return "", false
	}
	closeIdx := strings.Index(rest, ">")
	if closeIdx <= 0 {
		return "", false
	}
	return rest[1:closeIdx], true
}

// UnescapeString decodes escape sequences in RDF string literals.
// It handles simple escapes (\n, \t, etc.), Unicode escapes (\uXXXX), and Unicode long escapes (\UXXXXXXXX).
// Surrogate pairs are supported for \uXXXX sequences.
func UnescapeString(s string) (string, error) {
	var builder strings.Builder
	pos := 0
	for pos < len(s) {
		ch := s[pos]
		if ch == '\\' {
			if pos+1 >= len(s) {
				return "", fmt.Errorf("unterminated escape")
			}
			next := s[pos+1]
			var err error
			var advance int
			switch next {
			case 'n', 't', 'r', 'b', 'f', '"', '\'', '\\':
				advance, err = unescapeSimpleEscape(&builder, next)
			case 'u':
				advance, err = unescapeUnicodeEscape(&builder, s, pos)
			case 'U':
				advance, err = unescapeUnicodeLongEscape(&builder, s, pos)
			default:
				return "", fmt.Errorf("invalid escape sequence")
			}
			if err != nil {
				return "", err
			}
			pos += advance
			continue
		}
		builder.WriteByte(ch)
		pos++
	}
	return builder.String(), nil
}

// unescapeSimpleEscape handles simple escape sequences like \n, \t, etc.
func unescapeSimpleEscape(builder *strings.Builder, escapeChar byte) (int, error) {
	switch escapeChar {
	case 'n':
		builder.WriteByte('\n')
	case 't':
		builder.WriteByte('\t')
	case 'r':
		builder.WriteByte('\r')
	case 'b':
		builder.WriteByte('\b')
	case 'f':
		builder.WriteByte('\f')
	case '"':
		builder.WriteByte('"')
	case '\'':
		builder.WriteByte('\'')
	case '\\':
		builder.WriteByte('\\')
	}
	return 2, nil
}

// unescapeUnicodeEscape handles \uXXXX escape sequences, including surrogate pairs.
func unescapeUnicodeEscape(builder *strings.Builder, s string, pos int) (int, error) {
	if pos+unicodeEscapeLength >= len(s) {
		return 0, fmt.Errorf("invalid escape sequence")
	}
	codePoint := decodeUChar(s[pos+2 : pos+6])
	if codePoint < 0 {
		return 0, fmt.Errorf("invalid escape sequence")
	}

	if codePoint >= unicodeSurrogateHighStart && codePoint <= unicodeSurrogateHighEnd {
		// Surrogate pair - need second \uXXXX
		return unescapeSurrogatePair(builder, s, pos, codePoint)
	}

	if codePoint >= unicodeSurrogateLowStart && codePoint <= unicodeSurrogateLowEnd {
		return 0, fmt.Errorf("invalid escape sequence")
	}

	if !isValidUnicodeCodePoint(codePoint) {
		return 0, fmt.Errorf("invalid escape sequence")
	}

	builder.WriteRune(codePoint)
	return 6, nil
}

// unescapeSurrogatePair handles surrogate pair escape sequences \uXXXX\uYYYY.
func unescapeSurrogatePair(builder *strings.Builder, s string, pos int, high rune) (int, error) {
	if pos+11 >= len(s) || s[pos+6] != '\\' || s[pos+7] != 'u' {
		return 0, fmt.Errorf("invalid escape sequence")
	}
	low := decodeUChar(s[pos+8 : pos+12])
	if low < 0 || low < unicodeSurrogateLowStart || low > unicodeSurrogateLowEnd {
		return 0, fmt.Errorf("invalid escape sequence")
	}
	combined := unicodeSurrogateBase + ((high - unicodeSurrogateHighStart) << 10) + (low - unicodeSurrogateLowStart)
	if !isValidUnicodeCodePoint(rune(combined)) {
		return 0, fmt.Errorf("invalid escape sequence")
	}
	builder.WriteRune(rune(combined))
	return 12, nil
}

// unescapeUnicodeLongEscape handles \UXXXXXXXX escape sequences.
func unescapeUnicodeLongEscape(builder *strings.Builder, s string, pos int) (int, error) {
	if pos+unicodeLongEscapeLength >= len(s) {
		return 0, fmt.Errorf("invalid escape sequence")
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
			return 0, fmt.Errorf("invalid escape sequence")
		}
		codePoint = codePoint*16 + rune(digit)
	}
	if !isValidUnicodeCodePoint(codePoint) {
		return 0, fmt.Errorf("invalid escape sequence")
	}
	builder.WriteRune(codePoint)
	return 10, nil
}
