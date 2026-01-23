package rdf

import (
	"errors"
	"fmt"
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
	// Prefer line/column information when available
	if e.Line > 0 {
		if e.Column > 0 {
			return fmt.Sprintf("%s:%d:%d: %v", e.Format, e.Line, e.Column, e.Err)
		}
		return fmt.Sprintf("%s:%d: %v", e.Format, e.Line, e.Err)
	}
	// Fall back to offset or statement context
	switch {
	case e.Statement != "" && e.Offset >= 0:
		return fmt.Sprintf("%s: parse error at offset %d in %q: %v", e.Format, e.Offset, e.Statement, e.Err)
	case e.Statement != "":
		return fmt.Sprintf("%s: parse error in %q: %v", e.Format, e.Statement, e.Err)
	case e.Offset >= 0:
		return fmt.Sprintf("%s: parse error at offset %d: %v", e.Format, e.Offset, e.Err)
	default:
		return fmt.Sprintf("%s: parse error: %v", e.Format, e.Err)
	}
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
