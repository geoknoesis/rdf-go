package rdf

import (
	"bufio"
	"context"
	"io"
	"strings"
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

func decodeUChar(hexStr string) rune {
	if len(hexStr) != 4 && len(hexStr) != 8 {
		return -1
	}
	var codePoint rune
	for i := 0; i < len(hexStr); i++ {
		ch := hexStr[i]
		var digit rune
		switch {
		case ch >= '0' && ch <= '9':
			digit = rune(ch - '0')
		case ch >= 'a' && ch <= 'f':
			digit = rune(ch-'a') + 10
		case ch >= 'A' && ch <= 'F':
			digit = rune(ch-'A') + 10
		default:
			return -1
		}
		codePoint = codePoint*16 + digit
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
	if !strings.HasPrefix(line, "@prefix") {
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
	if strings.HasPrefix(line, "@") || !strings.HasPrefix(strings.ToUpper(line), "PREFIX") {
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
	if !strings.HasPrefix(strings.ToLower(line), "version") && !strings.HasPrefix(line, "@version") {
		return false
	}
	rest := strings.TrimSpace(line)
	if strings.HasPrefix(rest, "@version") {
		rest = strings.TrimSpace(rest[len("@version"):])
	} else {
		rest = strings.TrimSpace(rest[len("version"):])
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
	if !strings.HasPrefix(line, "@base") {
		return "", false
	}
	rest := strings.TrimSpace(line[len("@base"):])
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
	if strings.HasPrefix(line, "@") || !strings.HasPrefix(strings.ToUpper(line), "BASE") {
		return "", false
	}
	if strings.HasSuffix(strings.TrimSpace(line), ".") {
		return "", false
	}
	rest := strings.TrimSpace(line[len("BASE"):])
	if !strings.HasPrefix(rest, "<") {
		return "", false
	}
	closeIdx := strings.Index(rest, ">")
	if closeIdx <= 0 {
		return "", false
	}
	return rest[1:closeIdx], true
}
