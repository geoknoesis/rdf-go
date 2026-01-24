package rdf

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
)

func ExampleNewReader() {
	input := "<http://example.org/s> <http://example.org/p> <http://example.org/o> .\n"
	dec, err := NewReader(strings.NewReader(input), FormatNTriples)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer dec.Close()

	stmt, err := dec.Next()
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("%s %s %s\n", stmt.S.String(), stmt.P.String(), stmt.O.String())

	// Output:
	// http://example.org/s http://example.org/p http://example.org/o
}

func ExampleNewReader_autoDetect() {
	input := "<http://example.org/s> <http://example.org/p> <http://example.org/o> .\n"
	dec, err := NewReader(strings.NewReader(input), FormatAuto)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer dec.Close()

	stmt, err := dec.Next()
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("%s %s %s\n", stmt.S.String(), stmt.P.String(), stmt.O.String())

	// Output:
	// http://example.org/s http://example.org/p http://example.org/o
}

func ExampleNewWriter() {
	var buf bytes.Buffer
	enc, err := NewWriter(&buf, FormatNTriples)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	// G can be omitted (defaults to nil for triples)
	// G can be omitted (defaults to nil for triples)
	_ = enc.Write(Statement{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: Literal{Lexical: "v"},
		// G omitted - defaults to nil (triple)
	})
	_ = enc.Close()

	fmt.Print(buf.String())

	// Output:
	// <http://example.org/s> <http://example.org/p> "v" .
}

func ExampleParse() {
	input := "<http://example.org/s> <http://example.org/p> <http://example.org/o> .\n" +
		"<http://example.org/s2> <http://example.org/p2> <http://example.org/o2> .\n"
	count := 0
	err := Parse(context.Background(), strings.NewReader(input), FormatNTriples, func(Statement) error {
		count++
		return nil
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(count)

	// Output:
	// 2
}

func ExampleTripleTerm() {
	quoted := TripleTerm{
		S: IRI{Value: "http://example.org/s"},
		P: IRI{Value: "http://example.org/p"},
		O: IRI{Value: "http://example.org/o"},
	}
	// G can be omitted (defaults to nil for triples)
	stmt := Statement{
		S: quoted,
		P: IRI{Value: "http://example.org/said"},
		O: Literal{Lexical: "true"},
		// G omitted - defaults to nil (triple)
	}
	var buf bytes.Buffer
	enc, _ := NewWriter(&buf, FormatNTriples)
	_ = enc.Write(stmt)
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
	dec, err := NewReader(strings.NewReader(input), FormatNQuads)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer dec.Close()

	stmt, err := dec.Next()
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("%s %s %s %s\n", stmt.S.String(), stmt.P.String(), stmt.O.String(), stmt.G.String())

	// Output:
	// http://example.org/s http://example.org/p http://example.org/o http://example.org/g
}

func ExampleFormatTurtle() {
	input := "@prefix ex: <http://example.org/> .\nex:s ex:p \"v\" .\n"
	dec, err := NewReader(strings.NewReader(input), FormatTurtle)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer dec.Close()

	stmt, err := dec.Next()
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("%s %s %s\n", stmt.S.String(), stmt.P.String(), stmt.O.String())

	// Output:
	// http://example.org/s http://example.org/p "v"
}

func ExampleFormatTriG() {
	input := "<http://example.org/g> { <http://example.org/s> <http://example.org/p> <http://example.org/o> . }\n"
	dec, err := NewReader(strings.NewReader(input), FormatTriG)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer dec.Close()

	stmt, err := dec.Next()
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("%s %s %s %s\n", stmt.S.String(), stmt.P.String(), stmt.O.String(), stmt.G.String())

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
	dec, err := NewReader(strings.NewReader(input), FormatRDFXML)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer dec.Close()

	stmt, err := dec.Next()
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("%s %s %s\n", stmt.S.String(), stmt.P.String(), stmt.O.String())

	// Output:
	// http://example.org/s http://example.org/p "v"
}

func ExampleFormatJSONLD() {
	input := `{"@context":{"ex":"http://example.org/"},"@id":"ex:s","ex:p":"v"}`
	dec, err := NewReader(strings.NewReader(input), FormatJSONLD)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer dec.Close()

	stmt, err := dec.Next()
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("%s %s %s\n", stmt.S.String(), stmt.P.String(), stmt.O.String())

	// Output:
	// http://example.org/s http://example.org/p "v"
}

func ExampleEOF() {
	input := "<http://example.org/s> <http://example.org/p> <http://example.org/o> .\n"
	dec, _ := NewReader(strings.NewReader(input), FormatNTriples)
	defer dec.Close()
	_, _ = dec.Next()
	_, err := dec.Next()
	if err == io.EOF {
		fmt.Println("eof")
	}

	// Output:
	// eof
}
