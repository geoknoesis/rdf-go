# rdf-go

`rdf-go` is a small, fast RDF parsing/encoding library with streaming
APIs and RDF-star support. It is designed for low allocations and for use
in pipelines where RDF data should be processed incrementally.

## Author

**Stephane Fellah** - stephanef@geoknoesis.com  
Geosemantic-AI expert with 30 years of experience

Published by **Geoknoesis LLC** - www.geoknoesis.com

## Features

- Streaming readers (pull style) and writers (push style).
- Unified API: single `Reader` and `Writer` interfaces for all formats.
- `Statement` type: represents either a triple (G is nil) or a quad (G is non-nil).
- Convenience helper: `Parse` for streaming with handler functions.
- RDF-star via `TripleTerm` values.
- Multiple formats: Turtle, TriG, N-Triples, N-Quads, RDF/XML, JSON-LD.
- Automatic format detection with `FormatAuto`.

## Install

```
go get github.com/geoknoesis/rdf-go
```

## Quick Start

### Parse with Auto-Detection

The easiest way to parse RDF when you don't know the format. The library automatically detects the format from the input content:

```go
import (
    "context"
    "io"
    "github.com/geoknoesis/rdf-go"
)

// Parse with auto-detection: FormatAuto tells the library to detect the format
// The handler function is called for each statement found in the input
err := rdf.Parse(context.Background(), reader, rdf.FormatAuto, func(s rdf.Statement) error {
    // Each statement contains S (subject), P (predicate), O (object), and G (graph)
    // For triples, G will be nil. For quads (named graphs), G will be non-nil
    fmt.Printf("Subject: %s, Predicate: %s, Object: %s\n", 
        s.S.String(), s.P.String(), s.O.String())
    
    // Check if this is a quad (has a graph name)
    if s.IsQuad() {
        fmt.Printf("  Graph: %s\n", s.G.String())
    }
    
    // Return nil to continue processing, or an error to stop
    return nil
})

if err != nil {
    // Handle parsing errors (format detection failure, parse errors, etc.)
    log.Fatal(err)
}
```

### Decode (Pull Style)

For more control over the parsing process, use `NewReader` with a pull-style API. This gives you explicit control over when to read the next statement:

```go
import (
    "io"
    "github.com/geoknoesis/rdf-go"
)

// Create a reader for Turtle format
// You can also use rdf.FormatAuto to auto-detect the format
dec, err := rdf.NewReader(reader, rdf.FormatTurtle)
if err != nil {
    // Handle initialization errors (unsupported format, etc.)
    log.Fatal(err)
}
// Always close the reader to release resources
defer dec.Close()

// Pull-style: explicitly request the next statement
for {
    stmt, err := dec.Next()
    if err == io.EOF {
        // End of input reached - normal termination
        break
    }
    if err != nil {
        // Handle parsing errors
        log.Printf("Parse error: %v", err)
        return err
    }
    
    // Process the statement
    // Use stmt.IsTriple() to check if it's a triple (G is nil)
    // Use stmt.IsQuad() to check if it's a quad (G is non-nil)
    if stmt.IsTriple() {
        fmt.Printf("Triple: %s %s %s\n", 
            stmt.S.String(), stmt.P.String(), stmt.O.String())
    } else {
        fmt.Printf("Quad: %s %s %s (graph: %s)\n",
            stmt.S.String(), stmt.P.String(), stmt.O.String(), stmt.G.String())
    }
}
```

### Read All Statements

For small datasets, you can collect all statements into a slice using `Parse`:

```go
import (
    "context"
    "github.com/geoknoesis/rdf-go"
)

// Collect all statements into a slice
var stmts []rdf.Statement
err := rdf.Parse(context.Background(), reader, rdf.FormatAuto, func(s rdf.Statement) error {
    stmts = append(stmts, s)
    return nil
})
if err != nil {
    // Handle errors (format detection failure, parse errors, etc.)
    log.Fatal(err)
}

// Now you can process all statements
fmt.Printf("Loaded %d statements\n", len(stmts))
for i, stmt := range stmts {
    fmt.Printf("Statement %d: %s %s %s\n", 
        i+1, stmt.S.String(), stmt.P.String(), stmt.O.String())
}
```

