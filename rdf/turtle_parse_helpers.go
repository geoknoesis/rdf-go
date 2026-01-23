package rdf

import "strings"

func stripComment(line string) string {
	// Only strip comments that are outside of strings and IRIs.
	// Track whether we're inside a string or IRI.
	inString := false
	inIRI := false
	stringChar := byte(0) // Track which quote character started the string.

	for i := 0; i < len(line); i++ {
		ch := line[i]

		if inString {
			// Inside a string - check for closing quote.
			if ch == stringChar {
				// Check if it's escaped.
				if i > 0 && line[i-1] == '\\' {
					// Escaped quote, continue.
					continue
				}
				// End of string.
				inString = false
				stringChar = 0
			}
			continue
		}

		if inIRI {
			// Inside an IRI - check for closing >.
			if ch == '>' {
				// Check if it's escaped (though > shouldn't be escaped in IRIs).
				if i > 0 && line[i-1] == '\\' {
					continue
				}
				// End of IRI.
				inIRI = false
			}
			continue
		}

		// Not in string or IRI.
		if ch == '"' || ch == '\'' {
			// Start of string.
			inString = true
			stringChar = ch
		} else if ch == '<' {
			// Start of IRI.
			inIRI = true
		} else if ch == '#' {
			if i > 0 && line[i-1] == '\\' {
				// Escaped # (PN_LOCAL_ESC) - not a comment.
				continue
			}
			// Comment found outside string/IRI - strip from here.
			return line[:i]
		}
	}

	return line
}

func isStatementComplete(stmt string) bool {
	inString := false
	stringQuote := byte(0)
	longString := false
	inIRI := false
	bracketDepth := 0
	parenDepth := 0
	annotationDepth := 0

	for i := 0; i < len(stmt); i++ {
		ch := stmt[i]

		if inString {
			if ch == '\\' {
				i++
				continue
			}
			if ch == stringQuote {
				if longString {
					if i+2 < len(stmt) && stmt[i+1] == stringQuote && stmt[i+2] == stringQuote {
						inString = false
						longString = false
						i += 2
					}
				} else {
					inString = false
				}
			}
			continue
		}
		if inIRI {
			if ch == '>' && (i == 0 || stmt[i-1] != '\\') {
				inIRI = false
			}
			continue
		}

		if ch == '<' {
			inIRI = true
			continue
		}
		if ch == '"' || ch == '\'' {
			if i+2 < len(stmt) && stmt[i+1] == ch && stmt[i+2] == ch {
				inString = true
				longString = true
				stringQuote = ch
				i += 2
			} else {
				inString = true
				longString = false
				stringQuote = ch
			}
			continue
		}

		if i+1 < len(stmt) && stmt[i] == '{' && stmt[i+1] == '|' {
			annotationDepth++
			i++
			continue
		}
		if i+1 < len(stmt) && stmt[i] == '|' && stmt[i+1] == '}' {
			if annotationDepth > 0 {
				annotationDepth--
			}
			i++
			continue
		}
		switch ch {
		case '[':
			bracketDepth++
		case ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
		case '(':
			parenDepth++
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
		case '.':
			if bracketDepth == 0 && parenDepth == 0 && annotationDepth == 0 {
				if i > 0 && stmt[i-1] >= '0' && stmt[i-1] <= '9' {
					next := byte(0)
					if i+1 < len(stmt) {
						next = stmt[i+1]
					}
					if (next >= 'a' && next <= 'z') || (next >= 'A' && next <= 'Z') || next == '_' {
						break
					}
				}
				rest := strings.TrimSpace(stmt[i+1:])
				if rest == "" {
					return true
				}
			}
		}
	}
	return false
}

func splitTurtleStatements(input string) []string {
	var statements []string
	start := 0
	inString := false
	stringQuote := byte(0)
	longString := false
	inIRI := false
	bracketDepth := 0
	parenDepth := 0
	annotationDepth := 0
	tokenType := ""

	resetState := func() {
		inString = false
		stringQuote = 0
		longString = false
		inIRI = false
		bracketDepth = 0
		parenDepth = 0
		annotationDepth = 0
		tokenType = ""
	}

	for i := 0; i < len(input); i++ {
		ch := input[i]

		if inString {
			if ch == '\\' {
				i++
				continue
			}
			if ch == stringQuote {
				if longString {
					if i+2 < len(input) && input[i+1] == stringQuote && input[i+2] == stringQuote {
						inString = false
						longString = false
						tokenType = ""
						i += 2
					}
				} else {
					inString = false
					tokenType = ""
				}
			}
			continue
		}
		if inIRI {
			if ch == '>' {
				inIRI = false
				tokenType = ""
			}
			continue
		}

		if ch == '<' {
			inIRI = true
			tokenType = "iri"
			continue
		}
		if ch == '"' || ch == '\'' {
			if i+2 < len(input) && input[i+1] == ch && input[i+2] == ch {
				inString = true
				longString = true
				stringQuote = ch
				tokenType = "long_string"
				i += 2
			} else {
				inString = true
				longString = false
				stringQuote = ch
				tokenType = "string"
			}
			continue
		}

		if i+1 < len(input) && input[i] == '{' && input[i+1] == '|' {
			annotationDepth++
			i++
			continue
		}
		if i+1 < len(input) && input[i] == '|' && input[i+1] == '}' {
			if annotationDepth > 0 {
				annotationDepth--
			}
			i++
			continue
		}

		switch ch {
		case '[':
			bracketDepth++
			tokenType = "bracket"
		case ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
			tokenType = ""
		case '(':
			parenDepth++
			tokenType = "paren"
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
			tokenType = ""
		case '.':
			if bracketDepth == 0 && parenDepth == 0 && annotationDepth == 0 {
				if i > 0 && input[i-1] >= '0' && input[i-1] <= '9' {
					next := byte(0)
					if i+1 < len(input) {
						next = input[i+1]
					}
					if (next >= 'a' && next <= 'z') || (next >= 'A' && next <= 'Z') || next == '_' {
						break
					}
				}
				if tokenType == "string" || tokenType == "long_string" || tokenType == "iri" {
					break
				}
				prev := byte(0)
				if i > 0 {
					prev = input[i-1]
				}
				next := byte(0)
				if i+1 < len(input) {
					next = input[i+1]
				}
				if prev == '\\' {
					break
				}
				if prev >= '0' && prev <= '9' && (next == 'e' || next == 'E') {
					break
				}
				if next == 0 || next == ' ' || next == '\t' || next == '\r' || next == '\n' || next == ';' || next == ',' || next == ')' || next == ']' || next == '}' ||
					next == '<' || next == '_' || next == '[' || next == '(' || next == ':' || next == '@' || next == '{' ||
					(next >= 'a' && next <= 'z') || (next >= 'A' && next <= 'Z') || (next >= '0' && next <= '9') {
					statement := strings.TrimSpace(input[start : i+1])
					if statement != "" {
						statements = append(statements, statement)
					}
					start = i + 1
					resetState()
				}
			}
		}
	}
	rest := strings.TrimSpace(input[start:])
	if rest != "" {
		statements = append(statements, rest)
	}
	return statements
}
