package rdf

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// New quad decoder for TriG
type trigQuadDecoder struct {
	reader                     *bufio.Reader
	err                        error
	prefixes                   map[string]string
	baseIRI                    string
	graph                      Term
	pending                    []Quad // Buffer for quads from predicate/object lists
	allowQuotedTripleStatement bool
	inGraphBlock               bool
	remainder                  string
	opts                       DecodeOptions
}

func newTriGQuadDecoder(r io.Reader) QuadDecoder {
	return newTriGQuadDecoderWithOptions(r, DefaultDecodeOptions())
}

func newTriGQuadDecoderWithOptions(r io.Reader, opts DecodeOptions) QuadDecoder {
	if opts.AllowEnvOverrides && os.Getenv("TRIG_ALLOW_QT_STMT") != "" {
		opts.AllowQuotedTripleStatement = true
	}
	return &trigQuadDecoder{
		reader:                     bufio.NewReader(r),
		prefixes:                   map[string]string{},
		allowQuotedTripleStatement: opts.AllowQuotedTripleStatement,
		opts:                       normalizeDecodeOptions(opts),
	}
}

func (d *trigQuadDecoder) Next() (Quad, error) {
	// Return pending quads first (from predicate/object lists)
	if len(d.pending) > 0 {
		quad := d.pending[0]
		d.pending = d.pending[1:]
		return quad, nil
	}

	for {
		if err := d.checkContext(); err != nil {
			d.err = err
			return Quad{}, err
		}
		// Accumulate lines until we have a complete statement (ending with .)
		var statement strings.Builder
		graphForStatement := d.graph
		closeGraphAfter := false
		hitEOF := false

		for {
			if err := d.checkContext(); err != nil {
				d.err = err
				return Quad{}, err
			}
			var line string
			var err error
			line, err = d.nextLineOrRemainder()
			if err != nil {
				if err == io.EOF {
					isEmpty := statement.Len() == 0
					if isEmpty {
						return Quad{}, io.EOF
					}
					hitEOF = true
					break
				}
				d.err = err
				return Quad{}, err
			}

			trimmed := strings.TrimSpace(stripComment(line))
			isEmpty := statement.Len() == 0
			if trimmed == "" {
				if isEmpty {
					continue
				}
				continue
			}

			if isEmpty && isTrigDirectiveLine(trimmed) {
				combined, handled, err := d.maybeReadDirectiveContinuation(trimmed)
				if err != nil {
					d.err = err
					return Quad{}, err
				}
				if handled {
					continue
				}
				trimmed = combined
			}

			if d.inGraphBlock && isTrigDirectiveLine(trimmed) {
				d.err = d.wrapParseError("", fmt.Errorf("directives not allowed inside graph"))
				return Quad{}, d.err
			}

			if isEmpty && d.handleDirective(trimmed) {
				continue
			}

			if trimmed == "}" {
				if !isEmpty {
					closeGraphAfter = true
					break
				}
				d.graph = nil
				d.inGraphBlock = false
				graphForStatement = d.graph
				continue
			}

			openIdx, closeIdx := findGraphBlockBounds(trimmed)
			if d.inGraphBlock && openIdx >= 0 && !isAnnotationBlock(trimmed, openIdx) {
				d.err = d.wrapParseError("", fmt.Errorf("nested graph blocks are not allowed"))
				return Quad{}, d.err
			}
			if openIdx >= 0 && closeIdx > openIdx && !isAnnotationBlock(trimmed, openIdx) {
				quads, after, err := d.parseInlineGraphBlock(trimmed, openIdx, closeIdx)
				if err != nil {
					d.err = err
					return Quad{}, err
				}
				if after != "" {
					if !strings.Contains(after, "{") {
						d.err = d.wrapParseError("", fmt.Errorf("unexpected content after graph block"))
						return Quad{}, d.err
					}
					d.remainder = after
				}
				if len(quads) == 0 {
					continue
				}
				if len(quads) > 1 {
					d.pending = quads[1:]
				}
				return quads[0], nil
			}

			if openIdx >= 0 && closeIdx < 0 && !isAnnotationBlock(trimmed, openIdx) {
				graphToken := strings.TrimSpace(trimmed[:openIdx])
				graphTerm, err := d.parseGraphToken(graphToken)
				if err != nil {
					d.err = d.wrapParseError("", err)
					return Quad{}, d.err
				}
				d.graph = graphTerm
				d.inGraphBlock = true
				graphForStatement = d.graph
				after := strings.TrimSpace(trimmed[openIdx+1:])
				if after != "" {
					if !isEmpty {
						statement.WriteString(" ")
					}
					statement.WriteString(after)
					if d.opts.MaxStatementBytes > 0 && statement.Len() > d.opts.MaxStatementBytes {
						d.err = ErrStatementTooLong
						return Quad{}, d.err
					}
				}
				continue
			}

			handled, err := d.handleStartGraphBlock(trimmed, &graphForStatement)
			if err != nil {
				d.err = err
				return Quad{}, err
			}
			if handled {
				continue
			}

			shouldClose, err := d.handleInlineGraphClose(trimmed, &statement, &closeGraphAfter)
			if err != nil {
				d.err = err
				return Quad{}, err
			}
			if shouldClose {
				break
			}

			// Update isEmpty check since we're about to add content
			isEmpty = statement.Len() == 0
			if !isEmpty {
				statement.WriteString(" ")
			}
			statement.WriteString(trimmed)
			if d.opts.MaxStatementBytes > 0 && statement.Len() > d.opts.MaxStatementBytes {
				d.err = ErrStatementTooLong
				return Quad{}, d.err
			}
			// Check if statement is complete (only trim once when needed)
			if isStatementComplete(strings.TrimSpace(statement.String())) {
				break
			}
		}

		if hitEOF && d.inGraphBlock {
			d.err = d.wrapParseError("", fmt.Errorf("expected '}'"))
			return Quad{}, d.err
		}

		// Build the final statement string once
		line := strings.TrimSpace(statement.String())
		if closeGraphAfter && line != "" && !strings.HasSuffix(line, ".") {
			var lineBuilder strings.Builder
			lineBuilder.WriteString(line)
			lineBuilder.WriteString(" .")
			line = lineBuilder.String()
		}
		if line == "" {
			if closeGraphAfter {
				d.graph = nil
				d.inGraphBlock = false
			}
			continue
		}

		quads, err := d.processStatement(line, graphForStatement)
		if err != nil {
			d.err = err
			return Quad{}, err
		}
		if closeGraphAfter {
			d.graph = nil
			d.inGraphBlock = false
		}
		if len(quads) == 0 {
			continue
		}
		if len(quads) > 1 {
			d.pending = quads[1:]
		}
		return quads[0], nil
	}
}