### Encode (Push Style)

To write RDF data, use `NewWriter` with a push-style API. You explicitly write each statement:

```go
import (
    "bytes"
    "github.com/geoknoesis/rdf-go"
)

// Create a buffer to hold the encoded output
buf := &bytes.Buffer{}

// Create an writer for Turtle format
enc, err := rdf.NewWriter(buf, rdf.FormatTurtle)
if err != nil {
    // Handle initialization errors
    log.Fatal(err)
}
// Always close the writer to flush any remaining data
defer enc.Close()

// Create a statement (triple) - there are two ways:

// Option 1: Omit G field (defaults to nil for triples) - more readable!
stmt := rdf.Statement{
    S: rdf.IRI{Value: "http://example.org/s"},  // Subject: the resource
    P: rdf.IRI{Value: "http://example.org/p"},  // Predicate: the property
    O: rdf.IRI{Value: "http://example.org/o"}, // Object: the value
    // G is omitted and defaults to nil (this is a triple, not a quad)
}

// Option 2: Use the convenience function
stmt := rdf.NewTriple(
    rdf.IRI{Value: "http://example.org/s"},
    rdf.IRI{Value: "http://example.org/p"},
    rdf.IRI{Value: "http://example.org/o"},
)

// Write the statement to the writer
if err := enc.Write(stmt); err != nil {
    // Handle write errors
    log.Fatal(err)
}

// Flush any buffered data (important for some formats)
if err := enc.Flush(); err != nil {
    log.Fatal(err)
}

// The encoded RDF is now in buf
fmt.Print(buf.String())
// Output: <http://example.org/s> <http://example.org/p> <http://example.org/o> .
```

### Write Multiple Statements

For writing multiple statements, use `NewWriter` with a loop:

```go
import (
    "os"
    "github.com/geoknoesis/rdf-go"
)

// Prepare a slice of statements to write
stmts := []rdf.Statement{
    // Create triples - G can be omitted (defaults to nil)
    rdf.Statement{
        S: rdf.IRI{Value: "http://example.org/s1"},
        P: rdf.IRI{Value: "http://example.org/p1"},
        O: rdf.IRI{Value: "http://example.org/o1"},
    },
    // Or use the convenience function
    rdf.NewTriple(
        rdf.IRI{Value: "http://example.org/s2"},
        rdf.IRI{Value: "http://example.org/p2"},
        rdf.IRI{Value: "http://example.org/o2"},
    ),
}

// Write all statements to a file
file, _ := os.Create("output.ttl")
defer file.Close()

writer, err := rdf.NewWriter(file, rdf.FormatTurtle)
if err != nil {
    log.Fatal(err)
}
defer writer.Close()

for _, stmt := range stmts {
    if err := writer.Write(stmt); err != nil {
        log.Fatal(err)
    }
}
if err := writer.Flush(); err != nil {
    log.Fatal(err)
}
```

## RDF-star

RDF-star allows you to make statements about statements using quoted triples. The library represents quoted triples using `TripleTerm`:

```go
// Create a quoted triple - this represents a statement that can be used as a subject or object
quoted := rdf.TripleTerm{
    S: rdf.IRI{Value: "http://example.org/alice"},  // Subject of the quoted triple
    P: rdf.IRI{Value: "http://example.org/said"},  // Predicate of the quoted triple
    O: rdf.Literal{Lexical: "Hello"},                // Object of the quoted triple
}

// Use the quoted triple as a subject in a new statement
// This says: "The statement 'Alice said Hello' is asserted to be true"
stmt := rdf.Statement{
    S: quoted,  // Subject is the quoted triple (RDF-star feature)
    P: rdf.IRI{Value: "http://example.org/asserted"}, // Predicate: "is asserted"
    O: rdf.Literal{Lexical: "true"},                 // Object: true
    // G omitted - defaults to nil (this is a triple, not a quad)
}

// You can encode this to Turtle format, which supports RDF-star
enc, _ := rdf.NewWriter(&buf, rdf.FormatTurtle)
enc.Write(stmt)
// Output: <<http://example.org/alice http://example.org/said "Hello">> 
//          <http://example.org/asserted> "true" .
```



