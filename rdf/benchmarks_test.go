package rdf

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

// Benchmark data
var (
	benchTurtleInput = `@prefix ex: <http://example.org/> .
ex:s1 ex:p1 ex:o1 .
ex:s2 ex:p2 ex:o2 .
ex:s3 ex:p3 ex:o3 .
ex:s4 ex:p4 ex:o4 .
ex:s5 ex:p5 ex:o5 .
`

	benchNTriplesInput = `<http://example.org/s1> <http://example.org/p1> <http://example.org/o1> .
<http://example.org/s2> <http://example.org/p2> <http://example.org/o2> .
<http://example.org/s3> <http://example.org/p3> <http://example.org/o3> .
<http://example.org/s4> <http://example.org/p4> <http://example.org/o4> .
<http://example.org/s5> <http://example.org/p5> <http://example.org/o5> .
`

	benchTriGInput = `@prefix ex: <http://example.org/> .
GRAPH ex:g1 {
  ex:s1 ex:p1 ex:o1 .
  ex:s2 ex:p2 ex:o2 .
}
GRAPH ex:g2 {
  ex:s3 ex:p3 ex:o3 .
  ex:s4 ex:p4 ex:o4 .
}
`

	benchJSONLDInput = `{
  "@context": {"ex": "http://example.org/"},
  "@graph": [
    {"@id": "ex:s1", "ex:p1": "o1"},
    {"@id": "ex:s2", "ex:p2": "o2"},
    {"@id": "ex:s3", "ex:p3": "o3"},
    {"@id": "ex:s4", "ex:p4": "o4"},
    {"@id": "ex:s5", "ex:p5": "o5"}
  ]
}`

	benchRDFXMLInput = `<?xml version="1.0"?>
<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#"
         xmlns:ex="http://example.org/">
  <rdf:Description rdf:about="http://example.org/s1">
    <ex:p1 rdf:resource="http://example.org/o1"/>
  </rdf:Description>
  <rdf:Description rdf:about="http://example.org/s2">
    <ex:p2 rdf:resource="http://example.org/o2"/>
  </rdf:Description>
  <rdf:Description rdf:about="http://example.org/s3">
    <ex:p3 rdf:resource="http://example.org/o3"/>
  </rdf:Description>
</rdf:RDF>`
)

