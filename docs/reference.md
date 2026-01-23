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

### TripleFormat

```go
type TripleFormat string

const (
    TripleFormatTurtle   TripleFormat = "turtle"
    TripleFormatNTriples TripleFormat = "ntriples"
    TripleFormatRDFXML   TripleFormat = "rdfxml"
    TripleFormatJSONLD   TripleFormat = "jsonld"
)
```

`TripleFormat` identifies RDF serialization formats that only support triples.

### QuadFormat

```go
type QuadFormat string

const (
    QuadFormatTriG   QuadFormat = "trig"
    QuadFormatNQuads QuadFormat = "nquads"
)
```

`QuadFormat` identifies RDF serialization formats that support quads (named graphs).

## Interfaces

### TripleDecoder

```go
type TripleDecoder interface {
    Next() (Triple, error)
    Err() error
    Close() error
}
```

`TripleDecoder` streams RDF triples from an input.

**Methods:**
- `Next() (Triple, error)` - Returns the next triple, or `io.EOF` when done
- `Err() error` - Returns any error that occurred during decoding
- `Close() error` - Closes the decoder and releases resources

### QuadDecoder

```go
type QuadDecoder interface {
    Next() (Quad, error)
    Err() error
    Close() error
}
```

`QuadDecoder` streams RDF quads from an input.

**Methods:**
- `Next() (Quad, error)` - Returns the next quad, or `io.EOF` when done
- `Err() error` - Returns any error that occurred during decoding
- `Close() error` - Closes the decoder and releases resources

### TripleEncoder

```go
type TripleEncoder interface {
    Write(Triple) error
    Flush() error
    Close() error
}
```

`TripleEncoder` streams RDF triples to an output.

**Methods:**
- `Write(Triple) error` - Writes a triple to the output
- `Flush() error` - Flushes any buffered data
- `Close() error` - Closes the encoder and releases resources

### QuadEncoder

```go
type QuadEncoder interface {
    Write(Quad) error
    Flush() error
    Close() error
}
```

`QuadEncoder` streams RDF quads to an output.

**Methods:**
- `Write(Quad) error` - Writes a quad to the output
- `Flush() error` - Flushes any buffered data
- `Close() error` - Closes the encoder and releases resources

### TripleHandler

```go
type TripleHandler interface {
    Handle(Triple) error
}
```

`TripleHandler` processes triples in push mode.

### TripleHandlerFunc

```go
type TripleHandlerFunc func(Triple) error

func (h TripleHandlerFunc) Handle(t Triple) error
```

`TripleHandlerFunc` adapts a function to a `TripleHandler`.

### QuadHandler

```go
type QuadHandler interface {
    Handle(Quad) error
}
```

`QuadHandler` processes quads in push mode.

### QuadHandlerFunc

```go
type QuadHandlerFunc func(Quad) error

func (h QuadHandlerFunc) Handle(q Quad) error
```

`QuadHandlerFunc` adapts a function to a `QuadHandler`.

## Functions

### NewTripleDecoder

```go
func NewTripleDecoder(r io.Reader, format TripleFormat) (TripleDecoder, error)
```

`NewTripleDecoder` creates a decoder for the given triple format.

**Parameters:**
- `r` - Input reader
- `format` - Triple format to decode

**Returns:**
- `TripleDecoder` - The decoder instance
- `error` - Error if format is unsupported or initialization fails

**Example:**
```go
dec, err := rdf.NewTripleDecoder(reader, rdf.TripleFormatTurtle)
```

### NewQuadDecoder

```go
func NewQuadDecoder(r io.Reader, format QuadFormat) (QuadDecoder, error)
```

`NewQuadDecoder` creates a decoder for the given quad format.

**Parameters:**
- `r` - Input reader
- `format` - Quad format to decode

**Returns:**
- `QuadDecoder` - The decoder instance
- `error` - Error if format is unsupported or initialization fails

**Example:**
```go
dec, err := rdf.NewQuadDecoder(reader, rdf.QuadFormatTriG)
```

### NewTripleEncoder

```go
func NewTripleEncoder(w io.Writer, format TripleFormat) (TripleEncoder, error)
```

`NewTripleEncoder` creates an encoder for the given triple format.

**Parameters:**
- `w` - Output writer
- `format` - Triple format to encode

**Returns:**
- `TripleEncoder` - The encoder instance
- `error` - Error if format is unsupported or initialization fails

**Example:**
```go
enc, err := rdf.NewTripleEncoder(writer, rdf.TripleFormatNTriples)
```

### NewQuadEncoder

```go
func NewQuadEncoder(w io.Writer, format QuadFormat) (QuadEncoder, error)
```

`NewQuadEncoder` creates an encoder for the given quad format.

**Parameters:**
- `w` - Output writer
- `format` - Quad format to encode

**Returns:**
- `QuadEncoder` - The encoder instance
- `error` - Error if format is unsupported or initialization fails

**Example:**
```go
enc, err := rdf.NewQuadEncoder(writer, rdf.QuadFormatNQuads)
```

### ParseTriples