// buildStatement accumulates lines until we have a complete statement.
// Returns: statement string, graphForStatement, closeGraphAfter, hitEOF, optional quad (if inline graph block), error
func (d *trigQuadDecoder) buildStatement() (string, Term, bool, bool, *Quad, error) {
	var statement strings.Builder
	graphForStatement := d.graph
	closeGraphAfter := false
	hitEOF := false

	for {
		if err := d.checkContext(); err != nil {
			return "", nil, false, false, nil, err
		}
		var line string
		var err error
		line, err = d.nextLineOrRemainder()
		if err != nil {
			if err == io.EOF {
				isEmpty := statement.Len() == 0
				if isEmpty {
					return "", nil, false, false, nil, io.EOF
				}
				hitEOF = true
				break
			}
			return "", nil, false, false, nil, err
		}

		trimmed := strings.TrimSpace(stripComment(line))
		isEmpty := statement.Len() == 0
		if trimmed == "" {
			if isEmpty {
				continue
			}
			continue
		}

		if isEmpty && isTrigDirectiveLine(trimmed) {
			combined, handled, err := d.maybeReadDirectiveContinuation(trimmed)
			if err != nil {
				return "", nil, false, false, nil, err
			}
			if handled {
				continue
			}
			trimmed = combined
		}

		if d.inGraphBlock && isTrigDirectiveLine(trimmed) {
			return "", nil, false, false, nil, d.wrapParseError("", fmt.Errorf("directives not allowed inside graph"))
		}

		if isEmpty && d.handleDirective(trimmed) {
			continue
		}

		if trimmed == "}" {
			if !isEmpty {
				closeGraphAfter = true
				break
			}
			d.graph = nil
			d.inGraphBlock = false
			graphForStatement = d.graph
			continue
		}

		openIdx, closeIdx := findGraphBlockBounds(trimmed)
		if d.inGraphBlock && openIdx >= 0 && !isAnnotationBlock(trimmed, openIdx) {
			return "", nil, false, false, nil, d.wrapParseError("", fmt.Errorf("nested graph blocks are not allowed"))
		}
		if openIdx >= 0 && closeIdx > openIdx && !isAnnotationBlock(trimmed, openIdx) {
			quads, after, err := d.parseInlineGraphBlock(trimmed, openIdx, closeIdx)
			if err != nil {
				return "", nil, false, false, nil, err
			}
			if after != "" {
				if !strings.Contains(after, "{") {
					return "", nil, false, false, nil, d.wrapParseError("", fmt.Errorf("unexpected content after graph block"))
				}
				d.remainder = after
			}
			if len(quads) == 0 {
				continue
			}
			if len(quads) > 1 {
				d.pending = quads[1:]
			}
			quad := quads[0]
			return "", nil, false, false, &quad, nil
		}

		if openIdx >= 0 && closeIdx < 0 && !isAnnotationBlock(trimmed, openIdx) {
			graphToken := strings.TrimSpace(trimmed[:openIdx])
			graphTerm, err := d.parseGraphToken(graphToken)
			if err != nil {
				return "", nil, false, false, nil, d.wrapParseError("", err)
			}
			d.graph = graphTerm
			d.inGraphBlock = true
			graphForStatement = d.graph
			after := strings.TrimSpace(trimmed[openIdx+1:])
			if after != "" {
				if !isEmpty {
					statement.WriteString(" ")
				}
				statement.WriteString(after)
				if d.opts.MaxStatementBytes > 0 && statement.Len() > d.opts.MaxStatementBytes {
					return "", nil, false, false, nil, ErrStatementTooLong
				}
			}
			continue
		}

		handled, err := d.handleStartGraphBlock(trimmed, &graphForStatement)
		if err != nil {
			return "", nil, false, false, nil, err
		}
		if handled {
			continue
		}

		shouldClose, err := d.handleInlineGraphClose(trimmed, &statement, &closeGraphAfter)
		if err != nil {
			return "", nil, false, false, nil, err
		}
		if shouldClose {
			break
		}

		// Update isEmpty check since we're about to add content
		isEmpty = statement.Len() == 0
		if !isEmpty {
			statement.WriteString(" ")
		}
		statement.WriteString(trimmed)
		if d.opts.MaxStatementBytes > 0 && statement.Len() > d.opts.MaxStatementBytes {
			return "", nil, false, false, nil, ErrStatementTooLong
		}
		// Check if statement is complete (only trim once when needed)
		if isStatementComplete(strings.TrimSpace(statement.String())) {
			break
		}
	}

	return statement.String(), graphForStatement, closeGraphAfter, hitEOF, nil, nil
}

