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

Here's a minimal example that reads N-Triples format. This demonstrates the basic pull-style decoding pattern:

```go
package main

import (
    "fmt"
    "io"
    "strings"
    "github.com/geoknoesis/rdf-go"
)

func main() {
    // Sample RDF data in N-Triples format
    // Format: <subject> <predicate> <object> .
    input := `<http://example.org/s> <http://example.org/p> "v" .`
    
    // Create a reader for N-Triples format
    // The reader reads from a strings.Reader, but you can use any io.Reader
    dec, err := rdf.NewReader(strings.NewReader(input), rdf.FormatNTriples)
    if err != nil {
        // Handle initialization errors (unsupported format, etc.)
        panic(err)
    }
    // Always close the reader when done to release resources
    defer dec.Close()

    // Pull-style decoding: repeatedly call Next() to get statements
    for {
        stmt, err := dec.Next()
        if err == io.EOF {
            // End of input reached - normal termination
            break
        }
        if err != nil {
            // Handle parsing errors
            panic(err)
        }
        
        // Process the statement
        // Each statement has S (subject), P (predicate), O (object), and G (graph)
        fmt.Printf("Subject: %s\n", stmt.S.String())
        fmt.Printf("Predicate: %s\n", stmt.P.String())
        fmt.Printf("Object: %s\n", stmt.O.String())
        
        // Check if this is a quad (has a graph name)
        // For N-Triples, this will always be false (it's a triple-only format)
        if stmt.IsQuad() {
            fmt.Printf("Graph: %s\n", stmt.G.String())
        }
    }
}
```

### Writing RDF Data

Here's how to encode RDF data. This demonstrates the push-style encoding pattern:

```go
package main

import (
    "bytes"
    "fmt"
    "github.com/geoknoesis/rdf-go"
)

func main() {
    // Create a buffer to hold the encoded output
    // In real applications, you might write to a file or network connection
    var buf bytes.Buffer
    
    // Create an writer for N-Triples format
    enc, err := rdf.NewWriter(&buf, rdf.FormatNTriples)
    if err != nil {
        // Handle initialization errors
        panic(err)
    }
    // Always close the writer to flush any remaining data
    defer enc.Close()

    // Create a statement (triple) - there are two ways:
    
    // Option 1: Omit G field (defaults to nil for triples) - more readable!
    stmt := rdf.Statement{
        S: rdf.IRI{Value: "http://example.org/s"},  // Subject: identifies the resource
        P: rdf.IRI{Value: "http://example.org/p"},  // Predicate: the property/relationship
        O: rdf.Literal{Lexical: "v"},                // Object: the value (a string literal)
        // G is omitted - defaults to nil (this is a triple, not a quad)
    }
    
    // Option 2: Use the convenience function (cleaner for simple cases)
    stmt := rdf.NewTriple(
        rdf.IRI{Value: "http://example.org/s"},
        rdf.IRI{Value: "http://example.org/p"},
        rdf.Literal{Lexical: "v"},
    )
    
    // Write the statement to the writer
    if err := enc.Write(stmt); err != nil {
        // Handle write errors
        panic(err)
    }
    
    // Flush any buffered data (important for some formats)
    if err := enc.Flush(); err != nil {
        panic(err)
    }
    
    // The encoded RDF is now in the buffer
    fmt.Print(buf.String())
    // Output: <http://example.org/s> <http://example.org/p> "v" .
}
```

### Using Auto-Detection

When you don't know the format in advance, use `FormatAuto` to let the library detect it automatically. The library examines the input content to determine the format:

```go
import (
    "io"
    "github.com/geoknoesis/rdf-go"
)

// FormatAuto tells the reader to detect the format from the input
// The library reads a sample of the input and uses heuristics to identify the format
dec, err := rdf.NewReader(reader, rdf.FormatAuto)
if err != nil {
    // Handle errors (format detection failure, unsupported format, etc.)
    return err
}
defer dec.Close()

// The format is automatically detected from the input
// You can now read statements without knowing the format
for {
    stmt, err := dec.Next()
    if err == io.EOF {
        // End of input
        break
    }
    if err != nil {
        // Handle parsing errors
        return err
    }
    
    // Process statement - works the same regardless of detected format
    // The statement type (triple vs quad) depends on the format:
    // - Triple formats (Turtle, N-Triples, RDF/XML, JSON-LD): stmt.G will be nil
    // - Quad formats (TriG, N-Quads): stmt.G may be non-nil
    processStatement(stmt)
}
```

## Choosing a Format

The library uses a unified `Format` type for all formats:

```go
// Triple formats
rdf.FormatTurtle    // Turtle (.ttl)
rdf.FormatNTriples  // N-Triples (.nt)
rdf.FormatRDFXML    // RDF/XML (.rdf, .xml)
rdf.FormatJSONLD    // JSON-LD (.jsonld)

// Quad formats
rdf.FormatTriG   // TriG (.trig)
rdf.FormatNQuads // N-Quads (.nq)

// Auto-detection
rdf.FormatAuto   // Automatically detect format
```

You can parse format strings:

```go
format, ok := rdf.ParseFormat("ttl")
if !ok {
    // format not recognized
}

// Supported aliases:
// Turtle: "turtle", "ttl"
// N-Triples: "ntriples", "nt"
// RDF/XML: "rdfxml", "rdf", "xml"
// JSON-LD: "jsonld", "json-ld", "json"
// TriG: "trig"
// N-Quads: "nquads", "nq"
// Auto: "", "auto"
```

## Working with Statements

The `Statement` type is the unified type that represents either a triple or a quad. This makes the API simpler - you don't need separate types for triples and quads:

```go
// Creating a triple statement
// Option 1: Omit G (defaults to nil) - recommended for readability
triple := rdf.Statement{
    S: rdf.IRI{Value: "http://example.org/s"},  // Subject
    P: rdf.IRI{Value: "http://example.org/p"},  // Predicate
    O: rdf.Literal{Lexical: "v"},                // Object
    // G omitted - defaults to nil (this is a triple)
}

// Option 2: Use convenience function
triple := rdf.NewTriple(
    rdf.IRI{Value: "http://example.org/s"},
    rdf.IRI{Value: "http://example.org/p"},
    rdf.Literal{Lexical: "v"},
)

// Creating a quad statement (has a graph name)
// For quads, you must specify G (it cannot be nil)
quad := rdf.Statement{
    S: rdf.IRI{Value: "http://example.org/s"},
    P: rdf.IRI{Value: "http://example.org/p"},
    O: rdf.Literal{Lexical: "v"},
    G: rdf.IRI{Value: "http://example.org/g"}, // Graph name (non-nil = quad)
}

// Or use convenience function
quad := rdf.NewQuad(
    rdf.IRI{Value: "http://example.org/s"},
    rdf.IRI{Value: "http://example.org/p"},
    rdf.Literal{Lexical: "v"},
    rdf.IRI{Value: "http://example.org/g"},
)

// Check statement type at runtime
if stmt.IsTriple() {
    fmt.Println("This is a triple (G is nil)")
}
if stmt.IsQuad() {
    fmt.Println("This is a quad (G is non-nil)")
    fmt.Printf("Graph name: %s\n", stmt.G.String())
}

// Convert to specific types if needed
triple := stmt.AsTriple()  // Extracts S, P, O (ignores G)
quad := stmt.AsQuad()      // Extracts S, P, O, G
```

## Next Steps

- Learn about [RDF Concepts](concepts.md) in the library
- Explore [How To](how-to.md) guides for common tasks
- Check the [Reference](reference.md) for complete API documentation