```go
func ParseTriples(ctx context.Context, r io.Reader, format TripleFormat, handler TripleHandler) error
```

`ParseTriples` streams RDF triples to a handler function.

**Parameters:**
- `ctx` - Context for cancellation
- `r` - Input reader
- `format` - Triple format
- `handler` - Handler to process triples

**Returns:**
- `error` - Error if parsing fails

**Example:**
```go
err := rdf.ParseTriples(ctx, reader, rdf.TripleFormatTurtle,
    rdf.TripleHandlerFunc(func(t rdf.Triple) error {
        // process triple
        return nil
    }),
)
```

### ParseQuads

```go
func ParseQuads(ctx context.Context, r io.Reader, format QuadFormat, handler QuadHandler) error
```

`ParseQuads` streams RDF quads to a handler function.

**Parameters:**
- `ctx` - Context for cancellation
- `r` - Input reader
- `format` - Quad format
- `handler` - Handler to process quads

**Returns:**
- `error` - Error if parsing fails

**Example:**
```go
err := rdf.ParseQuads(ctx, reader, rdf.QuadFormatTriG,
    rdf.QuadHandlerFunc(func(q rdf.Quad) error {
        // process quad
        return nil
    }),
)
```

### ParseTriplesChan

```go
func ParseTriplesChan(ctx context.Context, r io.Reader, format TripleFormat) (<-chan Triple, <-chan error)
```

`ParseTriplesChan` returns a channel of triples and an error channel.

**Parameters:**
- `ctx` - Context for cancellation
- `r` - Input reader
- `format` - Triple format

**Returns:**
- `<-chan Triple` - Channel of triples
- `<-chan error` - Channel for errors

**Example:**
```go
out, errs := rdf.ParseTriplesChan(ctx, reader, rdf.TripleFormatTurtle)
for t := range out {
    // process triple
}
if err, ok := <-errs; ok && err != nil {
    // handle error
}
```

### ParseQuadsChan

```go
func ParseQuadsChan(ctx context.Context, r io.Reader, format QuadFormat) (<-chan Quad, <-chan error)
```

`ParseQuadsChan` returns a channel of quads and an error channel.

**Parameters:**
- `ctx` - Context for cancellation
- `r` - Input reader
- `format` - Quad format

**Returns:**
- `<-chan Quad` - Channel of quads
- `<-chan error` - Channel for errors

**Example:**
```go
out, errs := rdf.ParseQuadsChan(ctx, reader, rdf.QuadFormatTriG)
for q := range out {
    // process quad
}
if err, ok := <-errs; ok && err != nil {
    // handle error
}
```

### ParseTripleFormat

```go
func ParseTripleFormat(value string) (TripleFormat, bool)
```

`ParseTripleFormat` normalizes a format string and returns a `TripleFormat` if valid.

**Parameters:**
- `value` - Format string (e.g., "ttl", "turtle", "nt")

**Returns:**
- `TripleFormat` - The format constant
- `bool` - True if format was recognized

**Supported aliases:**
- Turtle: "turtle", "ttl"
- N-Triples: "ntriples", "nt"
- RDF/XML: "rdfxml", "rdf", "xml"
- JSON-LD: "jsonld", "json-ld", "json"

**Example:**
```go
format, ok := rdf.ParseTripleFormat("ttl")
```

### ParseQuadFormat

```go
func ParseQuadFormat(value string) (QuadFormat, bool)
```

`ParseQuadFormat` normalizes a format string and returns a `QuadFormat` if valid.

**Parameters:**
- `value` - Format string (e.g., "trig", "nq")

**Returns:**
- `QuadFormat` - The format constant
- `bool` - True if format was recognized

**Supported aliases:**
- TriG: "trig"
- N-Quads: "nquads", "nq"

**Example:**
```go
format, ok := rdf.ParseQuadFormat("nq")
```

## Errors

### ErrUnsupportedFormat

```go
var ErrUnsupportedFormat = errors.New("unsupported format")
```

`ErrUnsupportedFormat` is returned when a format is not supported by `NewTripleDecoder`, `NewQuadDecoder`, `NewTripleEncoder`, or `NewQuadEncoder`.

## Best Practices

1. **Always close decoders and encoders**: Use `defer dec.Close()` or `defer enc.Close()`

2. **Handle EOF properly**: Check for `io.EOF` when `Next()` returns an error

3. **Use streaming for large files**: Prefer `NewTripleDecoder`/`NewQuadDecoder` or `ParseTriples`/`ParseQuads` over loading all data into memory

4. **Check format support**: Use `ParseTripleFormat` or `ParseQuadFormat` to validate format strings before creating decoders/encoders

5. **Use the correct format type**: Triple formats can only be used with triple decoders/encoders, and quad formats can only be used with quad decoders/encoders

6. **Use context for cancellation**: Pass a context to `ParseTriples`, `ParseQuads`, `ParseTriplesChan`, and `ParseQuadsChan` to enable cancellation

7. **Flush encoders**: Call `Flush()` before `Close()` to ensure all data is written

