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
)

// ParseError provides structured context for parse failures.
type ParseError struct {
	Format    string
	Statement string
	Offset    int
	Err       error
}

func (e *ParseError) Error() string {
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
	if err == nil {
		return nil
	}
	var parseErr *ParseError
	if errors.As(err, &parseErr) {
		return err
	}
	return &ParseError{
		Format:    format,
		Statement: statement,
		Offset:    offset,
		Err:       err,
	}
}
