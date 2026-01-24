package rdf

import (
	"bytes"
	"strings"
	"testing"
)

// TestTurtlePrefixOrderingDeterministic verifies that Turtle output
// produces deterministic prefix ordering across multiple runs.
func TestTurtlePrefixOrderingDeterministic(t *testing.T) {
	// Create statements that will use multiple prefixes
	stmts := []Statement{
		NewTriple(
			IRI{Value: "http://example.org/z"},
			IRI{Value: "http://example.org/p"},
			IRI{Value: "http://example.org/o"},
		),
		NewTriple(
			IRI{Value: "http://example.org/a"},
			IRI{Value: "http://example.org/p"},
			IRI{Value: "http://example.org/o"},
		),
		NewTriple(
			IRI{Value: "http://example.org/m"},
			IRI{Value: "http://example.org/p"},
			IRI{Value: "http://example.org/o"},
		),
	}

	// Encode the same statements twice
	var buf1, buf2 bytes.Buffer
	enc1, err := NewWriter(&buf1, FormatTurtle)
	if err != nil {
		t.Fatalf("unexpected error creating encoder 1: %v", err)
	}
	enc2, err := NewWriter(&buf2, FormatTurtle)
	if err != nil {
		t.Fatalf("unexpected error creating encoder 2: %v", err)
	}

	// Write statements in the same order
	for _, stmt := range stmts {
		if err := enc1.Write(stmt); err != nil {
			t.Fatalf("unexpected error writing to encoder 1: %v", err)
		}
		if err := enc2.Write(stmt); err != nil {
			t.Fatalf("unexpected error writing to encoder 2: %v", err)
		}
	}

	if err := enc1.Close(); err != nil {
		t.Fatalf("unexpected error closing encoder 1: %v", err)
	}
	if err := enc2.Close(); err != nil {
		t.Fatalf("unexpected error closing encoder 2: %v", err)
	}

	output1 := buf1.String()
	output2 := buf2.String()

	// Output should be identical (deterministic)
	if output1 != output2 {
		t.Errorf("Turtle output is not deterministic:\nFirst run:\n%q\n\nSecond run:\n%q", output1, output2)
	}

	// Verify that prefixes are sorted alphabetically
	// Extract prefix declarations from output
	lines := strings.Split(output1, "\n")
	var prefixLines []string
	for _, line := range lines {
		if strings.HasPrefix(line, "@prefix") {
			prefixLines = append(prefixLines, line)
		}
	}

	// If we have multiple prefixes, they should be in sorted order
	if len(prefixLines) > 1 {
		for i := 1; i < len(prefixLines); i++ {
			if prefixLines[i-1] > prefixLines[i] {
				t.Errorf("Prefix declarations are not sorted alphabetically:\n%v", prefixLines)
				break
			}
		}
	}
}