## IRI Validation

The library provides optional strict IRI validation according to RFC 3987:

```go
import "github.com/geoknoesis/rdf-go"

// Enable strict IRI validation
dec, err := rdf.NewReader(reader, rdf.FormatTurtle, rdf.OptStrictIRIValidation())
if err != nil {
    return err
}
defer dec.Close()

// Or validate IRIs programmatically
iri := "http://example.org/resource"
if err := rdf.ValidateIRI(iri); err != nil {
    // Handle invalid IRI
    return fmt.Errorf("invalid IRI: %w", err)
}
```

**Note:** By default, IRI validation is lenient (no validation) for backward compatibility. Format-specific behavior:
- **N-Triples**: Always validates that IRIs have a scheme (absolute IRIs required per spec)
- **Turtle/TriG**: Allows relative IRIs with base resolution; no validation by default
- **RDF/XML**: Allows relative IRIs with base resolution; no validation by default
- **JSON-LD**: No validation by default

Enable `OptStrictIRIValidation()` for additional RFC 3987 validation across all formats.

## Error Handling

The library follows Go's standard error handling patterns. Always check for `io.EOF` to detect end of input:

```go
import (
    "errors"
    "io"
    "github.com/geoknoesis/rdf-go"
)

dec, err := rdf.NewReader(reader, rdf.FormatTurtle)
if err != nil {
    // Handle initialization errors (unsupported format, etc.)
    return err
}
defer dec.Close()

for {
    stmt, err := dec.Next()
    if err == io.EOF {
        // End of input reached - this is normal, not an error
        break
    }
    if err != nil {
        // Check if it's a parse error with position information
        var parseErr *rdf.ParseError
        if errors.As(err, &parseErr) {
            // ParseError includes detailed position information
            fmt.Printf("Parse error at line %d, column %d: %v\n", 
                parseErr.Line, parseErr.Column, parseErr.Err)
            // The error message also includes input excerpts with caret indicators
        } else {
            // Other errors (I/O errors, context cancellation, etc.)
            fmt.Printf("Error: %v\n", err)
        }
        return err
    }
    
    // Successfully read a statement - process it
    // Use stmt.IsTriple() or stmt.IsQuad() to check the statement type
    processStatement(stmt)
}
```

Error messages automatically include:
- Position information (line:column or offset)
- Input excerpts showing context around the error
- Caret indicators pointing to the error position

## Format Selection

The library uses a unified `Format` type for all RDF serialization formats. You can either use format constants or parse format strings:

```go
// Option 1: Parse format from a string (useful for user input or file extensions)
format, ok := rdf.ParseFormat("ttl")  // Returns FormatTurtle
if !ok {
    // Format string not recognized
    return fmt.Errorf("unknown format: %s", "ttl")
}

// Option 2: Use format constants directly (recommended for known formats)
dec, err := rdf.NewReader(reader, rdf.FormatTurtle)

// Option 3: Use auto-detection (library detects format from input)
dec, err := rdf.NewReader(reader, rdf.FormatAuto)
```

### Supported Formats

**Triple formats:**
- `rdf.FormatTurtle` - Turtle (.ttl)
- `rdf.FormatNTriples` - N-Triples (.nt)
- `rdf.FormatRDFXML` - RDF/XML (.rdf, .xml)
- `rdf.FormatJSONLD` - JSON-LD (.jsonld)

**Quad formats:**
- `rdf.FormatTriG` - TriG (.trig)
- `rdf.FormatNQuads` - N-Quads (.nq)

**Auto-detection:**
- `rdf.FormatAuto` - Automatically detect format from input

## Options

Configure reader/writer behavior using functional options. Options are applied in order and can be combined:

```go
import (
    "context"
    "time"
    "github.com/geoknoesis/rdf-go"
)

// Create a context with timeout for cancellation
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

// Configure reader with multiple options
dec, err := rdf.NewReader(reader, rdf.FormatTurtle,
    // Security: Apply safe limits for untrusted input
    // This sets reasonable defaults to prevent resource exhaustion attacks
    rdf.OptSafeLimits(),
    
    // Limit nesting depth (for collections, blank node lists, etc.)
    rdf.OptMaxDepth(50),
    
    // Set context for cancellation and timeouts
    rdf.OptContext(ctx),
    
    // Limit maximum line size (64KB)
    rdf.OptMaxLineBytes(64<<10),
    
    // Limit maximum statement size (256KB)
    rdf.OptMaxStatementBytes(256<<10),
    
    // Limit total number of statements to process
    rdf.OptMaxTriples(1_000_000),
)
if err != nil {
    return err
}
```

Available options:
- `OptContext(ctx)` - Set context for cancellation and timeouts
- `OptMaxLineBytes(n)` - Set maximum line size limit
- `OptMaxStatementBytes(n)` - Set maximum statement size limit
- `OptMaxDepth(n)` - Set maximum nesting depth limit
- `OptMaxTriples(n)` - Set maximum number of triples/quads to process
- `OptSafeLimits()` - Apply safe limits suitable for untrusted input
- `OptStrictIRIValidation()` - Enable strict IRI validation according to RFC 3987
- `OptExpandRDFXMLContainers()` - Enable RDF/XML container membership expansion (default: enabled)
- `OptDisableRDFXMLContainerExpansion()` - Disable RDF/XML container membership expansion

## Versioning & Compatibility

