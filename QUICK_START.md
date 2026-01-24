# Quick Start Guide: rdf-go

**Version:** Latest  
**Go Version:** 1.25.5+

---

## Installation

```bash
go get github.com/geoknoesis/rdf-go
```

---

## Basic Usage

### Reading RDF

```go
package main

import (
    "fmt"
    "io"
    "strings"
    "github.com/geoknoesis/rdf-go"
)

func main() {
    input := `<http://example.org/s> <http://example.org/p> <http://example.org/o> .`
    
    dec, err := rdf.NewReader(strings.NewReader(input), rdf.FormatNTriples)
    if err != nil {
        panic(err)
    }
    defer dec.Close()
    
    for {
        stmt, err := dec.Next()
        if err == io.EOF {
            break
        }
        if err != nil {
            panic(err)
        }
        
        fmt.Printf("Subject: %s\n", stmt.S)
        fmt.Printf("Predicate: %s\n", stmt.P)
        fmt.Printf("Object: %s\n", stmt.O)
    }
}
```

### Writing RDF

```go
package main

import (
    "os"
    "github.com/geoknoesis/rdf-go"
)

func main() {
    file, _ := os.Create("output.ttl")
    defer file.Close()
    
    writer, err := rdf.NewWriter(file, rdf.FormatTurtle)
    if err != nil {
        panic(err)
    }
    defer writer.Close()
    
    stmt := rdf.NewTriple(
        rdf.IRI{Value: "http://example.org/s"},
        rdf.IRI{Value: "http://example.org/p"},
        rdf.IRI{Value: "http://example.org/o"},
    )
    
    writer.Write(stmt)
    writer.Flush()
}
```

### Format Auto-Detection

```go
dec, err := rdf.NewReader(reader, rdf.FormatAuto)
// Automatically detects format from input
```

### With Options

```go
dec, err := rdf.NewReader(reader, rdf.FormatTurtle,
    rdf.OptStrictIRIValidation(),           // Enable strict IRI validation
    rdf.OptMaxDepth(50),                    // Limit nesting depth
    rdf.OptContext(ctx),                    // For cancellation
    rdf.OptExpandRDFXMLContainers(),        // Enable container expansion (default)
)
```

---

## Supported Formats

- **Turtle** (`rdf.FormatTurtle`) - Deterministic output
- **TriG** (`rdf.FormatTriG`) - Deterministic output
- **N-Triples** (`rdf.FormatNTriples`) - Deterministic output
- **N-Quads** (`rdf.FormatNQuads`) - Deterministic output
- **RDF/XML** (`rdf.FormatRDFXML`) - Container expansion supported
- **JSON-LD** (`rdf.FormatJSONLD`) - Canonicalization helpers available

---

## Key Features

- ✅ Streaming I/O for efficient processing
- ✅ Format auto-detection
- ✅ Configurable security limits
- ✅ Context cancellation support
- ✅ Strict IRI validation (optional)
- ✅ RDF/XML container expansion
- ✅ JSON-LD canonicalization
- ✅ Deterministic output for most formats

---

## Error Handling

```go
dec, err := rdf.NewReader(reader, rdf.FormatTurtle)
if err != nil {
    return err
}
defer dec.Close()

for {
    stmt, err := dec.Next()
    if err == io.EOF {
        break // Normal end of input
    }
    if err != nil {
        // Check for parse errors with position info
        var parseErr *rdf.ParseError
        if errors.As(err, &parseErr) {
            fmt.Printf("Error at line %d: %v\n", parseErr.Line, parseErr.Err)
        }
        return err
    }
    // Process statement
}
```

---

## Security

For untrusted input, always set explicit limits:

```go
dec, err := rdf.NewReader(reader, rdf.FormatTurtle,
    rdf.OptSafeLimits(),                    // Safe defaults
    rdf.OptContext(ctx),                    // For timeouts
)
```

Or set custom limits:

```go
dec, err := rdf.NewReader(reader, rdf.FormatTurtle,
    rdf.OptMaxLineBytes(64<<10),            // 64KB per line
    rdf.OptMaxStatementBytes(256<<10),       // 256KB per statement
    rdf.OptMaxDepth(50),                    // 50 levels of nesting
    rdf.OptMaxTriples(1_000_000),           // 1M triples max
)
```

---

## More Examples

See `README.md` for comprehensive documentation and examples.

---

**Quick Links:**
- [Full Documentation](README.md)
- [CHANGELOG](CHANGELOG.md)
- [Code Quality Report](FINAL_CODE_QUALITY_REPORT.md)

