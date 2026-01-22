package rdf

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
)

func ExampleNewDecoder() {
	input := "<http://example.org/s> <http://example.org/p> \"v\" .\n"
	dec, err := NewDecoder(strings.NewReader(input), FormatNTriples)
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
	fmt.Printf("%s %s %s\n", quad.S.String(), quad.P.String(), quad.O.String())

	// Output:
	// http://example.org/s http://example.org/p "v"
}

func ExampleNewEncoder() {
	var buf bytes.Buffer
	enc, err := NewEncoder(&buf, FormatNTriples)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	_ = enc.Write(Quad{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: Literal{Lexical: "v"},
	})
	_ = enc.Close()

	fmt.Print(buf.String())

	// Output:
	// <http://example.org/s> <http://example.org/p> "v" .
}

func ExampleParse() {
	input := "<http://example.org/s> <http://example.org/p> \"v\" .\n" +
		"<http://example.org/s2> <http://example.org/p2> \"v2\" .\n"
	count := 0
	err := Parse(context.Background(), strings.NewReader(input), FormatNTriples, HandlerFunc(func(Quad) error {
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

func ExampleParseChan() {
	input := "<http://example.org/s> <http://example.org/p> \"v\" .\n"
	out, errs := ParseChan(context.Background(), strings.NewReader(input), FormatNTriples)
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
	quad := Quad{
		S: quoted,
		P: IRI{Value: "http://example.org/said"},
		O: Literal{Lexical: "true"},
	}
	var buf bytes.Buffer
	enc, _ := NewEncoder(&buf, FormatNTriples)
	_ = enc.Write(quad)
	_ = enc.Close()
	fmt.Print(buf.String())

	// Output:
	// <<http://example.org/s http://example.org/p http://example.org/o>> <http://example.org/said> "true" .
}

func ExampleParseFormat() {
	format, ok := ParseFormat("ttl")
	if !ok {
		fmt.Println("unknown")
		return
	}
	fmt.Println(format)

	// Output:
	// turtle
}

func ExampleFormatNQuads() {
	input := "<http://example.org/s> <http://example.org/p> <http://example.org/o> <http://example.org/g> .\n"
	dec, err := NewDecoder(strings.NewReader(input), FormatNQuads)
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

func ExampleFormatTurtle() {
	input := "@prefix ex: <http://example.org/> .\nex:s ex:p \"v\" .\n"
	dec, err := NewDecoder(strings.NewReader(input), FormatTurtle)
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
	fmt.Printf("%s %s %s\n", quad.S.String(), quad.P.String(), quad.O.String())

	// Output:
	// http://example.org/s http://example.org/p "v"
}

func ExampleFormatTriG() {
	input := "<http://example.org/g> { <http://example.org/s> <http://example.org/p> <http://example.org/o> . }\n"
	dec, err := NewDecoder(strings.NewReader(input), FormatTriG)
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

func ExampleFormatRDFXML() {
	input := `<?xml version="1.0"?>
<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" xmlns:ex="http://example.org/">
  <rdf:Description rdf:about="http://example.org/s">
    <ex:p>v</ex:p>
  </rdf:Description>
</rdf:RDF>`
	dec, err := NewDecoder(strings.NewReader(input), FormatRDFXML)
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
	fmt.Printf("%s %s %s\n", quad.S.String(), quad.P.String(), quad.O.String())

	// Output:
	// http://example.org/s http://example.org/p "v"
}

func ExampleFormatJSONLD() {
	input := `{"@context":{"ex":"http://example.org/"},"@id":"ex:s","ex:p":"v"}`
	dec, err := NewDecoder(strings.NewReader(input), FormatJSONLD)
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
	fmt.Printf("%s %s %s\n", quad.S.String(), quad.P.String(), quad.O.String())

	// Output:
	// http://example.org/s http://example.org/p "v"
}

func ExampleEOF() {
	input := "<http://example.org/s> <http://example.org/p> \"v\" .\n"
	dec, _ := NewDecoder(strings.NewReader(input), FormatNTriples)
	defer dec.Close()
	_, _ = dec.Next()
	_, err := dec.Next()
	if err == io.EOF {
		fmt.Println("eof")
	}

	// Output:
	// eof
}