// TestTriGPrefixOrderingDeterministic verifies that TriG output
// produces deterministic prefix ordering across multiple runs.
func TestTriGPrefixOrderingDeterministic(t *testing.T) {
	// Create quads that will use multiple prefixes
	quads := []Statement{
		NewQuad(
			IRI{Value: "http://example.org/z"},
			IRI{Value: "http://example.org/p"},
			IRI{Value: "http://example.org/o"},
			IRI{Value: "http://example.org/g"},
		),
		NewQuad(
			IRI{Value: "http://example.org/a"},
			IRI{Value: "http://example.org/p"},
			IRI{Value: "http://example.org/o"},
			IRI{Value: "http://example.org/g"},
		),
		NewQuad(
			IRI{Value: "http://example.org/m"},
			IRI{Value: "http://example.org/p"},
			IRI{Value: "http://example.org/o"},
			IRI{Value: "http://example.org/g"},
		),
	}

	// Encode the same quads twice
	var buf1, buf2 bytes.Buffer
	enc1, err := NewWriter(&buf1, FormatTriG)
	if err != nil {
		t.Fatalf("unexpected error creating encoder 1: %v", err)
	}
	enc2, err := NewWriter(&buf2, FormatTriG)
	if err != nil {
		t.Fatalf("unexpected error creating encoder 2: %v", err)
	}

	// Write quads in the same order
	for _, quad := range quads {
		if err := enc1.Write(quad); err != nil {
			t.Fatalf("unexpected error writing to encoder 1: %v", err)
		}
		if err := enc2.Write(quad); err != nil {
			t.Fatalf("unexpected error writing to encoder 2: %v", err)
		}
	}

	if err := enc1.Close(); err != nil {
		t.Fatalf("unexpected error closing encoder 1: %v", err)
	}
	if err := enc2.Close(); err != nil {
		t.Fatalf("unexpected error closing encoder 2: %v", err)
	}

	output1 := buf1.String()
	output2 := buf2.String()

	// Output should be identical (deterministic)
	if output1 != output2 {
		t.Errorf("TriG output is not deterministic:\nFirst run:\n%q\n\nSecond run:\n%q", output1, output2)
	}

	// Verify that prefixes are sorted alphabetically
	lines := strings.Split(output1, "\n")
	var prefixLines []string
	for _, line := range lines {
		if strings.HasPrefix(line, "@prefix") {
			prefixLines = append(prefixLines, line)
		}
	}

	// If we have multiple prefixes, they should be in sorted order
	if len(prefixLines) > 1 {
		for i := 1; i < len(prefixLines); i++ {
			if prefixLines[i-1] > prefixLines[i] {
				t.Errorf("Prefix declarations are not sorted alphabetically:\n%v", prefixLines)
				break
			}
		}
	}
}

// TestStatementOrderingPreserved verifies that statement order
// is preserved in output (for formats that support it).
func TestStatementOrderingPreserved(t *testing.T) {
	stmts := []Statement{
		NewTriple(IRI{Value: "http://example.org/s1"}, IRI{Value: "http://example.org/p"}, IRI{Value: "http://example.org/o1"}),
		NewTriple(IRI{Value: "http://example.org/s2"}, IRI{Value: "http://example.org/p"}, IRI{Value: "http://example.org/o2"}),
		NewTriple(IRI{Value: "http://example.org/s3"}, IRI{Value: "http://example.org/p"}, IRI{Value: "http://example.org/o3"}),
	}

	formats := []Format{FormatTurtle, FormatNTriples, FormatNQuads, FormatTriG}
	for _, format := range formats {
		var buf bytes.Buffer
		enc, err := NewWriter(&buf, format)
		if err != nil {
			t.Fatalf("format %s: unexpected error: %v", format, err)
		}

		for _, stmt := range stmts {
			if err := enc.Write(stmt); err != nil {
				t.Fatalf("format %s: unexpected write error: %v", format, err)
			}
		}

		if err := enc.Close(); err != nil {
			t.Fatalf("format %s: unexpected close error: %v", format, err)
		}

		// Parse back and verify order
		dec, err := NewReader(strings.NewReader(buf.String()), format)
		if err != nil {
			t.Fatalf("format %s: unexpected decode error: %v", format, err)
		}

		var parsed []Statement
		for {
			stmt, err := dec.Next()
			if err != nil {
				break
			}
			parsed = append(parsed, stmt)
		}

		if err := dec.Close(); err != nil {
			t.Fatalf("format %s: unexpected decoder close error: %v", format, err)
		}

		// Verify we got the same number of statements
		if len(parsed) != len(stmts) {
			t.Errorf("format %s: expected %d statements, got %d", format, len(stmts), len(parsed))
		}

		// For deterministic formats, verify order matches
		// (Note: JSON-LD is non-deterministic, so we skip order check for it)
		if format != FormatJSONLD && format != FormatRDFXML {
			for i, expected := range stmts {
				if i >= len(parsed) {
					break
				}
				got := parsed[i]
				// Compare subjects (simplified check)
				if expected.S.String() != got.S.String() {
					t.Errorf("format %s: statement %d order mismatch: expected subject %s, got %s",
						format, i, expected.S.String(), got.S.String())
				}
			}
		}
	}
}