func BenchmarkTurtleDecodeLarge(b *testing.B) {
	input := strings.Repeat(benchTurtleInput, 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dec, err := NewReader(strings.NewReader(input), FormatTurtle)
		if err != nil {
			b.Fatal(err)
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
	}
}

func BenchmarkNTriplesDecodeLarge(b *testing.B) {
	input := strings.Repeat(benchNTriplesInput, 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dec, err := NewReader(strings.NewReader(input), FormatNTriples)
		if err != nil {
			b.Fatal(err)
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
	}
}

func BenchmarkTriGDecode(b *testing.B) {
	input := strings.Repeat(benchTriGInput, 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dec, err := NewReader(strings.NewReader(input), FormatTriG)
		if err != nil {
			b.Fatal(err)
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
	}
}

func BenchmarkJSONLDDecode(b *testing.B) {
	input := strings.Repeat(benchJSONLDInput, 10)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dec, _ := NewReader(strings.NewReader(input), FormatJSONLD)
		count := 0
		for {
			_, err := dec.Next()
			if err != nil {
				break
			}
			count++
		}
		dec.Close()
	}
}

func BenchmarkTurtleEncode(b *testing.B) {
	stmts := []Statement{
		{S: IRI{Value: "http://example.org/s1"}, P: IRI{Value: "http://example.org/p1"}, O: IRI{Value: "http://example.org/o1"}, G: nil},
		{S: IRI{Value: "http://example.org/s2"}, P: IRI{Value: "http://example.org/p2"}, O: IRI{Value: "http://example.org/o2"}, G: nil},
		{S: IRI{Value: "http://example.org/s3"}, P: IRI{Value: "http://example.org/p3"}, O: IRI{Value: "http://example.org/o3"}, G: nil},
		{S: IRI{Value: "http://example.org/s4"}, P: IRI{Value: "http://example.org/p4"}, O: IRI{Value: "http://example.org/o4"}, G: nil},
		{S: IRI{Value: "http://example.org/s5"}, P: IRI{Value: "http://example.org/p5"}, O: IRI{Value: "http://example.org/o5"}, G: nil},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		enc, err := NewWriter(&buf, FormatTurtle)
		if err != nil {
			b.Fatal(err)
		}
		for _, stmt := range stmts {
			if err := enc.Write(stmt); err != nil {
				b.Fatal(err)
			}
		}
		if err := enc.Close(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNTriplesEncodeLarge(b *testing.B) {
	stmts := []Statement{
		{S: IRI{Value: "http://example.org/s1"}, P: IRI{Value: "http://example.org/p1"}, O: IRI{Value: "http://example.org/o1"}, G: nil},
		{S: IRI{Value: "http://example.org/s2"}, P: IRI{Value: "http://example.org/p2"}, O: IRI{Value: "http://example.org/o2"}, G: nil},
		{S: IRI{Value: "http://example.org/s3"}, P: IRI{Value: "http://example.org/p3"}, O: IRI{Value: "http://example.org/o3"}, G: nil},
		{S: IRI{Value: "http://example.org/s4"}, P: IRI{Value: "http://example.org/p4"}, O: IRI{Value: "http://example.org/o4"}, G: nil},
		{S: IRI{Value: "http://example.org/s5"}, P: IRI{Value: "http://example.org/p5"}, O: IRI{Value: "http://example.org/o5"}, G: nil},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		enc, err := NewWriter(&buf, FormatNTriples)
		if err != nil {
			b.Fatal(err)
		}
		for _, stmt := range stmts {
			if err := enc.Write(stmt); err != nil {
				b.Fatal(err)
			}
		}
		if err := enc.Close(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseTriples(b *testing.B) {
	input := strings.Repeat(benchTurtleInput, 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		count := 0
		err := Parse(context.Background(), strings.NewReader(input), FormatTurtle, func(s Statement) error {
			count++
			return nil
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnescapeString(b *testing.B) {
	escaped := `Hello\nWorld\tTest\u0041\U0001F600`
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := UnescapeString(escaped)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkResolveIRI(b *testing.B) {
	base := "http://example.org/base/"
	relative := "path/to/resource"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = resolveIRI(base, relative)
	}
}

func BenchmarkFormatDetection(b *testing.B) {
	input := benchTurtleInput
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = detectFormatFromSample(strings.NewReader(input))
	}
}

// Large-scale benchmarks for GB-scale scenarios
// These benchmarks use larger inputs to measure performance at scale

// generateLargeTurtleInput generates approximately sizeBytes of Turtle data
func generateLargeTurtleInput(sizeBytes int) []byte {
	var buf strings.Builder
	stmt := "@prefix ex: <http://example.org/> .\nex:s ex:p ex:o .\n"
	stmtSize := len(stmt)
	repeats := sizeBytes / stmtSize
	if repeats < 1 {
		repeats = 1
	}
	for i := 0; i < repeats; i++ {
		buf.WriteString(stmt)
	}
	return []byte(buf.String())
}

// generateLargeNTriplesInput generates approximately sizeBytes of N-Triples data
func generateLargeNTriplesInput(sizeBytes int) []byte {
	var buf strings.Builder
	stmt := "<http://example.org/s> <http://example.org/p> <http://example.org/o> .\n"
	stmtSize := len(stmt)
	repeats := sizeBytes / stmtSize
	if repeats < 1 {
		repeats = 1
	}
	for i := 0; i < repeats; i++ {
		buf.WriteString(stmt)
	}
	return []byte(buf.String())
}

// BenchmarkTurtleDecode1MB benchmarks decoding 1MB of Turtle data
func BenchmarkTurtleDecode1MB(b *testing.B) {
	input := generateLargeTurtleInput(1 << 20) // 1MB
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		dec, err := NewReader(bytes.NewReader(input), FormatTurtle)
		if err != nil {
			b.Fatal(err)
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
	}
}

// BenchmarkTurtleDecode10MB benchmarks decoding 10MB of Turtle data
func BenchmarkTurtleDecode10MB(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping large benchmark in short mode")
	}
	input := generateLargeTurtleInput(10 << 20) // 10MB
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		dec, err := NewReader(bytes.NewReader(input), FormatTurtle)
		if err != nil {
			b.Fatal(err)
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
	}
}

// BenchmarkNTriplesDecode1MB benchmarks decoding 1MB of N-Triples data
func BenchmarkNTriplesDecode1MB(b *testing.B) {
	input := generateLargeNTriplesInput(1 << 20) // 1MB
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		dec, err := NewReader(bytes.NewReader(input), FormatNTriples)
		if err != nil {
			b.Fatal(err)
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
	}
}

// BenchmarkNTriplesDecode10MB benchmarks decoding 10MB of N-Triples data
func BenchmarkNTriplesDecode10MB(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping large benchmark in short mode")
	}
	input := generateLargeNTriplesInput(10 << 20) // 10MB
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		dec, err := NewReader(bytes.NewReader(input), FormatNTriples)
		if err != nil {
			b.Fatal(err)
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
	}
}

// BenchmarkTurtleEncode1MB benchmarks encoding 1MB worth of statements
func BenchmarkTurtleEncode1MB(b *testing.B) {
	// Generate approximately 1MB of statements
	stmt := Statement{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: IRI{Value: "http://example.org/o"},
	}
	stmtSize := len(stmt.S.String()) + len(stmt.P.String()) + len(stmt.O.String()) + 50 // approximate
	numStmts := (1 << 20) / stmtSize
	if numStmts < 1 {
		numStmts = 1
	}
	stmts := make([]Statement, numStmts)
	for i := range stmts {
		stmts[i] = stmt
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		enc, err := NewWriter(&buf, FormatTurtle)
		if err != nil {
			b.Fatal(err)
		}
		for _, stmt := range stmts {
			if err := enc.Write(stmt); err != nil {
				b.Fatal(err)
			}
		}
		if err := enc.Close(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkNTriplesEncode1MB benchmarks encoding 1MB worth of statements
func BenchmarkNTriplesEncode1MB(b *testing.B) {
	// Generate approximately 1MB of statements
	stmt := Statement{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: IRI{Value: "http://example.org/o"},
	}
	stmtSize := len(stmt.S.String()) + len(stmt.P.String()) + len(stmt.O.String()) + 50 // approximate
	numStmts := (1 << 20) / stmtSize
	if numStmts < 1 {
		numStmts = 1
	}
	stmts := make([]Statement, numStmts)
	for i := range stmts {
		stmts[i] = stmt
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		enc, err := NewWriter(&buf, FormatNTriples)
		if err != nil {
			b.Fatal(err)
		}
		for _, stmt := range stmts {
			if err := enc.Write(stmt); err != nil {
				b.Fatal(err)
			}
		}
		if err := enc.Close(); err != nil {
			b.Fatal(err)
		}
	}
}

// RDF/XML Benchmarks

func BenchmarkRDFXMLDecode(b *testing.B) {
	input := strings.Repeat(benchRDFXMLInput, 10)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		dec, err := NewReader(strings.NewReader(input), FormatRDFXML)
		if err != nil {
			b.Fatal(err)
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
	}
}

func BenchmarkRDFXMLEncode(b *testing.B) {
	stmts := []Statement{
		{S: IRI{Value: "http://example.org/s1"}, P: IRI{Value: "http://example.org/p1"}, O: IRI{Value: "http://example.org/o1"}, G: nil},
		{S: IRI{Value: "http://example.org/s2"}, P: IRI{Value: "http://example.org/p2"}, O: IRI{Value: "http://example.org/o2"}, G: nil},
		{S: IRI{Value: "http://example.org/s3"}, P: IRI{Value: "http://example.org/p3"}, O: IRI{Value: "http://example.org/o3"}, G: nil},
		{S: IRI{Value: "http://example.org/s4"}, P: IRI{Value: "http://example.org/p4"}, O: IRI{Value: "http://example.org/o4"}, G: nil},
		{S: IRI{Value: "http://example.org/s5"}, P: IRI{Value: "http://example.org/p5"}, O: IRI{Value: "http://example.org/o5"}, G: nil},
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		enc, err := NewWriter(&buf, FormatRDFXML)
		if err != nil {
			b.Fatal(err)
		}
		for _, stmt := range stmts {
			if err := enc.Write(stmt); err != nil {
				b.Fatal(err)
			}
		}
		if err := enc.Close(); err != nil {
			b.Fatal(err)
		}
	}
}

// JSON-LD Encode Benchmarks

func BenchmarkJSONLDEncode(b *testing.B) {
	stmts := []Statement{
		{S: IRI{Value: "http://example.org/s1"}, P: IRI{Value: "http://example.org/p1"}, O: IRI{Value: "http://example.org/o1"}, G: nil},
		{S: IRI{Value: "http://example.org/s2"}, P: IRI{Value: "http://example.org/p2"}, O: IRI{Value: "http://example.org/o2"}, G: nil},
		{S: IRI{Value: "http://example.org/s3"}, P: IRI{Value: "http://example.org/p3"}, O: IRI{Value: "http://example.org/o3"}, G: nil},
		{S: IRI{Value: "http://example.org/s4"}, P: IRI{Value: "http://example.org/p4"}, O: IRI{Value: "http://example.org/o4"}, G: nil},
		{S: IRI{Value: "http://example.org/s5"}, P: IRI{Value: "http://example.org/p5"}, O: IRI{Value: "http://example.org/o5"}, G: nil},
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		enc, err := NewWriter(&buf, FormatJSONLD)
		if err != nil {
			b.Fatal(err)
		}
		for _, stmt := range stmts {
			if err := enc.Write(stmt); err != nil {
				b.Fatal(err)
			}
		}
		if err := enc.Close(); err != nil {
			b.Fatal(err)
		}
	}
}

// Memory Allocation Benchmarks - All formats with allocation tracking

func BenchmarkTurtleDecodeAllocs(b *testing.B) {
	input := strings.Repeat(benchTurtleInput, 100)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dec, err := NewReader(strings.NewReader(input), FormatTurtle)
		if err != nil {
			b.Fatal(err)
		}
		for {
			_, err := dec.Next()
			if err != nil {
				break
			}
		}
		dec.Close()
	}
}

func BenchmarkNTriplesDecodeAllocs(b *testing.B) {
	input := strings.Repeat(benchNTriplesInput, 100)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dec, err := NewReader(strings.NewReader(input), FormatNTriples)
		if err != nil {
			b.Fatal(err)
		}
		for {
			_, err := dec.Next()
			if err != nil {
				break
			}
		}
		dec.Close()
	}
}

func BenchmarkJSONLDDecodeAllocs(b *testing.B) {
	input := strings.Repeat(benchJSONLDInput, 10)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dec, err := NewReader(strings.NewReader(input), FormatJSONLD)
		if err != nil {
			b.Fatal(err)
		}
		for {
			_, err := dec.Next()
			if err != nil {
				break
			}
		}
		dec.Close()
	}
}

func BenchmarkRDFXMLDecodeAllocs(b *testing.B) {
	input := strings.Repeat(benchRDFXMLInput, 10)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dec, err := NewReader(strings.NewReader(input), FormatRDFXML)
		if err != nil {
			b.Fatal(err)
		}
		for {
			_, err := dec.Next()
			if err != nil {
				break
			}
		}
		dec.Close()
	}
}

func BenchmarkTurtleEncodeAllocs(b *testing.B) {
	stmts := []Statement{
		{S: IRI{Value: "http://example.org/s1"}, P: IRI{Value: "http://example.org/p1"}, O: IRI{Value: "http://example.org/o1"}, G: nil},
		{S: IRI{Value: "http://example.org/s2"}, P: IRI{Value: "http://example.org/p2"}, O: IRI{Value: "http://example.org/o2"}, G: nil},
		{S: IRI{Value: "http://example.org/s3"}, P: IRI{Value: "http://example.org/p3"}, O: IRI{Value: "http://example.org/o3"}, G: nil},
		{S: IRI{Value: "http://example.org/s4"}, P: IRI{Value: "http://example.org/p4"}, O: IRI{Value: "http://example.org/o4"}, G: nil},
		{S: IRI{Value: "http://example.org/s5"}, P: IRI{Value: "http://example.org/p5"}, O: IRI{Value: "http://example.org/o5"}, G: nil},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		enc, err := NewWriter(&buf, FormatTurtle)
		if err != nil {
			b.Fatal(err)
		}
		for _, stmt := range stmts {
			if err := enc.Write(stmt); err != nil {
				b.Fatal(err)
			}
		}
		if err := enc.Close(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNTriplesEncodeAllocs(b *testing.B) {
	stmts := []Statement{
		{S: IRI{Value: "http://example.org/s1"}, P: IRI{Value: "http://example.org/p1"}, O: IRI{Value: "http://example.org/o1"}, G: nil},
		{S: IRI{Value: "http://example.org/s2"}, P: IRI{Value: "http://example.org/p2"}, O: IRI{Value: "http://example.org/o2"}, G: nil},
		{S: IRI{Value: "http://example.org/s3"}, P: IRI{Value: "http://example.org/p3"}, O: IRI{Value: "http://example.org/o3"}, G: nil},
		{S: IRI{Value: "http://example.org/s4"}, P: IRI{Value: "http://example.org/p4"}, O: IRI{Value: "http://example.org/o4"}, G: nil},
		{S: IRI{Value: "http://example.org/s5"}, P: IRI{Value: "http://example.org/p5"}, O: IRI{Value: "http://example.org/o5"}, G: nil},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		enc, err := NewWriter(&buf, FormatNTriples)
		if err != nil {
			b.Fatal(err)
		}
		for _, stmt := range stmts {
			if err := enc.Write(stmt); err != nil {
				b.Fatal(err)
			}
		}
		if err := enc.Close(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJSONLDEncodeAllocs(b *testing.B) {
	stmts := []Statement{
		{S: IRI{Value: "http://example.org/s1"}, P: IRI{Value: "http://example.org/p1"}, O: IRI{Value: "http://example.org/o1"}, G: nil},
		{S: IRI{Value: "http://example.org/s2"}, P: IRI{Value: "http://example.org/p2"}, O: IRI{Value: "http://example.org/o2"}, G: nil},
		{S: IRI{Value: "http://example.org/s3"}, P: IRI{Value: "http://example.org/p3"}, O: IRI{Value: "http://example.org/o3"}, G: nil},
		{S: IRI{Value: "http://example.org/s4"}, P: IRI{Value: "http://example.org/p4"}, O: IRI{Value: "http://example.org/o4"}, G: nil},
		{S: IRI{Value: "http://example.org/s5"}, P: IRI{Value: "http://example.org/p5"}, O: IRI{Value: "http://example.org/o5"}, G: nil},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		enc, err := NewWriter(&buf, FormatJSONLD)
		if err != nil {
			b.Fatal(err)
		}
		for _, stmt := range stmts {
			if err := enc.Write(stmt); err != nil {
				b.Fatal(err)
			}
		}
		if err := enc.Close(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRDFXMLEncodeAllocs(b *testing.B) {
	stmts := []Statement{
		{S: IRI{Value: "http://example.org/s1"}, P: IRI{Value: "http://example.org/p1"}, O: IRI{Value: "http://example.org/o1"}, G: nil},
		{S: IRI{Value: "http://example.org/s2"}, P: IRI{Value: "http://example.org/p2"}, O: IRI{Value: "http://example.org/o2"}, G: nil},
		{S: IRI{Value: "http://example.org/s3"}, P: IRI{Value: "http://example.org/p3"}, O: IRI{Value: "http://example.org/o3"}, G: nil},
		{S: IRI{Value: "http://example.org/s4"}, P: IRI{Value: "http://example.org/p4"}, O: IRI{Value: "http://example.org/o4"}, G: nil},
		{S: IRI{Value: "http://example.org/s5"}, P: IRI{Value: "http://example.org/p5"}, O: IRI{Value: "http://example.org/o5"}, G: nil},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		enc, err := NewWriter(&buf, FormatRDFXML)
		if err != nil {
			b.Fatal(err)
		}
		for _, stmt := range stmts {
			if err := enc.Write(stmt); err != nil {
				b.Fatal(err)
			}
		}
		if err := enc.Close(); err != nil {
			b.Fatal(err)
		}
	}
}

// Performance regression test helpers
// These benchmarks can be used in CI to detect performance regressions

// BenchmarkFormatComparison compares performance across all formats
func BenchmarkFormatComparison(b *testing.B) {
	formats := []Format{FormatTurtle, FormatNTriples, FormatTriG, FormatJSONLD, FormatRDFXML}

	for _, format := range formats {
		b.Run(format.String(), func(b *testing.B) {
			var input string
			switch format {
			case FormatTurtle:
				input = strings.Repeat(benchTurtleInput, 100)
			case FormatNTriples:
				input = strings.Repeat(benchNTriplesInput, 100)
			case FormatTriG:
				input = strings.Repeat(benchTriGInput, 100)
			case FormatJSONLD:
				input = strings.Repeat(benchJSONLDInput, 10)
			case FormatRDFXML:
				input = strings.Repeat(benchRDFXMLInput, 10)
			default:
				b.Skip("format not supported")
			}

			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				dec, err := NewReader(strings.NewReader(input), format)
				if err != nil {
					b.Fatal(err)
				}
				for {
					_, err := dec.Next()
					if err != nil {
						break
					}
				}
				dec.Close()
			}
		})
	}
}
