# How To

This guide covers common tasks and patterns when working with `rdf-go`.

## Parse RDF from a File

Reading RDF from a file follows the same pattern as reading from any `io.Reader`. Always handle errors and close resources properly:

```go
import (
    "io"
    "os"
    "github.com/geoknoesis/rdf-go"
)

// Open the file
file, err := os.Open("data.ttl")
if err != nil {
    return err
}
// Always close the file when done
defer file.Close()

// Create a reader for Turtle format
// The reader reads from the file
dec, err := rdf.NewReader(file, rdf.FormatTurtle)
if err != nil {
    // Handle initialization errors (unsupported format, etc.)
    return err
}
// Always close the reader to release resources
defer dec.Close()

// Read statements one by one
for {
    stmt, err := dec.Next()
    if err == io.EOF {
        // End of file reached - normal termination
        break
    }
    if err != nil {
        // Handle parsing errors
        return err
    }
    
    // Process the statement
    // Each statement has S (subject), P (predicate), O (object), and G (graph)
    processStatement(stmt)
}
```

## Use the Parse Helper

The `Parse` function provides a convenient way to process RDF statements with a handler function. It handles the reader lifecycle for you:

```go
import (
    "context"
    "strings"
    "github.com/geoknoesis/rdf-go"
)

// Sample RDF input
input := `<http://example.org/s> <http://example.org/p> "v" .`

// Count statements as they are parsed
count := 0

// Parse with a handler function
// The handler is called for each statement found in the input
err := rdf.Parse(context.Background(), strings.NewReader(input), rdf.FormatNTriples,
    func(s rdf.Statement) error {
        // This function is called for each statement
        count++
        
        // You can process the statement here
        fmt.Printf("Statement %d: %s %s %s\n", 
            count, s.S.String(), s.P.String(), s.O.String())
        
        // Return nil to continue, or an error to stop parsing
        return nil
    },
)

if err != nil {
    // Handle errors (parse errors, context cancellation, etc.)
    return err
}

fmt.Printf("Parsed %d statements\n", count)
```

## Use ReadAll for Small Datasets

For small datasets that fit in memory, `ReadAll` loads all statements at once. **Warning:** This loads everything into memory, so use `Parse` or `NewReader` for large files:

```go
import (
    "context"
    "github.com/geoknoesis/rdf-go"
)

// ReadAll loads all statements from the reader into a slice
// FormatAuto enables automatic format detection
stmts, err := rdf.ReadAll(context.Background(), reader, rdf.FormatAuto)
if err != nil {
    // Handle errors (format detection failure, parse errors, etc.)
    return err
}

// Now you have all statements in memory
fmt.Printf("Loaded %d statements\n", len(stmts))

// Process them
for i, stmt := range stmts {
    fmt.Printf("Statement %d: %s %s %s\n", 
        i+1, stmt.S.String(), stmt.P.String(), stmt.O.String())
    
    // Check if it's a quad
    if stmt.IsQuad() {
        fmt.Printf("  Graph: %s\n", stmt.G.String())
    }
}
```

## Convert Between Formats

Converting between formats is straightforward - read from one format and write to another. The unified `Statement` type works with all formats:

```go
import (
    "bytes"
    "io"
    "github.com/geoknoesis/rdf-go"
)

// Step 1: Create a reader for the input format (Turtle)
dec, err := rdf.NewReader(inputReader, rdf.FormatTurtle)
if err != nil {
    return err
}
defer dec.Close()

// Step 2: Create an writer for the output format (N-Triples)
var buf bytes.Buffer
enc, err := rdf.NewWriter(&buf, rdf.FormatNTriples)
if err != nil {
    return err
}
defer enc.Close()

// Step 3: Stream conversion - read from reader, write to writer
// This is memory-efficient as it processes one statement at a time
for {
    stmt, err := dec.Next()
    if err == io.EOF {
        // End of input
        break
    }
    if err != nil {
        // Handle read errors
        return err
    }
    
    // Write the statement to the writer
    // The writer automatically handles format conversion
    if err := enc.Write(stmt); err != nil {
        // Handle write errors
        return err
    }
}

