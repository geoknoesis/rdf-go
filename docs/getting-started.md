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
    
    dec, err := rdf.NewDecoder(strings.NewReader(input), rdf.FormatNTriples)
    if err != nil {
        panic(err)
    }
    defer dec.Close()

    for {
        quad, err := dec.Next()
        if err == io.EOF {
            break
        }
        if err != nil {
            panic(err)
        }
        fmt.Printf("Subject: %s\n", quad.S.String())
        fmt.Printf("Predicate: %s\n", quad.P.String())
        fmt.Printf("Object: %s\n", quad.O.String())
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
    
    enc, err := rdf.NewEncoder(&buf, rdf.FormatNTriples)
    if err != nil {
        panic(err)
    }
    defer enc.Close()

    quad := rdf.Quad{
        S: rdf.IRI{Value: "http://example.org/s"},
        P: rdf.IRI{Value: "http://example.org/p"},
        O: rdf.Literal{Lexical: "v"},
    }
    
    if err := enc.Write(quad); err != nil {
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

The library supports multiple RDF formats. You can specify the format when creating a decoder or encoder:

```go
// Supported formats
rdf.FormatTurtle    // Turtle (.ttl)
rdf.FormatTriG      // TriG (.trig)
rdf.FormatNTriples  // N-Triples (.nt)
rdf.FormatNQuads     // N-Quads (.nq)
rdf.FormatRDFXML     // RDF/XML (.rdf, .xml)
rdf.FormatJSONLD     // JSON-LD (.jsonld)
```

You can also parse format strings:

```go
format, ok := rdf.ParseFormat("ttl")
if !ok {
    // format not recognized
}
```

## Next Steps

- Learn about [RDF Concepts](concepts.md) in the library
- Explore [How To](how-to.md) guides for common tasks
- Check the [Reference](reference.md) for complete API documentation

