package rdf

import (
	"bufio"
	"io"
	"os"
	"strings"
)

// New triple decoder for Turtle
type turtleTripleDecoder struct {
	reader                     *bufio.Reader
	err                        error
	prefixes                   map[string]string
	baseIRI                    string
	pending                    []Triple // Buffer for triples from predicate/object lists
	allowQuotedTripleStatement bool
	opts                       DecodeOptions
}

func newTurtleTripleDecoder(r io.Reader) TripleDecoder {
	return newTurtleTripleDecoderWithOptions(r, DefaultDecodeOptions())
}

func newTurtleTripleDecoderWithOptions(r io.Reader, opts DecodeOptions) TripleDecoder {
	return &turtleTripleDecoder{
		reader:                     bufio.NewReader(r),
		prefixes:                   map[string]string{},
		opts:                       normalizeDecodeOptions(opts),
		allowQuotedTripleStatement: opts.AllowQuotedTripleStatement || os.Getenv("TURTLE_ALLOW_QT_STMT") != "",
	}
}

func (d *turtleTripleDecoder) Next() (Triple, error) {
	// Return pending triples first (from predicate/object lists)
	if len(d.pending) > 0 {
		triple := d.pending[0]
		d.pending = d.pending[1:]
		return triple, nil
	}

	for {
		if err := d.checkContext(); err != nil {
			d.err = err
			return Triple{}, err
		}
		// Accumulate lines until we have a complete statement (ending with .)
		var statement strings.Builder
		for {
			if err := d.checkContext(); err != nil {
				d.err = err
				return Triple{}, err
			}
			line, err := d.readLine()
			if err != nil {
				if err == io.EOF {
					if statement.Len() == 0 {
						return Triple{}, io.EOF
					}
					// Try to parse what we have so far
					break
				}
				d.err = err
				return Triple{}, err
			}

			// Check if this line is a directive (must be on a single line)
			trimmedLine := strings.TrimSpace(stripComment(line))
			if trimmedLine != "" && d.handleDirective(trimmedLine) {
				// Directive handled, continue to next line
				continue
			}

			if err := d.appendStatementPart(&statement, strings.TrimSpace(stripComment(line))); err != nil {
				d.err = err
				return Triple{}, err
			}

			// Check if statement is complete (ends with a top-level .)
			stmt := strings.TrimSpace(statement.String())
			if stmt != "" && isStatementComplete(stmt) {
				break
			}
		}

		line := statement.String()
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		triples, err := d.parseTripleLine(line)
		if err != nil {
			err = d.wrapParseError(line, err)
			d.err = err
			return Triple{}, err
		}
		if len(triples) == 0 {
			continue
		}
		// Return first triple, store rest in buffer
		if len(triples) > 1 {
			d.pending = triples[1:]
		}
		return triples[0], nil
	}
}

func (d *turtleTripleDecoder) Err() error { return d.err }
func (d *turtleTripleDecoder) Close() error {
	return nil
}

func (d *turtleTripleDecoder) readLine() (string, error) {
	return readLineWithLimit(d.reader, d.opts.MaxLineBytes)
}

func (d *turtleTripleDecoder) checkContext() error {
	return checkDecodeContext(d.opts.Context)
}

func (d *turtleTripleDecoder) appendStatementPart(builder *strings.Builder, part string) error {
	if builder.Len() > 0 {
		builder.WriteString(" ")
	}
	builder.WriteString(part)
	if d.opts.MaxStatementBytes > 0 && builder.Len() > d.opts.MaxStatementBytes {
		return ErrStatementTooLong
	}
	return nil
}

func (d *turtleTripleDecoder) wrapParseError(statement string, err error) error {
	if d.opts.DebugStatements || os.Getenv("TURTLE_DEBUG_STATEMENT") != "" {
		return WrapParseError("turtle", statement, -1, err)
	}
	return WrapParseError("turtle", "", -1, err)
}

func (d *turtleTripleDecoder) handleDirective(line string) bool {
	if prefix, iri, ok := parseAtPrefixDirective(line, true); ok {
		d.prefixes[prefix] = iri
		return true
	}
	if prefix, iri, ok := parseBarePrefixDirective(line); ok {
		d.prefixes[prefix] = iri
		return true
	}
	if parseVersionDirective(line) {
		d.allowQuotedTripleStatement = true
		return true
	}
	if iri, ok := parseAtBaseDirective(line); ok {
		d.baseIRI = iri
		return true
	}
	if iri, ok := parseBaseDirective(line); ok {
		d.baseIRI = iri
		return true
	}
	return false
}

func (d *turtleTripleDecoder) parseTripleLine(line string) ([]Triple, error) {
	cursor := &turtleCursor{
		input:                      line,
		prefixes:                   d.prefixes,
		base:                       d.baseIRI,
		expansionTriples:           []Triple{},
		blankNodeCounter:           0,
		allowQuotedTripleStatement: d.allowQuotedTripleStatement,
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
