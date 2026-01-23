package rdf

import (
	"io"
	"strings"
)

// DetectFormat attempts to detect the RDF format from input by examining the first few bytes.
// It returns the detected format and whether detection was successful.
// Detection is based on format signatures and heuristics.
func DetectFormat(r io.Reader) (TripleFormat, bool) {
	// Read a sample of the input (first 512 bytes should be enough)
	buf := make([]byte, 512)
	n, err := r.Read(buf)
	if err != nil && err != io.EOF {
		return "", false
	}
	sample := string(buf[:n])

	// Trim whitespace to focus on content
	sample = strings.TrimSpace(sample)
	if len(sample) == 0 {
		return "", false
	}

	// Check for JSON-LD (starts with { or [)
	if strings.HasPrefix(sample, "{") || strings.HasPrefix(sample, "[") {
		// Check for JSON-LD keywords
		if strings.Contains(sample, "@context") || strings.Contains(sample, "@id") || strings.Contains(sample, "@type") {
			return TripleFormatJSONLD, true
		}
		// Could still be JSON-LD even without explicit keywords
		if strings.HasPrefix(sample, "{") || strings.HasPrefix(sample, "[") {
			// Check if it's valid JSON structure
			if isValidJSONStructure(sample) {
				return TripleFormatJSONLD, true
			}
		}
	}

	// Check for RDF/XML (starts with <?xml or <rdf:)
	if strings.HasPrefix(sample, "<?xml") || strings.HasPrefix(sample, "<rdf:") || strings.HasPrefix(sample, "<rdf ") {
		return TripleFormatRDFXML, true
	}

	// Check for Turtle/TriG directives (@prefix, @base, PREFIX, BASE)
	upper := strings.ToUpper(sample)
	if strings.HasPrefix(upper, "@PREFIX") || strings.HasPrefix(upper, "PREFIX") ||
		strings.HasPrefix(upper, "@BASE") || strings.HasPrefix(upper, "BASE") ||
		strings.HasPrefix(upper, "@VERSION") || strings.HasPrefix(upper, "VERSION") {
		// Check if it's TriG (has GRAPH keyword or {})
		if strings.Contains(upper, "GRAPH") || strings.Contains(sample, "{") {
			// Could be TriG, but we can only detect triple formats here
			// TriG is a quad format, so we'd need DetectQuadFormat
			return TripleFormatTurtle, true
		}
		return TripleFormatTurtle, true
	}

	// Check for N-Triples/N-Quads pattern (IRI <...> or blank node _:)
	if strings.HasPrefix(sample, "<") || strings.HasPrefix(sample, "_:") {
		// Check if it ends with . (N-Triples) or has 4th component (N-Quads)
		// For N-Triples: <s> <p> <o> .
		// For N-Quads: <s> <p> <o> <g> .
		// We can't distinguish reliably without parsing, but N-Triples is more common
		// Count angle brackets and spaces to guess
		angleCount := strings.Count(sample, "<")
		spaceCount := strings.Count(sample, " ")
		if angleCount >= 3 && spaceCount >= 2 {
			// Check if it has 4 IRIs (N-Quads) or 3 (N-Triples)
			// Simple heuristic: if there are 4 < before the first >, it might be N-Quads
			// But this is unreliable, so default to N-Triples
			return TripleFormatNTriples, true
		}
	}

	// Check for Turtle patterns (prefixes, base IRIs, collections, blank node lists)
	if strings.Contains(sample, ":") && (strings.Contains(sample, "<") || strings.Contains(sample, "[")) {
		// Likely Turtle (uses prefixes and IRIs/blank nodes)
		return TripleFormatTurtle, true
	}

	// Default: unable to detect
	return "", false
}

// DetectQuadFormat attempts to detect quad-capable RDF formats.
func DetectQuadFormat(r io.Reader) (QuadFormat, bool) {
	// Read a sample of the input
	buf := make([]byte, 512)
	n, err := r.Read(buf)
	if err != nil && err != io.EOF {
		return "", false
	}
	sample := string(buf[:n])

	// Trim whitespace
	sample = strings.TrimSpace(sample)
	if len(sample) == 0 {
		return "", false
	}

	// Check for TriG (has GRAPH keyword or graph blocks {})
	upper := strings.ToUpper(sample)
	if strings.Contains(upper, "GRAPH") || strings.Contains(sample, "{") {
		// Check for TriG directives
		if strings.HasPrefix(upper, "@PREFIX") || strings.HasPrefix(upper, "PREFIX") ||
			strings.HasPrefix(upper, "@BASE") || strings.HasPrefix(upper, "BASE") {
			return QuadFormatTriG, true
		}
		// If it has { and looks like Turtle, it's likely TriG
		if strings.Contains(sample, "{") && (strings.Contains(sample, "<") || strings.Contains(sample, ":")) {
			return QuadFormatTriG, true
		}
	}

	// Check for N-Quads (4 IRIs per line: <s> <p> <o> <g> .)
	if strings.HasPrefix(sample, "<") {
		// Count IRIs - N-Quads has 4, N-Triples has 3
		// Simple heuristic: look for pattern with 4 angle-bracketed terms
		lines := strings.Split(sample, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			// If we see 4 < before the ., it's likely N-Quads
			if strings.HasSuffix(line, ".") {
				// Count all < in the line
				totalAngles := strings.Count(line, "<")
				if totalAngles >= 4 {
					return QuadFormatNQuads, true
				}
			}
		}
		// Default to N-Quads if we can't tell (conservative)
		return QuadFormatNQuads, true
	}

	return "", false
}

// isValidJSONStructure performs a basic check if the string looks like valid JSON.
func isValidJSONStructure(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return false
	}
	// Check for JSON delimiters
	first := s[0]
	last := s[len(s)-1]
	return (first == '{' && last == '}') || (first == '[' && last == ']')
}

// DetectFormatAuto attempts to detect either triple or quad format.
// It returns the format as a string and a boolean indicating success.
func DetectFormatAuto(r io.Reader) (string, bool) {
	// Try quad formats first (they're more specific)
	if quadFormat, ok := DetectQuadFormat(r); ok {
		return string(quadFormat), true
	}
	// Try triple formats
	if tripleFormat, ok := DetectFormat(r); ok {
		return string(tripleFormat), true
	}
	return "", false
}