This library follows [Semantic Versioning](https://semver.org/):

- **v1.x.x**: Backward compatible changes only (new features, bug fixes)
- **v2.x.x**: Breaking changes (if needed in the future)

### API Stability

The following APIs are considered stable and will maintain backward compatibility:
- `Reader` and `Writer` interfaces
- `Statement`, `Triple`, `Quad` types
- `Term` interface and implementations (`IRI`, `BlankNode`, `Literal`, `TripleTerm`)
- `NewReader()`, `NewWriter()`, `Parse()` functions
- Format constants (`FormatTurtle`, `FormatNTriples`, etc.)
- Option functions (`OptMaxDepth`, `OptSafeLimits`, etc.)

### Deprecation Policy

- Deprecated APIs will be marked with `// Deprecated:` comments
- Deprecated APIs will be removed in the next major version
- At least one minor version will include deprecation warnings before removal

### Go Version Support

- **Minimum Go version**: 1.25.5
- The library uses pure Go (no CGO dependencies)
- Compatible with all Go versions that support the minimum version

## Notes

- The API is intentionally small and favors streaming. For large inputs,
  use `NewReader` or `Parse` for efficient processing.
- All formats work with the unified `Reader` and `Writer` interfaces.
- The `Statement` type represents either a triple (G is nil) or a quad (G is non-nil).
- Use `stmt.IsTriple()` or `stmt.IsQuad()` to check the statement type.
- For any unsupported format, `NewReader`/`NewWriter` returns `rdf.ErrUnsupportedFormat`.
- RDF/XML container elements (rdf:Bag, rdf:Seq, rdf:Alt, rdf:List) support container membership expansion.
  By default, `rdf:li` elements are automatically converted to `rdf:_1`, `rdf:_2`, etc.
  Use `OptDisableRDFXMLContainerExpansion()` to disable this behavior.

## Security and Limits

### For Untrusted Input

**Always set explicit security limits** when processing untrusted input to prevent resource exhaustion attacks.

```go
// Use OptSafeLimits for untrusted input
dec, err := rdf.NewReader(r, rdf.FormatTurtle, rdf.OptSafeLimits())
```

Or set custom limits:

```go
dec, err := rdf.NewReader(r, rdf.FormatTurtle,
    rdf.OptMaxLineBytes(64<<10),      // 64KB per line
    rdf.OptMaxStatementBytes(256<<10), // 256KB per statement
    rdf.OptMaxDepth(50),              // 50 levels of nesting
    rdf.OptMaxTriples(1_000_000),      // 1M triples max
    rdf.OptContext(ctx),               // For cancellation/timeouts
)
```

### Security Limits

The following limits are available via options:

- **MaxLineBytes**: Maximum size of a single line (default: 1MB)
- **MaxStatementBytes**: Maximum size of a complete statement (default: 4MB)
- **MaxDepth**: Maximum nesting depth for collections, blank node lists, etc. (default: 100)
- **MaxTriples**: Maximum number of triples/quads to process (default: 10M)
- **Context**: Context for cancellation and timeouts

**Default limits are suitable for trusted input only.** For untrusted input, use `SafeDecodeOptions()` or set stricter limits.

### Error Diagnostics

Errors include line and column information, along with input excerpts for better debugging:

```go
stmt, err := dec.Next()
if err != nil {
    var parseErr *rdf.ParseError
    if errors.As(err, &parseErr) {
        fmt.Printf("Error at line %d, column %d: %v\n", 
            parseErr.Line, parseErr.Column, parseErr.Err)
        // Error messages automatically include input excerpts with caret indicators
        // Example output:
        // turtle:3:15: unexpected token
        //   ex:s ex:p ex:o .
        //            ^
    }
}
```

Error messages automatically include:
- Position information (line:column or offset)
- Input excerpts showing context around the error
- Caret indicators pointing to the error position

### Error Codes

For programmatic error handling, use the `Code()` function to get error codes:

```go
import "github.com/geoknoesis/rdf-go"

stmt, err := dec.Next()
if err != nil {
    code := rdf.Code(err)
    switch code {
    case rdf.ErrCodeUnsupportedFormat:
        // Handle unsupported format
    case rdf.ErrCodeLineTooLong:
        // Handle line too long
    case rdf.ErrCodeStatementTooLong:
        // Handle statement too long
    case rdf.ErrCodeDepthExceeded:
        // Handle depth exceeded
    case rdf.ErrCodeTripleLimitExceeded:
        // Handle triple limit exceeded
    case rdf.ErrCodeContextCanceled:
        // Handle context cancellation
    case rdf.ErrCodeParseError:
        // Handle general parse error
    default:
        // Handle unknown error
    }
}
```

**Available Error Codes:**
- `ErrCodeUnsupportedFormat` - Unsupported RDF format
- `ErrCodeLineTooLong` - Line exceeded configured limit
- `ErrCodeStatementTooLong` - Statement exceeded configured limit
- `ErrCodeDepthExceeded` - Nesting depth exceeded configured limit
- `ErrCodeTripleLimitExceeded` - Maximum number of triples/quads exceeded
- `ErrCodeParseError` - General parse error
- `ErrCodeContextCanceled` - Context was canceled
- `ErrCodeInvalidIRI` - Invalid IRI encountered
- `ErrCodeInvalidLiteral` - Invalid literal encountered

**Note:** `Code()` returns an empty string for `nil` errors and `io.EOF` (which is not an error condition).

### JSON-LD Options

JSON-LD decoding supports additional semantic limits via `JSONLDOptions`:

```go
jsonldOpts := rdf.JSONLDOptions{
    Context:       ctx,
    MaxInputBytes: 1 << 20,
    MaxNodes:      10000,
    MaxGraphItems: 10000,
    MaxQuads:      20000,
}
// JSON-LD options are used internally when FormatJSONLD is specified
dec, err := rdf.NewReader(r, rdf.FormatJSONLD)
```

**Note**: JSON-LD supports streaming parsing using `json.Reader` token-by-token processing. The reader processes nodes incrementally and emits triples/quads as they are parsed. However, there are some limitations:
- When `@graph` appears before `@context` in the JSON structure, the graph items must be buffered until the context is available (this is necessary for correct term expansion)
- Nested objects and arrays are fully decoded into memory (this is required for JSON-LD context processing and term expansion)
- **Remote context resolution**: The streaming reader supports remote context URLs when a `DocumentLoader` is provided in `JSONLDOptions`. If `@context` is a string URL, it will be loaded via the `DocumentLoader` before processing.
- For very large documents with `@graph` before `@context`, consider reordering the JSON structure to place `@context` first, or use other RDF formats (Turtle, N-Triples, TriG, N-Quads) which have more efficient streaming characteristics

---

## Output Determinism

The library provides deterministic output for most formats, with some format-specific considerations:

### Deterministic Formats

**Turtle/TriG:**
- Prefix declarations are sorted alphabetically (deterministic order)
- Statement order matches input order
- Prefix selection uses longest matching namespace (deterministic algorithm)
- Blank node labels are preserved from input

**N-Triples/N-Quads:**
- Output order matches input order exactly
- No prefix abbreviations (fully expanded IRIs)
- Fully deterministic output

**RDF/XML:**
- XML structure is deterministic
- Element order matches input order

### Non-Deterministic Formats

**JSON-LD:**
- ⚠️ **Key ordering is non-deterministic** due to Go's map iteration order
- JSON object keys may appear in different orders across runs
- This is a limitation of Go's `encoding/json` package
- For deterministic JSON-LD output, use the `CanonicalizeJSONLD()` function:

```go
import "github.com/geoknoesis/rdf-go"

// Encode to JSON-LD
var buf bytes.Buffer
enc, _ := rdf.NewWriter(&buf, rdf.FormatJSONLD)
enc.Write(stmt)
enc.Close()

// Canonicalize for deterministic output
canonical, err := rdf.CanonicalizeJSONLD(buf.Bytes())
if err != nil {
    return err
}
// canonical now contains deterministic JSON-LD
```

### Best Practices

- For reproducible builds, use Turtle, TriG, N-Triples, or N-Quads
- If JSON-LD determinism is required, post-process with a JSON canonicalizer
- Round-trip tests verify semantic equivalence (isomorphic graphs) rather than byte-for-byte equality

## Supported Features Matrix

| Feature | Turtle | TriG | N-Triples | N-Quads | RDF/XML | JSON-LD |
|---------|--------|------|-----------|---------|---------|---------|
| **Parsing** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Encoding** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Named Graphs** | ❌ | ✅ | ❌ | ✅ | ❌ | ✅ |
| **Prefixes** | ✅ | ✅ | ❌ | ❌ | ✅ | ✅ |
| **Base IRI** | ✅ | ✅ | ❌ | ❌ | ✅ | ✅ |
| **Blank Nodes** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Collections** | ✅ | ✅ | ❌ | ❌ | ✅ | ✅ |
| **RDF-star** | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ |
| **Streaming** | ✅ | ✅ | ✅ | ✅ | ✅ | ⚠️* |
| **Deterministic Output** | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ |

\* JSON-LD streaming has limitations: buffers `@graph` when it appears before `@context`

### Format-Specific Notes

- **RDF/XML**: Container membership expansion is implemented and enabled by default.
  Use `OptDisableRDFXMLContainerExpansion()` to disable automatic expansion of `rdf:li` to `rdf:_n`.
- **JSON-LD**: Compaction and framing are not supported (I/O library only)
- **JSON-LD**: Remote context resolution supported when `DocumentLoader` is provided

## Performance

The library is optimized for performance with:
- Streaming architecture to minimize memory usage
- Low allocation patterns using `strings.Builder` and buffer reuse
- Efficient string operations and parsing
- Comprehensive benchmarks available in `rdf/benchmarks_test.go`

### Running Benchmarks

Run benchmarks:
```bash
go test ./rdf -bench=. -benchmem -run=^$
```

### Benchmark Results

Benchmark results vary by system and input size. Key benchmarks include:

- `BenchmarkTurtleDecodeLarge` - Large Turtle file decoding
- `BenchmarkNTriplesDecodeLarge` - Large N-Triples file decoding
- `BenchmarkTriGDecode` - TriG format decoding
- `BenchmarkJSONLDDecode` - JSON-LD format decoding
- `BenchmarkTurtleEncode` - Turtle encoding
- `BenchmarkNTriplesEncodeLarge` - N-Triples encoding
- `BenchmarkUnescapeString` - String unescaping performance
- `BenchmarkResolveIRI` - IRI resolution performance
- `BenchmarkFormatDetection` - Format detection performance

**Performance Characteristics:**
- **Streaming**: All formats support streaming with constant memory usage (except JSON-LD edge cases)
- **Throughput**: Typically 10K-100K+ triples/second depending on format and input complexity
- **Memory**: O(1) memory usage for streaming parsers (bounded by security limits)
- **Allocations**: Optimized to minimize allocations using buffer reuse and `strings.Builder`

For detailed performance analysis, run benchmarks on your target system and input data.

