package rdf

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrUnsupportedFormat indicates an unsupported format.
	ErrUnsupportedFormat = errors.New("unsupported RDF format")
	// ErrLineTooLong indicates a line exceeded the configured limit.
	ErrLineTooLong = errors.New("rdf: line exceeds configured limit")
	// ErrStatementTooLong indicates a statement exceeded the configured limit.
	ErrStatementTooLong = errors.New("rdf: statement exceeds configured limit")
	// ErrDepthExceeded indicates that nesting depth exceeded the configured limit.
	ErrDepthExceeded = errors.New("rdf: nesting depth exceeded configured limit")
	// ErrTripleLimitExceeded indicates that the maximum number of triples/quads was exceeded.
	ErrTripleLimitExceeded = errors.New("rdf: maximum number of triples/quads exceeded")
)

// ParseError provides structured context for parse failures.
type ParseError struct {
	Format    string // Format name (e.g., "turtle", "ntriples")
	Statement string // Offending statement or input excerpt
	Line      int    // 1-based line number (0 if unknown)
	Column    int    // 1-based column number (0 if unknown)
	Offset    int    // Byte offset in input (0 if unknown)
	Err       error  // Underlying error
}

func (e *ParseError) Error() string {
	// Build error message with position information
	var msg strings.Builder
	msg.WriteString(e.Format)
	
	// Add position information
	if e.Line > 0 {
		if e.Column > 0 {
			fmt.Fprintf(&msg, ":%d:%d", e.Line, e.Column)
		} else {
			fmt.Fprintf(&msg, ":%d", e.Line)
		}
	} else if e.Offset >= 0 {
		fmt.Fprintf(&msg, " (offset %d)", e.Offset)
	}
	
	msg.WriteString(": ")
	msg.WriteString(e.Err.Error())
	
	// Add input excerpt if available
	if e.Statement != "" {
		excerpt := e.formatExcerpt()
		if excerpt != "" {
			msg.WriteString("\n  ")
			msg.WriteString(excerpt)
		}
	}
	
	return msg.String()
}

// formatExcerpt formats a readable excerpt of the statement around the error position.
func (e *ParseError) formatExcerpt() string {
	if e.Statement == "" {
		return ""
	}
	
	const maxExcerptLen = 80
	const contextLen = 40
	
	// If we have column information, show context around the error
	if e.Column > 0 && len(e.Statement) > 0 {
		start := e.Column - 1
		if start < 0 {
			start = 0
		}
		
		// Show context before and after
		excerptStart := start - contextLen
		if excerptStart < 0 {
			excerptStart = 0
		}
		excerptEnd := start + contextLen
		if excerptEnd > len(e.Statement) {
			excerptEnd = len(e.Statement)
		}
		
		excerpt := e.Statement[excerptStart:excerptEnd]
		if excerptStart > 0 {
			excerpt = "..." + excerpt
		}
		if excerptEnd < len(e.Statement) {
			excerpt = excerpt + "..."
		}
		
		// Add caret pointing to error position
		caretPos := start - excerptStart
		if excerptStart > 0 {
			caretPos += 3 // Account for "..."
		}
		if caretPos < 0 {
			caretPos = 0
		}
		if caretPos >= len(excerpt) {
			caretPos = len(excerpt) - 1
		}
		
		// Build excerpt with caret
		var result strings.Builder
		result.WriteString(excerpt)
		result.WriteString("\n  ")
		for i := 0; i < caretPos; i++ {
			result.WriteByte(' ')
		}
		result.WriteByte('^')
		
		return result.String()
	}
	
	// Fall back to truncated statement
	if len(e.Statement) > maxExcerptLen {
		return e.Statement[:maxExcerptLen] + "..."
	}
	return e.Statement
}

func (e *ParseError) Unwrap() error { return e.Err }

// WrapParseError adds format/statement context to a parse error.
func WrapParseError(format, statement string, offset int, err error) error {
	return WrapParseErrorWithPosition(format, statement, 0, 0, offset, err)
}

// WrapParseErrorWithPosition adds format/statement/position context to a parse error.
func WrapParseErrorWithPosition(format, statement string, line, column, offset int, err error) error {
	if err == nil {
		return nil
	}
	var parseErr *ParseError
	if errors.As(err, &parseErr) {
		// Preserve existing position info if better than what we have
		if parseErr.Line > 0 && line == 0 {
			line = parseErr.Line
		}
		if parseErr.Column > 0 && column == 0 {
			column = parseErr.Column
		}
		if parseErr.Offset >= 0 && offset < 0 {
			offset = parseErr.Offset
		}
		return &ParseError{
			Format:    format,
			Statement: statement,
			Line:      line,
			Column:    column,
			Offset:    offset,
			Err:       err,
		}
	}
	return &ParseError{
		Format:    format,
		Statement: statement,
		Line:      line,
		Column:    column,
		Offset:    offset,
		Err:       err,
	}
}
