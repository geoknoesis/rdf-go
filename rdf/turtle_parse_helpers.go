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

// turtleStatementState tracks the parsing state when scanning Turtle statements.
type turtleStatementState struct {
	inString        bool
	stringQuote     byte
	longString      bool
	inIRI           bool
	bracketDepth    int
	parenDepth      int
	annotationDepth int
}

// reset resets the state to initial values.
func (s *turtleStatementState) reset() {
	s.inString = false
	s.stringQuote = 0
	s.longString = false
	s.inIRI = false
	s.bracketDepth = 0
	s.parenDepth = 0
	s.annotationDepth = 0
}

// updateState processes a character and updates the parsing state.
// Returns true if the character was consumed as part of a multi-character construct.
func (s *turtleStatementState) updateState(ch byte, input string, pos int) (consumed int) {
	if s.inString {
		if ch == '\\' {
			return 1 // Skip escape character, next iteration will handle escaped char
		}
		if ch == s.stringQuote {
			if s.longString {
				if pos+2 < len(input) && input[pos+1] == s.stringQuote && input[pos+2] == s.stringQuote {
					s.inString = false
					s.longString = false
					return 2 // Consumed 2 more chars (total 3)
				}
			} else {
				s.inString = false
			}
		}
		return 0
	}

	if s.inIRI {
		if ch == '>' && (pos == 0 || input[pos-1] != '\\') {
			s.inIRI = false
		}
		return 0
	}

	if ch == '<' {
		s.inIRI = true
		return 0
	}

	if ch == '"' || ch == '\'' {
		if pos+2 < len(input) && input[pos+1] == ch && input[pos+2] == ch {
			s.inString = true
			s.longString = true
			s.stringQuote = ch
			return 2 // Consumed 2 more chars (total 3)
		} else {
			s.inString = true
			s.longString = false
			s.stringQuote = ch
		}
		return 0
	}

	if pos+1 < len(input) && ch == '{' && input[pos+1] == '|' {
		s.annotationDepth++
		return 1 // Consumed 1 more char
	}
	if pos+1 < len(input) && ch == '|' && input[pos+1] == '}' {
		if s.annotationDepth > 0 {
			s.annotationDepth--
		}
		return 1 // Consumed 1 more char
	}

	switch ch {
	case '[':
		s.bracketDepth++
	case ']':
		if s.bracketDepth > 0 {
			s.bracketDepth--
		}
	case '(':
		s.parenDepth++
	case ')':
		if s.parenDepth > 0 {
			s.parenDepth--
		}
	}
	return 0
}

// isBalanced returns true if all brackets, parens, and annotations are balanced.
func (s *turtleStatementState) isBalanced() bool {
	return s.bracketDepth == 0 && s.parenDepth == 0 && s.annotationDepth == 0
}

func isStatementComplete(stmt string) bool {
	var state turtleStatementState

	for i := 0; i < len(stmt); i++ {
		ch := stmt[i]
		consumed := state.updateState(ch, stmt, i)
		i += consumed

		if ch == '.' && state.isBalanced() {
			if i > 0 && stmt[i-1] >= '0' && stmt[i-1] <= '9' {
				next := byte(0)
				if i+1 < len(stmt) {
					next = stmt[i+1]
				}
				if (next >= 'a' && next <= 'z') || (next >= 'A' && next <= 'Z') || next == '_' {
					continue
				}
			}
			rest := strings.TrimSpace(stmt[i+1:])
			if rest == "" {
				return true
			}
		}
	}
	return false
}

func splitTurtleStatements(input string) []string {
	var statements []string
	start := 0
	var state turtleStatementState
	tokenType := ""

	for i := 0; i < len(input); i++ {
		ch := input[i]
		consumed := state.updateState(ch, input, i)

		// Update tokenType for tracking (used in splitTurtleStatements logic)
		if ch == '<' {
			tokenType = "iri"
		} else if ch == '"' || ch == '\'' {
			if consumed == 2 {
				tokenType = "long_string"
			} else {
				tokenType = "string"
			}
		} else if ch == '[' {
			tokenType = "bracket"
		} else if ch == ']' || ch == '(' || ch == ')' {
			if ch == ']' || ch == ')' {
				tokenType = ""
			} else {
				tokenType = "paren"
			}
		} else if !state.inString && !state.inIRI {
			tokenType = ""
		}

		i += consumed

		if ch == '.' && state.isBalanced() {
			if i > 0 && input[i-1] >= '0' && input[i-1] <= '9' {
				next := byte(0)
				if i+1 < len(input) {
					next = input[i+1]
				}
				if (next >= 'a' && next <= 'z') || (next >= 'A' && next <= 'Z') || next == '_' {
					continue
				}
			}
			if tokenType == "string" || tokenType == "long_string" || tokenType == "iri" {
				continue
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
				continue
			}
			if prev >= '0' && prev <= '9' && (next == 'e' || next == 'E') {
				continue
			}
			if next == 0 || next == ' ' || next == '\t' || next == '\r' || next == '\n' || next == ';' || next == ',' || next == ')' || next == ']' || next == '}' ||
				next == '<' || next == '_' || next == '[' || next == '(' || next == ':' || next == '@' || next == '{' ||
				(next >= 'a' && next <= 'z') || (next >= 'A' && next <= 'Z') || (next >= '0' && next <= '9') {
				statement := strings.TrimSpace(input[start : i+1])
				if statement != "" {
					statements = append(statements, statement)
				}
				start = i + 1
				state.reset()
				tokenType = ""
			}
		}
	}
	rest := strings.TrimSpace(input[start:])
	if rest != "" {
		statements = append(statements, rest)
	}
	return statements
}
