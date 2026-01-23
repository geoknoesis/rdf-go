# How To

This guide covers common tasks and patterns when working with `rdf-go`.

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

dec, err := rdf.NewTripleDecoder(file, rdf.TripleFormatTurtle)
if err != nil {
    return err
}
defer dec.Close()

for {
    triple, err := dec.Next()
    if err == io.EOF {
        break
    }
    if err != nil {
        return err
    }
    // process triple
}
```

## Use the ParseTriples Helper

The `ParseTriples` function provides a convenient way to process RDF triples with a handler function:

```go
import (
    "context"
    "strings"
    "github.com/geoknoesis/rdf-go"
)

input := `<http://example.org/s> <http://example.org/p> "v" .`

count := 0
err := rdf.ParseTriples(context.Background(), strings.NewReader(input), rdf.TripleFormatNTriples,
    rdf.TripleHandlerFunc(func(t rdf.Triple) error {
        count++
        return nil
    }),
)
```

## Use ParseQuads Helper

The `ParseQuads` function provides a convenient way to process RDF quads with a handler function:

```go
import (
    "context"
    "strings"
    "github.com/geoknoesis/rdf-go"
)

input := `<http://example.org/s> <http://example.org/p> "v" <http://example.org/g> .`

count := 0
err := rdf.ParseQuads(context.Background(), strings.NewReader(input), rdf.QuadFormatNQuads,
    rdf.QuadHandlerFunc(func(q rdf.Quad) error {
        count++
        return nil
    }),
)
```

## Use ParseTriplesChan for Concurrent Processing

`ParseTriplesChan` returns channels that can be used with goroutines:

```go
import (
    "context"
    "strings"
    "github.com/geoknoesis/rdf-go"
)

input := `<http://example.org/s> <http://example.org/p> "v" .`

out, errs := rdf.ParseTriplesChan(context.Background(), strings.NewReader(input), rdf.TripleFormatNTriples)

// Process triples in a goroutine
go func() {
    for t := range out {
        // process triple
    }
}()

// Check for errors
if err, ok := <-errs; ok && err != nil {
    // handle error
}
```

## Use ParseQuadsChan for Concurrent Processing

`ParseQuadsChan` returns channels that can be used with goroutines:

```go
import (
    "context"
    "strings"
    "github.com/geoknoesis/rdf-go"
)

input := `<http://example.org/s> <http://example.org/p> "v" <http://example.org/g> .`

out, errs := rdf.ParseQuadsChan(context.Background(), strings.NewReader(input), rdf.QuadFormatNQuads)

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

// Read from one format (triple format)
dec, err := rdf.NewTripleDecoder(inputReader, rdf.TripleFormatTurtle)
if err != nil {
    return err
}
defer dec.Close()

// Write to another format (triple format)
var buf bytes.Buffer
enc, err := rdf.NewTripleEncoder(&buf, rdf.TripleFormatNTriples)
if err != nil {
    return err
}
defer enc.Close()

// Stream conversion
for {
    triple, err := dec.Next()
    if err == io.EOF {
        break
    }
    if err != nil {
        return err
    }
    if err := enc.Write(triple); err != nil {
        return err
    }
}

if err := enc.Flush(); err != nil {
    return err
}
```

## Filter Triples

You can filter triples during parsing:

```go
err := rdf.ParseTriples(context.Background(), reader, rdf.TripleFormatTurtle,
    rdf.TripleHandlerFunc(func(t rdf.Triple) error {
        // Only process triples with a specific predicate
        if t.P.Value == "http://example.org/name" {
            // process triple
        }
        return nil
    }),
)
```

## Filter Quads

You can filter quads during parsing:

```go
err := rdf.ParseQuads(context.Background(), reader, rdf.QuadFormatTriG,
    rdf.QuadHandlerFunc(func(q rdf.Quad) error {
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
dec, err := rdf.NewTripleDecoder(reader, rdf.TripleFormatTurtle)
if err != nil {
    if err == rdf.ErrUnsupportedFormat {
        // handle unsupported format
    }
    return err
}
defer dec.Close()

for {
    triple, err := dec.Next()
    if err == io.EOF {
        break
    }
    if err != nil {
        // Log error but continue processing if possible
        log.Printf("Error reading triple: %v", err)
        continue
    }
    // process triple
}
```

## Work with Named Graphs

```go
dec, err := rdf.NewQuadDecoder(reader, rdf.QuadFormatTriG)
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
stmt := rdf.Triple{
    S: quoted,
    P: rdf.IRI{Value: "http://example.org/asserted"},
    O: rdf.Literal{Lexical: "true"},
}

// Encode it
enc, _ := rdf.NewTripleEncoder(&buf, rdf.TripleFormatTurtle)
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
// Parse triple format from user input or file extension
format, ok := rdf.ParseTripleFormat("ttl")
if !ok {
    // handle unknown format
}

// Triple format aliases:
// "turtle", "ttl" -> TripleFormatTurtle
// "ntriples", "nt" -> TripleFormatNTriples
// "rdfxml", "rdf", "xml" -> TripleFormatRDFXML
// "jsonld", "json-ld", "json" -> TripleFormatJSONLD

// Parse quad format from user input or file extension
format, ok := rdf.ParseQuadFormat("nq")
if !ok {
    // handle unknown format
}

// Quad format aliases:
// "trig" -> QuadFormatTriG
// "nquads", "nq" -> QuadFormatNQuads
```

## Batch Processing

For efficient batch processing, use buffering:

```go
const batchSize = 1000
batch := make([]rdf.Triple, 0, batchSize)

err := rdf.ParseTriples(context.Background(), reader, rdf.TripleFormatTurtle,
    rdf.TripleHandlerFunc(func(t rdf.Triple) error {
        batch = append(batch, t)
        if len(batch) >= batchSize {
            // process batch
            processBatch(batch)
            batch = batch[:0] // reset slice
        }
        return nil
    }),
)

// Process remaining triples
if len(batch) > 0 {
    processBatch(batch)
}
```

For quads:

```go
const batchSize = 1000
batch := make([]rdf.Quad, 0, batchSize)

err := rdf.ParseQuads(context.Background(), reader, rdf.QuadFormatTriG,
    rdf.QuadHandlerFunc(func(q rdf.Quad) error {
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

