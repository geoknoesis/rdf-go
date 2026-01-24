# API Reference

Complete reference documentation for `rdf-go`.

## Types

### Term Interface

```go
type Term interface {
    Kind() TermKind
    String() string
}
```

`Term` is the interface that all RDF terms implement.

### TermKind

```go
type TermKind uint8

const (
    TermIRI TermKind = iota
    TermBlankNode
    TermLiteral
    TermTriple
)
```

`TermKind` identifies the type of an RDF term.

### IRI

```go
type IRI struct {
    Value string
}
```

`IRI` represents an RDF IRI (Internationalized Resource Identifier).

**Methods:**
- `Kind() TermKind` - Returns `TermIRI`
- `String() string` - Returns the IRI value

### BlankNode

```go
type BlankNode struct {
    ID string
}
```

`BlankNode` represents an RDF blank node.

**Methods:**
- `Kind() TermKind` - Returns `TermBlankNode`
- `String() string` - Returns `"_:" + ID`

### Literal

```go
type Literal struct {
    Lexical  string
    Datatype IRI
    Lang     string
}
```

`Literal` represents an RDF literal value.

**Fields:**
- `Lexical` - The lexical form of the literal
- `Datatype` - Optional datatype IRI
- `Lang` - Optional language tag

**Methods:**
- `Kind() TermKind` - Returns `TermLiteral`
- `String() string` - Returns formatted literal with datatype or language tag

### TripleTerm

```go
type TripleTerm struct {
    S Term
    P IRI
    O Term
}
```

`TripleTerm` represents an RDF-star quoted triple.

**Methods:**
- `Kind() TermKind` - Returns `TermTriple`
- `String() string` - Returns formatted quoted triple

### Triple

```go
type Triple struct {
    S Term
    P IRI
    O Term
}
```

`Triple` represents an RDF triple (subject, predicate, object).

**Methods:**
- `ToStatement() Statement` - Converts triple to a statement
- `ToQuad() Quad` - Converts triple to a quad in the default graph
- `ToQuadInGraph(graph Term) Quad` - Converts triple to a quad in a named graph

### Quad

```go
type Quad struct {
    S Term
    P IRI
    O Term
    G Term
}
```

`Quad` represents an RDF quad (triple with optional graph name).

**Methods:**
- `IsZero() bool` - Reports whether the quad has no subject/predicate/object
- `ToTriple() Triple` - Extracts the triple from a quad (ignores graph)
- `InDefaultGraph() bool` - Reports whether the quad is in the default graph
- `ToStatement() Statement` - Converts quad to a statement

### Statement

```go
type Statement struct {
    S Term
    P IRI
    O Term
    G Term  // nil for triples, non-nil for quads (can be omitted, defaults to nil)
}
```

`Statement` represents an RDF statement, which can be either a triple or a quad.
If `G` is `nil` (or omitted), it represents a triple. If `G` is non-nil, it represents a quad.

**Creating Statements:**

```go
// Option 1: Omit G (defaults to nil for triples)
stmt := rdf.Statement{
    S: rdf.IRI{Value: "http://example.org/s"},
    P: rdf.IRI{Value: "http://example.org/p"},
    O: rdf.IRI{Value: "http://example.org/o"},
    // G omitted - defaults to nil (triple)
}

// Option 2: Use convenience function
stmt := rdf.NewTriple(
    rdf.IRI{Value: "http://example.org/s"},
    rdf.IRI{Value: "http://example.org/p"},
    rdf.IRI{Value: "http://example.org/o"},
)

// For quads, specify G
quad := rdf.Statement{
    S: rdf.IRI{Value: "http://example.org/s"},
    P: rdf.IRI{Value: "http://example.org/p"},
    O: rdf.IRI{Value: "http://example.org/o"},
    G: rdf.IRI{Value: "http://example.org/g"},
}

// Or use convenience function
quad := rdf.NewQuad(
    rdf.IRI{Value: "http://example.org/s"},
    rdf.IRI{Value: "http://example.org/p"},
    rdf.IRI{Value: "http://example.org/o"},
    rdf.IRI{Value: "http://example.org/g"},
)
```

**Methods:**
- `IsTriple() bool` - Reports whether the statement is a triple (G is nil)
- `IsQuad() bool` - Reports whether the statement is a quad (G is non-nil)
- `AsTriple() Triple` - Returns the statement as a triple (ignores graph)
- `AsQuad() Quad` - Returns the statement as a quad

### Format

```go
type Format string

const (
    FormatAuto Format = ""
    
    // Triple formats
    FormatTurtle   Format = "turtle"
    FormatNTriples Format = "ntriples"
    FormatRDFXML   Format = "rdfxml"
    FormatJSONLD   Format = "jsonld"
    
    // Quad formats
    FormatTriG   Format = "trig"
    FormatNQuads Format = "nquads"
)
```

`Format` represents an RDF serialization format.

**Methods:**
- `ParseFormat(s string) (Format, bool)` - Parses a format string
- `IsQuadFormat() bool` - Reports whether the format supports quads
- `String() string` - Returns the canonical format name

## Interfaces

### Reader

```go
type Reader interface {
    Next() (Statement, error)
    Close() error
}
```

`Reader` streams RDF statements from an input. A statement can be either a triple (G is nil) or a quad (G is non-nil).

**Methods:**
- `Next() (Statement, error)` - Returns the next statement, or `io.EOF` when done
- `Close() error` - Closes the reader and releases resources

### Writer

```go
type Writer interface {
    Write(Statement) error
    Flush() error
    Close() error
}
```

`Writer` streams RDF statements to an output. For triple-only formats, the graph (G) field is ignored.

