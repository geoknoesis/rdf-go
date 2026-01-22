# API Reference

Complete reference documentation for `grit/rdf-go`.

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

### Format

```go
type Format string

const (
    FormatTurtle   Format = "turtle"
    FormatTriG     Format = "trig"
    FormatNTriples Format = "ntriples"
    FormatNQuads   Format = "nquads"
    FormatRDFXML   Format = "rdfxml"
    FormatJSONLD   Format = "jsonld"
)
```

`Format` identifies RDF serialization formats.

## Interfaces

### Decoder

```go
type Decoder interface {
    Next() (Quad, error)
    Err() error
    Close() error
}
```

`Decoder` streams RDF quads from an input.

**Methods:**
- `Next() (Quad, error)` - Returns the next quad, or `io.EOF` when done
- `Err() error` - Returns any error that occurred during decoding
- `Close() error` - Closes the decoder and releases resources

### Encoder

```go
type Encoder interface {
    Write(Quad) error
    Flush() error
    Close() error
}
```

`Encoder` streams RDF quads to an output.

**Methods:**
- `Write(Quad) error` - Writes a quad to the output
- `Flush() error` - Flushes any buffered data
- `Close() error` - Closes the encoder and releases resources

### Handler

```go
type Handler interface {
    Handle(Quad) error
}
```

`Handler` processes quads in push mode.

### HandlerFunc

```go
type HandlerFunc func(Quad) error

func (h HandlerFunc) Handle(q Quad) error
```

`HandlerFunc` adapts a function to a `Handler`.

## Functions

### NewDecoder

```go
func NewDecoder(r io.Reader, format Format) (Decoder, error)
```

`NewDecoder` creates a decoder for the given format.

**Parameters:**
- `r` - Input reader
- `format` - RDF format to decode

**Returns:**
- `Decoder` - The decoder instance
- `error` - Error if format is unsupported or initialization fails

**Example:**
```go
dec, err := rdf.NewDecoder(reader, rdf.FormatTurtle)
```

### NewEncoder

```go
func NewEncoder(w io.Writer, format Format) (Encoder, error)
```

`NewEncoder` creates an encoder for the given format.

**Parameters:**
- `w` - Output writer
- `format` - RDF format to encode

**Returns:**
- `Encoder` - The encoder instance
- `error` - Error if format is unsupported or initialization fails

**Example:**
```go
enc, err := rdf.NewEncoder(writer, rdf.FormatNTriples)
```

### Parse

```go
func Parse(ctx context.Context, r io.Reader, format Format, handler Handler) error
```

`Parse` streams RDF data to a handler function.

**Parameters:**
- `ctx` - Context for cancellation
- `r` - Input reader
- `format` - RDF format
- `handler` - Handler to process quads

**Returns:**
- `error` - Error if parsing fails

**Example:**
```go
err := rdf.Parse(ctx, reader, rdf.FormatTurtle,
    rdf.HandlerFunc(func(q rdf.Quad) error {
        // process quad
        return nil
    }),
)
```

### ParseChan

```go
func ParseChan(ctx context.Context, r io.Reader, format Format) (<-chan Quad, <-chan error)
```

`ParseChan` returns a channel of quads and an error channel.

**Parameters:**
- `ctx` - Context for cancellation
- `r` - Input reader
- `format` - RDF format

**Returns:**
- `<-chan Quad` - Channel of quads
- `<-chan error` - Channel for errors

**Example:**
```go
out, errs := rdf.ParseChan(ctx, reader, rdf.FormatTurtle)
for q := range out {
    // process quad
}
if err, ok := <-errs; ok && err != nil {
    // handle error
}
```

### ParseFormat

```go
func ParseFormat(value string) (Format, bool)
```

`ParseFormat` normalizes a format string.

**Parameters:**
- `value` - Format string (e.g., "ttl", "turtle", "nt")

**Returns:**
- `Format` - The format constant
- `bool` - True if format was recognized

**Supported aliases:**
- Turtle: "turtle", "ttl"
- TriG: "trig"
- N-Triples: "ntriples", "nt"
- N-Quads: "nquads", "nq"
- RDF/XML: "rdfxml", "rdf", "xml"
- JSON-LD: "jsonld", "json-ld", "json"

**Example:**
```go
format, ok := rdf.ParseFormat("ttl")
```

## Errors

### ErrUnsupportedFormat

```go
var ErrUnsupportedFormat = errors.New("unsupported format")
```

`ErrUnsupportedFormat` is returned when a format is not supported by `NewDecoder` or `NewEncoder`.

## Best Practices

1. **Always close decoders and encoders**: Use `defer dec.Close()` or `defer enc.Close()`

2. **Handle EOF properly**: Check for `io.EOF` when `Next()` returns an error

3. **Use streaming for large files**: Prefer `NewDecoder` or `Parse` over loading all data into memory

4. **Check format support**: Use `ParseFormat` to validate format strings before creating decoders/encoders

5. **Use context for cancellation**: Pass a context to `Parse` and `ParseChan` to enable cancellation

6. **Flush encoders**: Call `Flush()` before `Close()` to ensure all data is written

