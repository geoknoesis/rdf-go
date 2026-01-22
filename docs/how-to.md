# How To

This guide covers common tasks and patterns when working with `grit/rdf-go`.

## Parse RDF from a File

```go
import (
    "os"
    "github.com/geoknoesis/rdf-go"
)

file, err := os.Open("data.ttl")
if err != nil {
    return err
}
defer file.Close()

dec, err := rdf.NewDecoder(file, rdf.FormatTurtle)
if err != nil {
    return err
}
defer dec.Close()

for {
    quad, err := dec.Next()
    if err == io.EOF {
        break
    }
    if err != nil {
        return err
    }
    // process quad
}
```

## Use the Parse Helper

The `Parse` function provides a convenient way to process RDF with a handler function:

```go
import (
    "context"
    "strings"
    "github.com/geoknoesis/rdf-go"
)

input := `<http://example.org/s> <http://example.org/p> "v" .`

count := 0
err := rdf.Parse(context.Background(), strings.NewReader(input), rdf.FormatNTriples,
    rdf.HandlerFunc(func(q rdf.Quad) error {
        count++
        return nil
    }),
)
```

## Use ParseChan for Concurrent Processing

`ParseChan` returns channels that can be used with goroutines:

```go
import (
    "context"
    "strings"
    "github.com/geoknoesis/rdf-go"
)

input := `<http://example.org/s> <http://example.org/p> "v" .`

out, errs := rdf.ParseChan(context.Background(), strings.NewReader(input), rdf.FormatNTriples)

// Process quads in a goroutine
go func() {
    for q := range out {
        // process quad
    }
}()

// Check for errors
if err, ok := <-errs; ok && err != nil {
    // handle error
}
```

## Convert Between Formats

```go
import (
    "bytes"
    "io"
    "github.com/geoknoesis/rdf-go"
)

// Read from one format
dec, err := rdf.NewDecoder(inputReader, rdf.FormatTurtle)
if err != nil {
    return err
}
defer dec.Close()

// Write to another format
var buf bytes.Buffer
enc, err := rdf.NewEncoder(&buf, rdf.FormatNTriples)
if err != nil {
    return err
}
defer enc.Close()

// Stream conversion
for {
    quad, err := dec.Next()
    if err == io.EOF {
        break
    }
    if err != nil {
        return err
    }
    if err := enc.Write(quad); err != nil {
        return err
    }
}

if err := enc.Flush(); err != nil {
    return err
}
```

## Filter Quads

You can filter quads during parsing:

```go
err := rdf.Parse(context.Background(), reader, rdf.FormatTurtle,
    rdf.HandlerFunc(func(q rdf.Quad) error {
        // Only process quads with a specific predicate
        if q.P.Value == "http://example.org/name" {
            // process quad
        }
        return nil
    }),
)
```

## Handle Errors Gracefully

```go
dec, err := rdf.NewDecoder(reader, rdf.FormatTurtle)
if err != nil {
    if err == rdf.ErrUnsupportedFormat {
        // handle unsupported format
    }
    return err
}
defer dec.Close()

for {
    quad, err := dec.Next()
    if err == io.EOF {
        break
    }
    if err != nil {
        // Log error but continue processing if possible
        log.Printf("Error reading quad: %v", err)
        continue
    }
    // process quad
}
```

## Work with Named Graphs

```go
dec, err := rdf.NewDecoder(reader, rdf.FormatTriG)
if err != nil {
    return err
}
defer dec.Close()

for {
    quad, err := dec.Next()
    if err == io.EOF {
        break
    }
    if err != nil {
        return err
    }
    
    // Check if quad is in a named graph
    if quad.G != nil {
        fmt.Printf("Graph: %s\n", quad.G.String())
    }
    
    // process quad
}
```

## Create RDF-star Quoted Triples

```go
// Create a quoted triple
quoted := rdf.TripleTerm{
    S: rdf.IRI{Value: "http://example.org/alice"},
    P: rdf.IRI{Value: "http://example.org/said"},
    O: rdf.Literal{Lexical: "Hello"},
}

// Use it as a subject
stmt := rdf.Quad{
    S: quoted,
    P: rdf.IRI{Value: "http://example.org/asserted"},
    O: rdf.Literal{Lexical: "true"},
}

// Encode it
enc, _ := rdf.NewEncoder(&buf, rdf.FormatTurtle)
_ = enc.Write(stmt)
_ = enc.Close()
```

## Detect Term Types

```go
func processTerm(term rdf.Term) {
    switch term.Kind() {
    case rdf.TermIRI:
        iri := term.(rdf.IRI)
        fmt.Printf("IRI: %s\n", iri.Value)
    case rdf.TermBlankNode:
        bnode := term.(rdf.BlankNode)
        fmt.Printf("Blank node: %s\n", bnode.ID)
    case rdf.TermLiteral:
        lit := term.(rdf.Literal)
        fmt.Printf("Literal: %s\n", lit.Lexical)
    case rdf.TermTriple:
        triple := term.(rdf.TripleTerm)
        fmt.Printf("Quoted triple: %s\n", triple.String())
    }
}
```

## Parse Format from String

```go
// Parse format from user input or file extension
format, ok := rdf.ParseFormat("ttl")
if !ok {
    // handle unknown format
}

// Common aliases:
// "turtle", "ttl" -> FormatTurtle
// "trig" -> FormatTriG
// "ntriples", "nt" -> FormatNTriples
// "nquads", "nq" -> FormatNQuads
// "rdfxml", "rdf", "xml" -> FormatRDFXML
// "jsonld", "json-ld", "json" -> FormatJSONLD
```

## Batch Processing

For efficient batch processing, use buffering:

```go
const batchSize = 1000
batch := make([]rdf.Quad, 0, batchSize)

err := rdf.Parse(context.Background(), reader, rdf.FormatTurtle,
    rdf.HandlerFunc(func(q rdf.Quad) error {
        batch = append(batch, q)
        if len(batch) >= batchSize {
            // process batch
            processBatch(batch)
            batch = batch[:0] // reset slice
        }
        return nil
    }),
)

// Process remaining quads
if len(batch) > 0 {
    processBatch(batch)
}
```

