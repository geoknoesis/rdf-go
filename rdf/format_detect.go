package rdf

import (
	"io"
	"strings"
)

const (
	// formatDetectionBufferSize is the number of bytes to read for format detection
	formatDetectionBufferSize = 512
)

// detectFormatFromSample attempts to detect the RDF format from input by examining the first few bytes.
// It returns the detected format and whether detection was successful.
// Detection is based on format signatures and heuristics.
// This is an internal helper function.
func detectFormatFromSample(r io.Reader) (Format, bool) {
	// Read a sample of the input (first 512 bytes should be enough)
	buf := make([]byte, formatDetectionBufferSize)
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
			return FormatJSONLD, true
		}
		// Could still be JSON-LD even without explicit keywords
		if strings.HasPrefix(sample, "{") || strings.HasPrefix(sample, "[") {
			// Check if it's valid JSON structure
			if isValidJSONStructure(sample) {
				return FormatJSONLD, true
			}
		}
	}

	// Check for RDF/XML (starts with <?xml or <rdf:)
	if strings.HasPrefix(sample, "<?xml") || strings.HasPrefix(sample, "<rdf:") || strings.HasPrefix(sample, "<rdf ") {
		return FormatRDFXML, true
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
			return FormatTurtle, true
		}
		return FormatTurtle, true
	}

	// Check for N-Triples/N-Quads pattern (IRI <...> or blank node _:)
	// N-Triples/N-Quads start with < or contain _: and don't have prefixes or directives
	// Also check for blank node syntax _: which is used in N-Triples
	hasNTriplesPattern := (strings.HasPrefix(sample, "<") || strings.Contains(sample, " _:") || strings.HasPrefix(sample, "_:")) &&
		!strings.Contains(sample, "@prefix") && !strings.Contains(sample, "@base") &&
		!strings.Contains(strings.ToUpper(sample), "PREFIX") && !strings.Contains(strings.ToUpper(sample), "BASE") &&
		!strings.Contains(sample, "[") && !strings.Contains(sample, "(")

	if hasNTriplesPattern {
		// Check if it ends with . (N-Triples) or has 4th component (N-Quads)
		// For N-Triples: <s> <p> <o> . or <s> <p> _:b0 .
		// For N-Quads: <s> <p> <o> <g> .
		// Count angle brackets to guess
		angleCount := strings.Count(sample, "<")
		if angleCount >= 3 || strings.Contains(sample, " _:") || strings.HasPrefix(sample, "_:") {
			// Default to N-Triples (more common)
			return FormatNTriples, true
		}
	}

	// Check for Turtle patterns (prefixes, base IRIs, collections, blank node lists)
	// Turtle can have prefixes, base, or use : for prefixed names (but not _:), or [] for blank nodes
	hasTurtlePattern := strings.Contains(sample, "@prefix") || strings.Contains(sample, "@base") ||
		strings.Contains(strings.ToUpper(sample), "PREFIX") || strings.Contains(strings.ToUpper(sample), "BASE") ||
		strings.Contains(sample, "[") || strings.Contains(sample, "(")

	// Check for prefixed names (but exclude _: blank node syntax)
	if !hasTurtlePattern && strings.Contains(sample, ":") {
		// Check if it's a prefixed name (like ex:s) and not a blank node (_:b0)
		// Prefixed names typically appear after whitespace or at start
		parts := strings.Fields(sample)
		for _, part := range parts {
			if strings.Contains(part, ":") && !strings.HasPrefix(part, "_:") && !strings.HasPrefix(part, "<") {
				hasTurtlePattern = true
				break
			}
		}
	}

	if hasTurtlePattern {
		// Likely Turtle (uses prefixes, blank nodes, or collections)
		return FormatTurtle, true
	}

	// Default: unable to detect
	return "", false
}

// detectQuadFormat attempts to detect quad-capable RDF formats.
// This is an internal helper function.
func detectQuadFormat(r io.Reader) (Format, bool) {
	// Read a sample of the input
	buf := make([]byte, formatDetectionBufferSize)
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
			return FormatTriG, true
		}
		// If it has { and looks like Turtle, it's likely TriG
		if strings.Contains(sample, "{") && (strings.Contains(sample, "<") || strings.Contains(sample, ":")) {
			return FormatTriG, true
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
					return FormatNQuads, true
				}
			}
		}
		// Default to N-Quads if we can't tell (conservative)
		return FormatNQuads, true
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

// detectFormatAuto attempts to detect either triple or quad format.
// It returns the format as a string and a boolean indicating success.
// Note: This function reads from the reader, so the reader position will be advanced.
// If you need to preserve the reader position, use io.TeeReader or buffer the input.
func detectFormatAuto(r io.Reader) (string, bool) {
	// Read a sample first
	buf := make([]byte, formatDetectionBufferSize)
	n, err := r.Read(buf)
	if err != nil && err != io.EOF {
		return "", false
	}
	sample := string(buf[:n])

	// Try quad formats first (they're more specific)
	quadReader := strings.NewReader(sample)
	if quadFormat, ok := detectQuadFormat(quadReader); ok {
		return string(quadFormat), true
	}
	// Try triple formats
	tripleReader := strings.NewReader(sample)
	if tripleFormat, ok := detectFormatFromSample(tripleReader); ok {
		return string(tripleFormat), true
	}
	return "", false
}
