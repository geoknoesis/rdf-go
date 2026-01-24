package rdf

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

// PerformanceRegressionTest provides a framework for detecting performance regressions.
// These tests can be run in CI to ensure performance doesn't degrade over time.

// performanceThresholds defines acceptable performance thresholds for various operations.
// These values should be updated based on actual benchmark results.
var performanceThresholds = map[string]time.Duration{
	"TurtleDecode100":   10 * time.Millisecond,
	"NTriplesDecode100": 5 * time.Millisecond,
	"JSONLDDecode10":    20 * time.Millisecond,
	"RDFXMLDecode10":    15 * time.Millisecond,
	"TurtleEncode5":     1 * time.Millisecond,
	"NTriplesEncode5":   1 * time.Millisecond,
	"JSONLDEncode5":     2 * time.Millisecond,
	"RDFXMLEncode5":     3 * time.Millisecond,
}

// TestPerformanceRegression_TurtleDecode checks that Turtle decoding performance hasn't regressed.
func TestPerformanceRegression_TurtleDecode(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance regression test in short mode")
	}

	input := strings.Repeat(benchTurtleInput, 100)
	threshold := performanceThresholds["TurtleDecode100"]

	start := time.Now()
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
	if err != nil {
		t.Fatalf("failed to create reader: %v", err)
	}
	count := 0
	for {
		_, err := dec.Next()
		if err != nil {
			break
		}
		count++
	}
	dec.Close()
	duration := time.Since(start)

	if duration > threshold {
		t.Errorf("Turtle decode performance regression: %v exceeds threshold %v", duration, threshold)
	}
}

// TestPerformanceRegression_NTriplesDecode checks that N-Triples decoding performance hasn't regressed.
func TestPerformanceRegression_NTriplesDecode(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance regression test in short mode")
	}

	input := strings.Repeat(benchNTriplesInput, 100)
	threshold := performanceThresholds["NTriplesDecode100"]

	start := time.Now()
	dec, err := NewReader(strings.NewReader(input), FormatNTriples)
	if err != nil {
		t.Fatalf("failed to create reader: %v", err)
	}
	count := 0
	for {
		_, err := dec.Next()
		if err != nil {
			break
		}
		count++
	}
	dec.Close()
	duration := time.Since(start)

	if duration > threshold {
		t.Errorf("N-Triples decode performance regression: %v exceeds threshold %v", duration, threshold)
	}
}

// TestPerformanceRegression_JSONLDDecode checks that JSON-LD decoding performance hasn't regressed.
func TestPerformanceRegression_JSONLDDecode(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance regression test in short mode")
	}

	input := strings.Repeat(benchJSONLDInput, 10)
	threshold := performanceThresholds["JSONLDDecode10"]

	start := time.Now()
	dec, err := NewReader(strings.NewReader(input), FormatJSONLD)
	if err != nil {
		t.Fatalf("failed to create reader: %v", err)
	}
	count := 0
	for {
		_, err := dec.Next()
		if err != nil {
			break
		}
		count++
	}
	dec.Close()
	duration := time.Since(start)

	if duration > threshold {
		t.Errorf("JSON-LD decode performance regression: %v exceeds threshold %v", duration, threshold)
	}
}

// TestPerformanceRegression_RDFXMLDecode checks that RDF/XML decoding performance hasn't regressed.
func TestPerformanceRegression_RDFXMLDecode(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance regression test in short mode")
	}

	input := strings.Repeat(benchRDFXMLInput, 10)
	threshold := performanceThresholds["RDFXMLDecode10"]

	start := time.Now()
	dec, err := NewReader(strings.NewReader(input), FormatRDFXML)
	if err != nil {
		t.Fatalf("failed to create reader: %v", err)
	}
	count := 0
	for {
		_, err := dec.Next()
		if err != nil {
			break
		}
		count++
	}
	dec.Close()
	duration := time.Since(start)

	if duration > threshold {
		t.Errorf("RDF/XML decode performance regression: %v exceeds threshold %v", duration, threshold)
	}
}

// TestPerformanceRegression_Encode checks that encoding performance hasn't regressed.
func TestPerformanceRegression_Encode(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance regression test in short mode")
	}

	stmts := []Statement{
		{S: IRI{Value: "http://example.org/s1"}, P: IRI{Value: "http://example.org/p1"}, O: IRI{Value: "http://example.org/o1"}, G: nil},
		{S: IRI{Value: "http://example.org/s2"}, P: IRI{Value: "http://example.org/p2"}, O: IRI{Value: "http://example.org/o2"}, G: nil},
		{S: IRI{Value: "http://example.org/s3"}, P: IRI{Value: "http://example.org/p3"}, O: IRI{Value: "http://example.org/o3"}, G: nil},
		{S: IRI{Value: "http://example.org/s4"}, P: IRI{Value: "http://example.org/p4"}, O: IRI{Value: "http://example.org/o4"}, G: nil},
		{S: IRI{Value: "http://example.org/s5"}, P: IRI{Value: "http://example.org/p5"}, O: IRI{Value: "http://example.org/o5"}, G: nil},
	}

	formats := []struct {
		format    Format
		threshold time.Duration
		name      string
	}{
		{FormatTurtle, performanceThresholds["TurtleEncode5"], "Turtle"},
		{FormatNTriples, performanceThresholds["NTriplesEncode5"], "N-Triples"},
		{FormatJSONLD, performanceThresholds["JSONLDEncode5"], "JSON-LD"},
		{FormatRDFXML, performanceThresholds["RDFXMLEncode5"], "RDF/XML"},
	}

	for _, fmt := range formats {
		t.Run(fmt.name, func(t *testing.T) {
			start := time.Now()
			var buf bytes.Buffer
			enc, err := NewWriter(&buf, fmt.format)
			if err != nil {
				t.Fatalf("failed to create writer: %v", err)
			}
			for _, stmt := range stmts {
				if err := enc.Write(stmt); err != nil {
					t.Fatalf("failed to write statement: %v", err)
				}
			}
			if err := enc.Close(); err != nil {
				t.Fatalf("failed to close writer: %v", err)
			}
			duration := time.Since(start)

			if duration > fmt.threshold {
				t.Errorf("%s encode performance regression: %v exceeds threshold %v", fmt.name, duration, fmt.threshold)
			}
		})
	}
}