// Step 4: Flush any buffered data
if err := enc.Flush(); err != nil {
    return err
}

// The converted RDF is now in buf
fmt.Print(buf.String())
```

## Filter Statements

You can filter statements during parsing by checking their properties in the handler function:

```go
import (
    "context"
    "github.com/geoknoesis/rdf-go"
)

// Filter statements by predicate
err := rdf.Parse(context.Background(), reader, rdf.FormatTurtle,
    func(s rdf.Statement) error {
        // Only process statements with a specific predicate
        // s.P is the predicate (IRI)
        if s.P.Value == "http://example.org/name" {
            // This statement has the "name" predicate
            fmt.Printf("Name: %s\n", s.O.String())
        }
        
        // You can filter by other properties too:
        // - s.S: subject
        // - s.O: object
        // - s.G: graph (for quads)
        // - s.IsTriple() or s.IsQuad(): statement type
        
        // Return nil to continue, or an error to stop
        return nil
    },
)

if err != nil {
    return err
}
```

## Work with Named Graphs

Named graphs allow you to group statements together. Quad formats (TriG, N-Quads) support named graphs:

```go
import (
    "io"
    "github.com/geoknoesis/rdf-go"
)

// Use a quad format to read named graphs
// TriG is like Turtle but supports named graphs
dec, err := rdf.NewReader(reader, rdf.FormatTriG)
if err != nil {
    return err
}
defer dec.Close()

// Group statements by graph
graphs := make(map[string][]rdf.Statement)

for {
    stmt, err := dec.Next()
    if err == io.EOF {
        break
    }
    if err != nil {
        return err
    }
    
    // Check if statement is a quad (has a graph name)
    if stmt.IsQuad() {
        // This statement belongs to a named graph
        graphName := stmt.G.String()
        fmt.Printf("Statement in graph: %s\n", graphName)
        
        // Group statements by graph
        graphs[graphName] = append(graphs[graphName], stmt)
    } else {
        // This statement is in the default graph (no graph name)
        fmt.Println("Statement in default graph")
        graphs[""] = append(graphs[""], stmt)
    }
}

// Now you can process each graph separately
for graphName, stmts := range graphs {
    fmt.Printf("Graph '%s' has %d statements\n", graphName, len(stmts))
}
```

## Create RDF-star Quoted Triples

RDF-star allows you to make statements about statements. This is useful for provenance, trust, or metadata about statements:

```go
import (
    "bytes"
    "github.com/geoknoesis/rdf-go"
)

// Step 1: Create a quoted triple
// This represents a statement that can be used as a subject or object
quoted := rdf.TripleTerm{
    S: rdf.IRI{Value: "http://example.org/alice"},  // Subject of the quoted triple
    P: rdf.IRI{Value: "http://example.org/said"},  // Predicate of the quoted triple
    O: rdf.Literal{Lexical: "Hello"},              // Object of the quoted triple
}

// Step 2: Use the quoted triple as a subject in a new statement
// This says: "The statement 'Alice said Hello' is asserted to be true"
stmt := rdf.Statement{
    S: quoted,  // Subject is the quoted triple (RDF-star feature)
    P: rdf.IRI{Value: "http://example.org/asserted"}, // Predicate: "is asserted"
    O: rdf.Literal{Lexical: "true"},                 // Object: true
    // G omitted - defaults to nil (this is a triple)
}

// Step 3: Encode it to a format that supports RDF-star (Turtle, TriG)
var buf bytes.Buffer
enc, err := rdf.NewWriter(&buf, rdf.FormatTurtle)
if err != nil {
    return err
}
defer enc.Close()

if err := enc.Write(stmt); err != nil {
    return err
}

// The encoded output shows the quoted triple
fmt.Print(buf.String())
// Output: <<http://example.org/alice http://example.org/said "Hello">> 
//          <http://example.org/asserted> "true" .
```

## Detect Term Types

All RDF terms implement the `Term` interface. Use `Kind()` to determine the type and then use type assertions to access specific fields:

```go
import "github.com/geoknoesis/rdf-go"

