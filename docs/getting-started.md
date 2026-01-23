# Getting Started

This guide will help you get started with `rdf-go` by installing the library and writing your first program.

## Installation

Install the library using Go modules:

```bash
go get github.com/geoknoesis/rdf-go
```

## Your First Program

Let's create a simple program that reads and parses RDF data.

### Reading RDF Data

Here's a minimal example that reads N-Triples format:

```go
package main

import (
    "fmt"
    "io"
    "strings"
    "github.com/geoknoesis/rdf-go"
)

func main() {
    input := `<http://example.org/s> <http://example.org/p> "v" .`
    
    dec, err := rdf.NewTripleDecoder(strings.NewReader(input), rdf.TripleFormatNTriples)
    if err != nil {
        panic(err)
    }
    defer dec.Close()

    for {
        triple, err := dec.Next()
        if err == io.EOF {
            break
        }
        if err != nil {
            panic(err)
        }
        fmt.Printf("Subject: %s\n", triple.S.String())
        fmt.Printf("Predicate: %s\n", triple.P.String())
        fmt.Printf("Object: %s\n", triple.O.String())
    }
}
```

### Writing RDF Data

Here's how to encode RDF data:

```go
package main

import (
    "bytes"
    "fmt"
    "github.com/geoknoesis/rdf-go"
)

func main() {
    var buf bytes.Buffer
    
    enc, err := rdf.NewTripleEncoder(&buf, rdf.TripleFormatNTriples)
    if err != nil {
        panic(err)
    }
    defer enc.Close()

    triple := rdf.Triple{
        S: rdf.IRI{Value: "http://example.org/s"},
        P: rdf.IRI{Value: "http://example.org/p"},
        O: rdf.Literal{Lexical: "v"},
    }
    
    if err := enc.Write(triple); err != nil {
        panic(err)
    }
    
    if err := enc.Flush(); err != nil {
        panic(err)
    }
    
    fmt.Print(buf.String())
    // Output: <http://example.org/s> <http://example.org/p> "v" .
}
```

## Choosing a Format

The library supports multiple RDF formats, separated into triple-only and quad formats:

**Triple formats** (use with `NewTripleDecoder`/`NewTripleEncoder`):
```go
rdf.TripleFormatTurtle    // Turtle (.ttl)
rdf.TripleFormatNTriples  // N-Triples (.nt)
rdf.TripleFormatRDFXML    // RDF/XML (.rdf, .xml)
rdf.TripleFormatJSONLD    // JSON-LD (.jsonld)
```

**Quad formats** (use with `NewQuadDecoder`/`NewQuadEncoder`):
```go
rdf.QuadFormatTriG   // TriG (.trig)
rdf.QuadFormatNQuads // N-Quads (.nq)
```

You can also parse format strings:

```go
// For triple formats
format, ok := rdf.ParseTripleFormat("ttl")
if !ok {
    // format not recognized
}

// For quad formats
format, ok := rdf.ParseQuadFormat("nq")
if !ok {
    // format not recognized
}
```

## Next Steps

- Learn about [RDF Concepts](concepts.md) in the library
- Explore [How To](how-to.md) guides for common tasks
- Check the [Reference](reference.md) for complete API documentation

