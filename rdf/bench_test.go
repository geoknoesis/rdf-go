package rdf

import (
	"bytes"
	"strings"
	"testing"
)

func BenchmarkNTriplesDecode(b *testing.B) {
	line := "<http://example.org/s> <http://example.org/p> \"v\" .\n"
	input := strings.Repeat(line, 1000)
	b.SetBytes(int64(len(input)))
	for i := 0; i < b.N; i++ {
		dec, _ := NewReader(strings.NewReader(input), FormatNTriples)
		for {
			_, err := dec.Next()
			if err != nil {
				break
			}
		}
	}
}

func BenchmarkNTriplesEncode(b *testing.B) {
	buf := &bytes.Buffer{}
	enc, _ := NewWriter(buf, FormatNTriples)
	stmt := Statement{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: Literal{Lexical: "v"},
		G: nil,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		_ = enc.Write(stmt)
		_ = enc.Flush()
	}
}

func BenchmarkTurtleDecode(b *testing.B) {
	input := "@prefix ex: <http://example.org/> .\nex:s ex:p \"v\" .\n"
	b.SetBytes(int64(len(input)))
	for i := 0; i < b.N; i++ {
		dec, _ := NewReader(strings.NewReader(input), FormatTurtle)
		for {
			_, err := dec.Next()
			if err != nil {
				break
			}
		}
	}
}