func processTerm(term rdf.Term) {
    // Use Kind() to determine the term type
    switch term.Kind() {
    case rdf.TermIRI:
        // Type assert to IRI to access the Value field
        iri := term.(rdf.IRI)
        fmt.Printf("IRI: %s\n", iri.Value)
        
    case rdf.TermBlankNode:
        // Type assert to BlankNode to access the ID field
        bnode := term.(rdf.BlankNode)
        fmt.Printf("Blank node: %s\n", bnode.ID)
        
    case rdf.TermLiteral:
        // Type assert to Literal to access Lexical, Datatype, Lang fields
        lit := term.(rdf.Literal)
        fmt.Printf("Literal: %s", lit.Lexical)
        if lit.Datatype.Value != "" {
            fmt.Printf(" (datatype: %s)", lit.Datatype.Value)
        }
        if lit.Lang != "" {
            fmt.Printf(" (lang: %s)", lit.Lang)
        }
        fmt.Println()
        
    case rdf.TermTriple:
        // Type assert to TripleTerm (RDF-star quoted triple)
        triple := term.(rdf.TripleTerm)
        fmt.Printf("Quoted triple: %s\n", triple.String())
    }
}

// Example usage:
func processStatement(stmt rdf.Statement) {
    fmt.Println("Subject:")
    processTerm(stmt.S)
    
    fmt.Println("Object:")
    processTerm(stmt.O)
}
```

## Parse Format from String

```go
// Parse format from user input or file extension
format, ok := rdf.ParseFormat("ttl")
if !ok {
    // handle unknown format
}

// Supported aliases:
// "turtle", "ttl" -> FormatTurtle
// "ntriples", "nt" -> FormatNTriples
// "rdfxml", "rdf", "xml" -> FormatRDFXML
// "jsonld", "json-ld", "json" -> FormatJSONLD
// "trig" -> FormatTriG
// "nquads", "nq" -> FormatNQuads
// "", "auto" -> FormatAuto
```

## Batch Processing

For efficient batch processing, use buffering:

```go
const batchSize = 1000
batch := make([]rdf.Statement, 0, batchSize)

err := rdf.Parse(context.Background(), reader, rdf.FormatTurtle,
    func(s rdf.Statement) error {
        batch = append(batch, s)
        if len(batch) >= batchSize {
            // process batch
            processBatch(batch)
            batch = batch[:0] // reset slice
        }
        return nil
    },
)

// Process remaining statements
if len(batch) > 0 {
    processBatch(batch)
}
```

## Use Auto-Detection

The library can automatically detect the format from input:

```go
// Auto-detect format
dec, err := rdf.NewReader(reader, rdf.FormatAuto)
if err != nil {
    return err
}
defer dec.Close()

// Or use Parse with auto-detection
err := rdf.Parse(context.Background(), reader, rdf.FormatAuto,
    func(s rdf.Statement) error {
        // process statement
        return nil
    },
)
```

## Set Security Limits

For untrusted input, always set security limits:

```go
dec, err := rdf.NewReader(reader, rdf.FormatTurtle,
    rdf.OptSafeLimits(),        // Apply safe limits
    rdf.OptMaxDepth(50),        // Set maximum nesting depth
    rdf.OptContext(ctx),        // Set context for cancellation
    rdf.OptMaxLineBytes(64<<10), // Set maximum line size
)
```

## Convert Between Statement, Triple, and Quad

```go
// Triple to Statement
triple := rdf.Triple{...}
stmt := triple.ToStatement()

// Quad to Statement
quad := rdf.Quad{...}
stmt := quad.ToStatement()

// Statement to Triple
triple := stmt.AsTriple()

// Statement to Quad
quad := stmt.AsQuad()

// Check statement type
if stmt.IsTriple() {
    // Handle triple
}
if stmt.IsQuad() {
    // Handle quad
}
```
