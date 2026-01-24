# rdf-go

A high-performance, streaming RDF library for Go with support for multiple RDF serialization formats.

## Features

- **Unified API**: Single interface for reading/writing all RDF formats
- **Streaming**: Low-memory, streaming parsers for large datasets
- **Multiple Formats**: Turtle, N-Triples, TriG, N-Quads, RDF/XML, JSON-LD
- **Auto-Detection**: Automatic format detection from input
- **Performance**: Optimized for speed and low memory allocation
- **Security**: Configurable limits for untrusted input
- **RDF-star**: Support for quoted triples (RDF-star)

## Quick Start

```go
import "github.com/geoknoesis/rdf-go/rdf"

// Read RDF with auto-detection
dec, err := rdf.NewReader(strings.NewReader(input), rdf.FormatAuto)
if err != nil {
    log.Fatal(err)
}
defer dec.Close()

for {
    stmt, err := dec.Next()
    if err == io.EOF {
        break
    }
    if err != nil {
        log.Fatal(err)
    }
    // Process statement
    if stmt.IsTriple() {
        // Handle triple
    } else if stmt.IsQuad() {
        // Handle quad
    }
}
```

## Supported Formats

### Triple Formats
- **Turtle** (`FormatTurtle`) - Compact, human-readable format
- **N-Triples** (`FormatNTriples`) - Line-based, simple format
- **RDF/XML** (`FormatRDFXML`) - XML-based format
- **JSON-LD** (`FormatJSONLD`) - JSON-based linked data format

### Quad Formats
- **TriG** (`FormatTriG`) - Extended Turtle with named graphs
- **N-Quads** (`FormatNQuads`) - Extended N-Triples with named graphs

## API Overview

### Reading RDF

```go
// With format specified
dec, err := rdf.NewReader(reader, rdf.FormatTurtle)

// With auto-detection
dec, err := rdf.NewReader(reader, rdf.FormatAuto)

// With options
dec, err := rdf.NewReader(reader, rdf.FormatTurtle,
    rdf.OptMaxLineBytes(1<<20),      // 1MB line limit
    rdf.OptMaxTriples(10_000_000),    // 10M triple limit
    rdf.OptContext(ctx),              // Context for cancellation
)
```

### Writing RDF

```go
enc, err := rdf.NewWriter(writer, rdf.FormatTurtle)
if err != nil {
    log.Fatal(err)
}
defer enc.Close()

stmt := rdf.NewTriple(
    rdf.IRI{Value: "http://example.org/s"},
    rdf.IRI{Value: "http://example.org/p"},
    rdf.IRI{Value: "http://example.org/o"},
)

if err := enc.Write(stmt); err != nil {
    log.Fatal(err)
}
```

### Parsing with Handler

```go
err := rdf.Parse(context.Background(), reader, rdf.FormatTurtle,
    func(stmt rdf.Statement) error {
        // Process each statement
        return nil
    },
)
```

## Error Handling

The library uses structured error codes for programmatic error handling:

```go
err := dec.Next()
if err != nil {
    switch rdf.Code(err) {
    case rdf.ErrCodeLineTooLong:
        // Handle line too long
    case rdf.ErrCodeDepthExceeded:
        // Handle depth exceeded
    case rdf.ErrCodeContextCanceled:
        // Handle cancellation
    default:
        // Handle other errors
    }
}
```

See [ERROR_HANDLING.md](ERROR_HANDLING.md) for comprehensive error handling guide.

## Security

For untrusted input, use safe defaults:

```go
opts := rdf.SafeMode() // Conservative limits
dec, err := rdf.NewReader(untrustedReader, rdf.FormatAuto, opts...)
```

## Performance

The library is optimized for performance:

- Streaming parsers minimize memory usage
- Low allocation design
- Efficient format detection
- Comprehensive benchmarks included

See `benchmarks_test.go` for performance benchmarks and `benchmark_profiling.go` for profiling tools.

## Documentation

- **Package Documentation**: See [doc.go](doc.go) for API documentation
- **Error Handling**: See [ERROR_HANDLING.md](ERROR_HANDLING.md) for error handling guide
- **Examples**: See `examples_test.go` for usage examples
- **Benchmarks**: See `benchmarks_test.go` for performance benchmarks

## License

Copyright 2026 Geoknoesis LLC (www.geoknoesis.com)

## Author

Stephane Fellah (stephanef@geoknoesis.com)  
Geosemantic-AI expert with 30 years of experience