func (d *trigQuadDecoder) Err() error { return d.err }
func (d *trigQuadDecoder) Close() error {
	return nil
}

func (d *trigQuadDecoder) readLine() (string, error) {
	return readLineWithLimit(d.reader, d.opts.MaxLineBytes)
}

func (d *trigQuadDecoder) checkContext() error {
	return checkDecodeContext(d.opts.Context)
}

func (d *trigQuadDecoder) parseGraphToken(token string) (Term, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, nil
	}
	upper := strings.ToUpper(token)
	if strings.HasPrefix(upper, directiveGraph+" ") {
		token = strings.TrimSpace(token[len(directiveGraph+" "):])
		if token == "" {
			return nil, fmt.Errorf("expected graph name")
		}
	}
	if strings.HasPrefix(token, "[") && token != "[]" {
		return nil, fmt.Errorf("invalid graph name")
	}
	if strings.HasPrefix(token, "(") {
		return nil, fmt.Errorf("invalid graph name")
	}
	cursor := &turtleCursor{input: token, prefixes: d.prefixes, base: d.baseIRI}
	term, err := cursor.parseTerm(false)
	if err != nil {
		return nil, err
	}
	cursor.skipWS()
	if cursor.pos != len(cursor.input) {
		return nil, fmt.Errorf("invalid graph name")
	}
	return term, nil
}

