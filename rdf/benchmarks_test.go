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
		dec := NewJSONLDTripleDecoder(strings.NewReader(input), JSONLDOptions{})
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
		_, _ = DetectFormat(strings.NewReader(input))
	}
}