**Methods:**
- `Write(Statement) error` - Writes a statement to the output
- `Flush() error` - Flushes any buffered data
- `Close() error` - Closes the writer and releases resources

### Handler

```go
type Handler func(Statement) error
```

`Handler` processes statements in push mode. It's a function type that can be passed to `Parse`.

## Functions

### NewReader

```go
func NewReader(r io.Reader, format Format, opts ...Option) (Reader, error)
```

`NewReader` creates a reader for the specified format. If format is `FormatAuto` (empty string), the format is automatically detected. Auto-detection reads from the reader, so the reader position will be advanced.

**Parameters:**
- `r` - Input reader
- `format` - Format to read (use `FormatAuto` for auto-detection)
- `opts` - Optional configuration options

**Returns:**
- `Reader` - The reader instance
- `error` - Error if format is unsupported or initialization fails

**Example:**
```go
reader, err := rdf.NewReader(reader, rdf.FormatTurtle)
```

### NewWriter

```go
func NewWriter(w io.Writer, format Format, opts ...Option) (Writer, error)
```

`NewWriter` creates a writer for the specified format.

**Parameters:**
- `w` - Output writer
- `format` - Format to write
- `opts` - Optional configuration options

**Returns:**
- `Writer` - The writer instance
- `error` - Error if format is unsupported or initialization fails

**Example:**
```go
writer, err := rdf.NewWriter(writer, rdf.FormatNTriples)
```

### Parse

```go
func Parse(ctx context.Context, r io.Reader, format Format, handler Handler, opts ...Option) error
```

`Parse` parses RDF from the reader and streams statements to the handler. If format is `FormatAuto` (empty string), the format is automatically detected.

**Parameters:**
- `ctx` - Context for cancellation
- `r` - Input reader
- `format` - Format to parse (use `FormatAuto` for auto-detection)
- `handler` - Handler function to process statements
- `opts` - Optional configuration options

**Returns:**
- `error` - Error if parsing fails

**Example:**
```go
err := rdf.Parse(context.Background(), reader, rdf.FormatTurtle, func(s rdf.Statement) error {
    // process statement
    return nil
})
```

### ParseFormat

```go
func ParseFormat(s string) (Format, bool)
```

`ParseFormat` normalizes a format string and returns a `Format`. Supports common aliases (e.g., "ttl" -> `FormatTurtle`, "nt" -> `FormatNTriples`).

**Parameters:**
- `s` - Format string (e.g., "ttl", "turtle", "nt", "trig", "nq")

**Returns:**
- `Format` - The format constant
- `bool` - True if format was recognized

**Supported aliases:**
- Turtle: "turtle", "ttl"
- N-Triples: "ntriples", "nt"
- RDF/XML: "rdfxml", "rdf", "xml"
- JSON-LD: "jsonld", "json-ld", "json"
- TriG: "trig"
- N-Quads: "nquads", "nq"
- Auto: "", "auto"

**Example:**
```go
format, ok := rdf.ParseFormat("ttl")
```

## Options

Options configure reader/writer behavior using functional options.

### Option

```go
type Option func(*Options)
```

`Option` is a function that modifies `Options`.

### Options

```go
type Options struct {
    Context context.Context
    
    // Security limits for untrusted input
    MaxLineBytes      int
    MaxStatementBytes int
    MaxDepth          int
    MaxTriples        int64
    
    // Format-specific options
    AllowQuotedTripleStatement bool
    DebugStatements            bool
}
```

`Options` configures parser/writer behavior.

### Option Functions

- `OptContext(ctx context.Context) Option` - Set context for cancellation and timeouts
- `OptMaxLineBytes(maxBytes int) Option` - Set maximum line size limit
- `OptMaxStatementBytes(maxBytes int) Option` - Set maximum statement size limit
- `OptMaxDepth(maxDepth int) Option` - Set maximum nesting depth limit
- `OptMaxTriples(maxTriples int64) Option` - Set maximum number of triples/quads to process
- `OptSafeLimits() Option` - Apply safe limits suitable for untrusted input

**Example:**
```go
reader, err := rdf.NewReader(reader, rdf.FormatTurtle,
    rdf.OptSafeLimits(),
    rdf.OptMaxDepth(50),
    rdf.OptContext(ctx),
)
```

## Errors

### ErrUnsupportedFormat

```go
var ErrUnsupportedFormat = errors.New("unsupported format")
```

`ErrUnsupportedFormat` is returned when a format is not supported by `NewReader` or `NewWriter`.

### ParseError

```go
type ParseError struct {
    Format  string
    Line    int
    Column  int
    Offset  int
    Err     error
    Message string
}
```

`ParseError` represents a parsing error with position information.

**Methods:**
- `Error() string` - Returns a formatted error message
- `Unwrap() error` - Returns the underlying error

## Best Practices

1. **Always close readers and writers**: Use `defer reader.Close()` or `defer writer.Close()`

2. **Handle EOF properly**: Check for `io.EOF` when `Next()` returns an error

3. **Use streaming for large files**: Use `NewReader` or `Parse` for efficient processing of large inputs

4. **Check format support**: Use `ParseFormat` to validate format strings before creating readers/writers

5. **Use Statement type**: The unified `Statement` type works with all formats. Use `stmt.IsTriple()` or `stmt.IsQuad()` to check the statement type.

6. **Use context for cancellation**: Pass a context to `Parse` to enable cancellation

7. **Flush writers**: Call `Flush()` before `Close()` to ensure all data is written

8. **Set security limits**: For untrusted input, always use `OptSafeLimits()` or set explicit limits