func isTrigDirectiveLine(line string) bool {
	lower := strings.ToLower(line)
	if strings.HasPrefix(lower, "@prefix") || strings.HasPrefix(lower, "@base") || strings.HasPrefix(lower, "@version") {
		return true
	}
	if strings.HasPrefix(lower, "prefix") || strings.HasPrefix(lower, "base") || strings.HasPrefix(lower, "version") {
		return true
	}
	return false
}

func (d *trigQuadDecoder) handleDirective(line string) bool {
	if prefix, iri, ok := parseAtPrefixDirective(line, false); ok {
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

func (d *trigQuadDecoder) parseTripleLine(line string) ([]Quad, error) {
	debugStatements := d.opts.DebugStatements || (d.opts.AllowEnvOverrides && os.Getenv("TURTLE_DEBUG_STATEMENT") != "")
	opts := TurtleParseOptions{
		Prefixes:        d.prefixes,
		BaseIRI:         d.baseIRI,
		AllowQuoted:     d.allowQuotedTripleStatement,
		DebugStatements: debugStatements,
	}
	triples, err := parseTurtleTripleLineWithOptions(opts, line)
	if err != nil {
		return nil, err
	}
	quads := make([]Quad, 0, len(triples))
	for _, triple := range triples {
		quads = append(quads, Quad{S: triple.S, P: triple.P, O: triple.O, G: d.graph})
	}
	return quads, nil
}

// processStatement parses a statement line and returns quads with the specified graph.
func (d *trigQuadDecoder) processStatement(line string, graphForStatement Term) ([]Quad, error) {
	statements := splitTurtleStatements(line)
	var quads []Quad
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		// Use strings.Builder for string concatenation instead of +
		if !strings.HasSuffix(stmt, ".") {
			var stmtBuilder strings.Builder
			stmtBuilder.WriteString(stmt)
			stmtBuilder.WriteString(" .")
			stmt = stmtBuilder.String()
		}
		stmt = normalizeTriGStatement(stmt)
		parsed, err := d.parseTripleLine(stmt)
		if err != nil {
			return nil, d.wrapParseError(stmt, err)
		}
		for i := range parsed {
			parsed[i].G = graphForStatement
		}
		quads = append(quads, parsed...)
	}
	return quads, nil
}

func (d *trigQuadDecoder) parseInlineGraphBlock(trimmed string, openIdx, closeIdx int) ([]Quad, string, error) {
	graphToken := strings.TrimSpace(trimmed[:openIdx])
	inner := strings.TrimSpace(trimmed[openIdx+1 : closeIdx])
	after := strings.TrimSpace(trimmed[closeIdx+1:])
	graphTerm, err := d.parseGraphToken(graphToken)
	if err != nil {
		return nil, "", d.wrapParseError("", err)
	}
	if inner == "" {
		return nil, after, nil
	}
	statements := splitTurtleStatements(inner)
	debugStatements := d.opts.DebugStatements || (d.opts.AllowEnvOverrides && os.Getenv("TURTLE_DEBUG_STATEMENT") != "")
	var quads []Quad
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if !strings.HasSuffix(stmt, ".") {
			var stmtBuilder strings.Builder
			stmtBuilder.WriteString(stmt)
			stmtBuilder.WriteString(" .")
			stmt = stmtBuilder.String()
		}
		stmt = normalizeTriGStatement(stmt)
		opts := TurtleParseOptions{
			Prefixes:        d.prefixes,
			BaseIRI:         d.baseIRI,
			AllowQuoted:     d.allowQuotedTripleStatement,
			DebugStatements: debugStatements,
		}
		triples, err := parseTurtleTripleLineWithOptions(opts, stmt)
		if err != nil {
			return nil, "", d.wrapParseError(stmt, err)
		}
		for _, triple := range triples {
			quads = append(quads, Quad{S: triple.S, P: triple.P, O: triple.O, G: graphTerm})
		}
	}
	return quads, after, nil
}

