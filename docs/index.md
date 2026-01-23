# rdf-go

A small, fast RDF parsing/encoding library with streaming APIs and RDF-star support.

## Overview

`rdf-go` is designed for low allocations and for use in pipelines where RDF data should be processed incrementally. It provides a compact RDF model with streaming parsers and encoders, focusing on fast, low-allocation I/O with a small surface area.

## Key Features

- **Streaming APIs**: Pull-style decoders and push-style encoders for efficient memory usage
- **Unified API**: Single `Reader` and `Writer` interfaces for all formats
- **Statement Type**: Represents either a triple (G is nil) or a quad (G is non-nil)
- **Multiple Formats**: Support for Turtle, TriG, N-Triples, N-Quads, RDF/XML, and JSON-LD
- **Auto-Detection**: Automatic format detection with `FormatAuto`
- **RDF-star Support**: Quoted triples via `TripleTerm` values
- **Convenience Helpers**: `Parse`, `ReadAll`, `WriteAll` functions for easy integration
- **Low Allocations**: Optimized for performance in high-throughput scenarios

## Quick Example

```go
import (
    "io"
    "strings"
    "github.com/geoknoesis/rdf-go"
)

input := `<http://example.org/s> <http://example.org/p> "v" .`
reader, err := rdf.NewReader(strings.NewReader(input), rdf.FormatNTriples)
if err != nil {
    // handle error
}
defer dec.Close()

for {
    stmt, err := dec.Next()
    if err == io.EOF {
        break
    }
    if err != nil {
        // handle error
    }
    // process statement (stmt.S, stmt.P, stmt.O)
    // Use stmt.IsTriple() or stmt.IsQuad() to check type
}
```

## Documentation

- [Getting Started](getting-started.md) - Installation and basic usage
- [Concepts](concepts.md) - Understanding RDF terms, statements, and formats
- [How To](how-to.md) - Common tasks and patterns
- [Reference](reference.md) - Complete API documentation

## Installation

```bash
go get github.com/geoknoesis/rdf-go
```

## License

See the repository for license information.
