# rdf-go

`rdf-go` is a small, fast RDF parsing/encoding library with streaming
APIs and RDF-star support. It is designed for low allocations and for use
in pipelines where RDF data should be processed incrementally.

## Author

**Stephane Fellah** - stephanef@geoknoesis.com  
Geosemantic-AI expert with 30 years of experience

Published by **Geoknoesis LLC** - www.geoknoesis.com

## Features

- Streaming decoders (pull style) and encoders (push style).
- Type-safe API: separate triple and quad decoders/encoders.
- Convenience helpers: `ParseTriples`, `ParseQuads`, `ParseTriplesChan`, `ParseQuadsChan`.
- RDF-star via `TripleTerm` values.
- Multiple formats: Turtle, TriG, N-Triples, N-Quads, RDF/XML, JSON-LD.

## Install

```
go get github.com/geoknoesis/rdf-go
```

## Quick Start

### Decode Triples (pull)

```go
input := `<http://example.org/s> <http://example.org/p> "v" .`
dec, err := rdf.NewTripleDecoder(strings.NewReader(input), rdf.TripleFormatNTriples)
if err != nil {
    // handle error
}
defer dec.Close()

for {
    triple, err := dec.Next()
    if err == io.EOF {
        break
    }
    if err != nil {
        // handle error
    }
    // use triple.S / triple.P / triple.O
}
```

### Decode Quads (pull)

```go
input := `<http://example.org/s> <http://example.org/p> "v" <http://example.org/g> .`
dec, err := rdf.NewQuadDecoder(strings.NewReader(input), rdf.QuadFormatNQuads)
if err != nil {
    // handle error
}
defer dec.Close()

for {
    quad, err := dec.Next()
    if err == io.EOF {
        break
    }
    if err != nil {
        // handle error
    }
    // use quad.S / quad.P / quad.O / quad.G
}
```

### Encode Triples (push)

```go
buf := &bytes.Buffer{}
enc, err := rdf.NewTripleEncoder(buf, rdf.TripleFormatNTriples)
if err != nil {
    // handle error
}
defer enc.Close()

_ = enc.Write(rdf.Triple{
    S: rdf.IRI{Value: "http://example.org/s"},
    P: rdf.IRI{Value: "http://example.org/p"},
    O: rdf.Literal{Lexical: "v"},
})
_ = enc.Flush()
```

### Encode Quads (push)

```go
buf := &bytes.Buffer{}
enc, err := rdf.NewQuadEncoder(buf, rdf.QuadFormatNQuads)
if err != nil {
    // handle error
}
defer enc.Close()

_ = enc.Write(rdf.Quad{
    S: rdf.IRI{Value: "http://example.org/s"},
    P: rdf.IRI{Value: "http://example.org/p"},
    O: rdf.Literal{Lexical: "v"},
    G: rdf.IRI{Value: "http://example.org/g"},
})
_ = enc.Flush()
```

### Parse Triples (handler)

```go
count := 0
err := rdf.ParseTriples(context.Background(), strings.NewReader(input), rdf.TripleFormatNTriples,
    rdf.TripleHandlerFunc(func(t rdf.Triple) error {
        count++
        return nil
    }),
)
```

### ParseQuads (handler)

```go
count := 0
err := rdf.ParseQuads(context.Background(), strings.NewReader(input), rdf.QuadFormatNQuads,
    rdf.QuadHandlerFunc(func(q rdf.Quad) error {
        count++
        return nil
    }),
)
```

### ParseTriplesChan (channel)

```go
out, errs := rdf.ParseTriplesChan(context.Background(), strings.NewReader(input), rdf.TripleFormatNTriples)
for t := range out {
    _ = t
}
if err, ok := <-errs; ok && err != nil {
    // handle error
}
```

## RDF-star

Quoted triples are represented by `TripleTerm` and can be nested:

```go
quoted := rdf.TripleTerm{
    S: rdf.IRI{Value: "http://example.org/s"},
    P: rdf.IRI{Value: "http://example.org/p"},
    O: rdf.IRI{Value: "http://example.org/o"},
}
stmt := rdf.Triple{
    S: quoted,
    P: rdf.IRI{Value: "http://example.org/said"},
    O: rdf.Literal{Lexical: "true"},
}
```

## Format Selection

The library provides separate format types for triple-only and quad formats:

```go
// For triple formats (Turtle, N-Triples, RDF/XML, JSON-LD)
format, ok := rdf.ParseTripleFormat("nt")
if !ok {
    // fallback
}