func (d *trigQuadDecoder) wrapParseError(statement string, err error) error {
	if d.opts.DebugStatements || (d.opts.AllowEnvOverrides && os.Getenv("TURTLE_DEBUG_STATEMENT") != "") {
		return WrapParseError("trig", statement, -1, err)
	}
	return WrapParseError("trig", "", -1, err)
}

func (d *trigQuadDecoder) handleStartGraphBlock(trimmed string, graphForStatement *Term) (bool, error) {
	if !strings.HasSuffix(trimmed, "{") {
		return false, nil
	}
	graphToken := strings.TrimSpace(strings.TrimSuffix(trimmed, "{"))
	graphTerm, err := d.parseGraphToken(graphToken)
	if err != nil {
		return false, d.wrapParseError("", err)
	}
	d.graph = graphTerm
	d.inGraphBlock = true
	*graphForStatement = d.graph
	return true, nil
}

func (d *trigQuadDecoder) handleInlineGraphClose(trimmed string, statement *strings.Builder, closeGraphAfter *bool) (bool, error) {
	if !d.inGraphBlock || !strings.Contains(trimmed, "}") || strings.Contains(trimmed, "|}") {
		return false, nil
	}
	closeIdx := strings.Index(trimmed, "}")
	before := strings.TrimSpace(trimmed[:closeIdx])
	after := strings.TrimSpace(trimmed[closeIdx+1:])
	if before != "" {
		if statement.Len() > 0 {
			statement.WriteString(" ")
		}
		statement.WriteString(before)
		if d.opts.MaxStatementBytes > 0 && statement.Len() > d.opts.MaxStatementBytes {
			return false, ErrStatementTooLong
		}
	}
	*closeGraphAfter = true
	if after != "" {
		d.remainder = after
	}
	return true, nil
}

func findGraphBlockBounds(trimmed string) (int, int) {
	openIdx := strings.Index(trimmed, "{")
	closeIdx := -1
	if openIdx >= 0 {
		for i := openIdx + 1; i < len(trimmed); i++ {
			if trimmed[i] == '}' {
				if i > 0 && trimmed[i-1] == '|' {
					continue
				}
				closeIdx = i
				break
			}
		}
	}
	return openIdx, closeIdx
}

func isAnnotationBlock(trimmed string, openIdx int) bool {
	return openIdx+1 < len(trimmed) && trimmed[openIdx+1] == '|'
}

func (d *trigQuadDecoder) nextLineOrRemainder() (string, error) {
	if d.remainder != "" {
		line := d.remainder
		d.remainder = ""
		return line, nil
	}
	return d.readLine()
}

func (d *trigQuadDecoder) maybeReadDirectiveContinuation(trimmed string) (string, bool, error) {
	if !isTrigDirectiveLine(trimmed) || strings.Contains(trimmed, "<") || strings.Contains(trimmed, ":") {
		return trimmed, false, nil
	}
	nextLine, err := d.nextLineOrRemainder()
	if err != nil {
		if err == io.EOF {
			return "", false, fmt.Errorf("incomplete directive")
		}
		return "", false, err
	}
	combined := strings.TrimSpace(trimmed + " " + strings.TrimSpace(stripComment(nextLine)))
	if d.handleDirective(combined) {
		return combined, true, nil
	}
	return combined, false, nil
}
