package rdf

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
)

func ExampleNewTripleDecoder() {
	input := "<http://example.org/s> <http://example.org/p> \"v\" .\n"
	dec, err := NewTripleDecoder(strings.NewReader(input), TripleFormatNTriples)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer dec.Close()

	triple, err := dec.Next()
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("%s %s %s\n", triple.S.String(), triple.P.String(), triple.O.String())

	// Output:
	// http://example.org/s http://example.org/p "v"
}

func ExampleNewTripleEncoder() {
	var buf bytes.Buffer
	enc, err := NewTripleEncoder(&buf, TripleFormatNTriples)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	_ = enc.Write(Triple{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: Literal{Lexical: "v"},
	})
	_ = enc.Close()

	fmt.Print(buf.String())

	// Output:
	// <http://example.org/s> <http://example.org/p> "v" .
}

func ExampleParseTriples() {
	input := "<http://example.org/s> <http://example.org/p> \"v\" .\n" +
		"<http://example.org/s2> <http://example.org/p2> \"v2\" .\n"
	count := 0
	err := ParseTriples(context.Background(), strings.NewReader(input), TripleFormatNTriples, TripleHandlerFunc(func(Triple) error {
		count++
		return nil
	}))
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(count)

	// Output:
	// 2
}

func ExampleParseTriplesChan() {
	input := "<http://example.org/s> <http://example.org/p> \"v\" .\n"
	out, errs := ParseTriplesChan(context.Background(), strings.NewReader(input), TripleFormatNTriples)
	count := 0
	for range out {
		count++
	}
	if err, ok := <-errs; ok && err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(count)

	// Output:
	// 1
}

func ExampleTripleTerm() {
	quoted := TripleTerm{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: IRI{Value: "http://example.org/o"},
	}
	triple := Triple{
		S: quoted,
		P: IRI{Value: "http://example.org/said"},
		O: Literal{Lexical: "true"},
	}
	var buf bytes.Buffer
	enc, _ := NewTripleEncoder(&buf, TripleFormatNTriples)
	_ = enc.Write(triple)
	_ = enc.Close()
	fmt.Print(buf.String())

	// Output:
	// <<http://example.org/s http://example.org/p http://example.org/o>> <http://example.org/said> "true" .
}

func ExampleParseTripleFormat() {
	format, ok := ParseTripleFormat("ttl")
	if !ok {
		fmt.Println("unknown")
		return
	}
	fmt.Println(format)

	// Output:
	// turtle
}

func ExampleQuadFormatNQuads() {
	input := "<http://example.org/s> <http://example.org/p> <http://example.org/o> <http://example.org/g> .\n"
	dec, err := NewQuadDecoder(strings.NewReader(input), QuadFormatNQuads)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer dec.Close()

	quad, err := dec.Next()
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("%s %s %s %s\n", quad.S.String(), quad.P.String(), quad.O.String(), quad.G.String())

	// Output:
	// http://example.org/s http://example.org/p http://example.org/o http://example.org/g
}

func ExampleTripleFormatTurtle() {
	input := "@prefix ex: <http://example.org/> .\nex:s ex:p \"v\" .\n"
	dec, err := NewTripleDecoder(strings.NewReader(input), TripleFormatTurtle)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer dec.Close()

	triple, err := dec.Next()
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("%s %s %s\n", triple.S.String(), triple.P.String(), triple.O.String())

	// Output:
	// http://example.org/s http://example.org/p "v"
}

func ExampleQuadFormatTriG() {
	input := "<http://example.org/g> { <http://example.org/s> <http://example.org/p> <http://example.org/o> . }\n"
	dec, err := NewQuadDecoder(strings.NewReader(input), QuadFormatTriG)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer dec.Close()

	quad, err := dec.Next()
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("%s %s %s %s\n", quad.S.String(), quad.P.String(), quad.O.String(), quad.G.String())

	// Output:
	// http://example.org/s http://example.org/p http://example.org/o http://example.org/g
}

func ExampleTripleFormatRDFXML() {
	input := `<?xml version="1.0"?>
<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" xmlns:ex="http://example.org/">
  <rdf:Description rdf:about="http://example.org/s">
    <ex:p>v</ex:p>
  </rdf:Description>
</rdf:RDF>`
	dec, err := NewTripleDecoder(strings.NewReader(input), TripleFormatRDFXML)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer dec.Close()

	triple, err := dec.Next()
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("%s %s %s\n", triple.S.String(), triple.P.String(), triple.O.String())

	// Output:
	// http://example.org/s http://example.org/p "v"
}

func ExampleTripleFormatJSONLD() {
	input := `{"@context":{"ex":"http://example.org/"},"@id":"ex:s","ex:p":"v"}`
	dec, err := NewTripleDecoder(strings.NewReader(input), TripleFormatJSONLD)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer dec.Close()

	triple, err := dec.Next()
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("%s %s %s\n", triple.S.String(), triple.P.String(), triple.O.String())

	// Output:
	// http://example.org/s http://example.org/p "v"
}

func ExampleEOF() {
	input := "<http://example.org/s> <http://example.org/p> \"v\" .\n"
	dec, _ := NewTripleDecoder(strings.NewReader(input), TripleFormatNTriples)
	defer dec.Close()
	_, _ = dec.Next()
	_, err := dec.Next()
	if err == io.EOF {
		fmt.Println("eof")
	}

	// Output:
	// eof
}