// For quad formats (TriG, N-Quads)
format, ok := rdf.ParseQuadFormat("nq")
if !ok {
    // fallback
}
```

## Supported Formats

**Triple formats:**
- `rdf.TripleFormatTurtle` - Turtle (.ttl)
- `rdf.TripleFormatNTriples` - N-Triples (.nt)
- `rdf.TripleFormatRDFXML` - RDF/XML (.rdf, .xml)
- `rdf.TripleFormatJSONLD` - JSON-LD (.jsonld)

**Quad formats:**
- `rdf.QuadFormatTriG` - TriG (.trig)
- `rdf.QuadFormatNQuads` - N-Quads (.nq)

## Notes

- The API is intentionally small and favors streaming. For large inputs,
  prefer `NewTripleDecoder`/`NewQuadDecoder` or `ParseTriples`/`ParseQuads` instead of buffering all results.
- The API enforces type safety: triple formats can only be used with triple decoders/encoders,
  and quad formats can only be used with quad decoders/encoders.
- For any format that is not supported, `NewTripleDecoder`/`NewQuadDecoder`/`NewTripleEncoder`/`NewQuadEncoder` returns
  `rdf.ErrUnsupportedFormat`.
- RDF/XML container elements (rdf:Bag, rdf:Seq, rdf:Alt, rdf:List) are parsed as node elements;
  container membership expansion is not implemented.

## Security and Limits

### For Untrusted Input

**Always set explicit security limits** when processing untrusted input to prevent resource exhaustion attacks.

```go
// Use SafeDecodeOptions for untrusted input
opts := rdf.SafeDecodeOptions()
dec, err := rdf.NewTripleDecoderWithOptions(r, rdf.TripleFormatTurtle, opts)
```

Or set custom limits:

```go
dec, err := rdf.NewTripleDecoderWithOptions(r, rdf.TripleFormatTurtle, rdf.DecodeOptions{
    MaxLineBytes:      64 << 10,  // 64KB per line
    MaxStatementBytes: 256 << 10, // 256KB per statement
    MaxDepth:          50,         // 50 levels of nesting
    MaxTriples:        1_000_000,  // 1M triples max
    Context:           ctx,        // For cancellation/timeouts
})
```

### Security Limits

The following limits are available in `DecodeOptions`:

- **MaxLineBytes**: Maximum size of a single line (default: 1MB)
- **MaxStatementBytes**: Maximum size of a complete statement (default: 4MB)
- **MaxDepth**: Maximum nesting depth for collections, blank node lists, etc. (default: 100)
- **MaxTriples**: Maximum number of triples/quads to process (default: 10M)
- **Context**: Context for cancellation and timeouts

**Default limits are suitable for trusted input only.** For untrusted input, use `SafeDecodeOptions()` or set stricter limits.

### Error Diagnostics

Errors now include line and column information when available:

```go
triple, err := dec.Next()
if err != nil {
    var parseErr *rdf.ParseError
    if errors.As(err, &parseErr) {
        fmt.Printf("Error at line %d, column %d: %v\n", 
            parseErr.Line, parseErr.Column, parseErr.Err)
    }
}
```

### JSON-LD Limits

JSON-LD decoding supports additional semantic limits:

```go
opts := rdf.JSONLDOptions{
    Context:       ctx,
    MaxInputBytes: 1 << 20,
    MaxNodes:      10000,
    MaxGraphItems: 10000,
    MaxQuads:      20000,
}
dec := rdf.NewJSONLDTripleDecoder(r, opts)
```

**Note**: JSON-LD currently loads the entire document into memory before parsing. This is a known limitation - JSON-LD does not support true streaming due to the need to process the entire JSON structure and expand contexts. For very large documents, consider:
- Using other RDF formats (Turtle, N-Triples, TriG, N-Quads) which support streaming
- Setting `MaxInputBytes` limit in `JSONLDOptions` to prevent excessive memory usage
- Processing JSON-LD documents in smaller chunks if possible

