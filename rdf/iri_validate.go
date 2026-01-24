package rdf

import (
	"fmt"
	"net/url"
	"strings"
)

// ValidateIRI validates an IRI string according to RFC 3987.
// Returns an error if the IRI is invalid, nil otherwise.
//
// This function performs basic IRI validation:
// - Checks for valid scheme (required for absolute IRIs)
// - Validates IRI structure using Go's url.Parse
// - Ensures the IRI is well-formed
//
// Note: This is a basic validation. For full RFC 3987 compliance,
// consider using a specialized IRI validation library.
func ValidateIRI(iri string) error {
	if iri == "" {
		return fmt.Errorf("empty IRI")
	}

	// Parse the IRI using Go's url.Parse (which handles IRIs reasonably well)
	parsed, err := url.Parse(iri)
	if err != nil {
		return fmt.Errorf("invalid IRI syntax: %w", err)
	}

	// For absolute IRIs, scheme is required
	if parsed.Scheme == "" {
		// Relative IRIs are valid, but we should check if it looks like it should be absolute
		// If it starts with //, it's a network-path reference (needs scheme)
		if strings.HasPrefix(iri, "//") {
			return fmt.Errorf("relative IRI without scheme: %s", iri)
		}
		// If it doesn't start with /, it might be a relative IRI (which is valid)
		// But if it contains : and doesn't start with /, it might be missing a scheme
		if strings.Contains(iri, ":") && !strings.HasPrefix(iri, "/") && !strings.HasPrefix(iri, "./") && !strings.HasPrefix(iri, "../") {
			// Check if the part before : looks like it could be a scheme
			parts := strings.SplitN(iri, ":", 2)
			if len(parts) == 2 {
				scheme := parts[0]
				// Basic scheme validation: should be alphanumeric with +, -, . allowed
				validScheme := true
				if len(scheme) == 0 {
					validScheme = false
				}
				for _, r := range scheme {
					if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
						(r >= '0' && r <= '9') || r == '+' || r == '-' || r == '.') {
						validScheme = false
						break
					}
				}
				if !validScheme {
					return fmt.Errorf("IRI appears to be missing a scheme: %s", iri)
				}
			}
		}
	} else {
		// Validate scheme
		if len(parsed.Scheme) == 0 {
			return fmt.Errorf("empty scheme in IRI: %s", iri)
		}
		// Scheme should start with a letter
		first := parsed.Scheme[0]
		if !((first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z')) {
			return fmt.Errorf("scheme must start with a letter: %s", iri)
		}
	}

	// Check for invalid characters (basic check)
	// IRIs can contain Unicode characters, but we'll do basic ASCII validation
	for i, r := range iri {
		// Control characters (except space which might be percent-encoded)
		if r < 0x20 && r != '\t' && r != '\n' && r != '\r' {
			return fmt.Errorf("invalid control character at position %d in IRI: %s", i, iri)
		}
		// Some characters that should be percent-encoded
		if r == '<' || r == '>' || r == '"' || r == '{' || r == '}' || r == '|' || r == '^' || r == '`' || r == '\\' {
			// These might be valid if percent-encoded, but raw they're problematic
			// We'll be lenient and only warn about obviously problematic cases
			if r == '<' || r == '>' {
				return fmt.Errorf("invalid character '%c' at position %d in IRI (should be percent-encoded): %s", r, i, iri)
			}
		}
	}

	return nil
}
