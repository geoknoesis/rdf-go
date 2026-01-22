# grit/rdf-go

`grit/rdf-go` is a small, fast RDF parsing/encoding library with streaming
APIs and RDF-star support. It is designed for low allocations and for use
in pipelines where RDF data should be processed incrementally.

## Features

- Streaming decoders (pull style) and encoders (push style).
- Convenience helpers: `Parse` and `ParseChan`.
- RDF-star via `TripleTerm` values.
- Multiple formats: Turtle, TriG, N-Triples, N-Quads, RDF/XML, JSON-LD.

## Install

```
go get grit/rdf-go
```

## Quick Start

### Decode (pull)

```go
input := `<http://example.org/s> <http://example.org/p> "v" .`
dec, err := rdf.NewDecoder(strings.NewReader(input), rdf.FormatNTriples)
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

### Encode (push)

```go
buf := &bytes.Buffer{}
enc, err := rdf.NewEncoder(buf, rdf.FormatNTriples)
if err != nil {
    // handle error
}
defer enc.Close()

_ = enc.Write(rdf.Quad{
    S: rdf.IRI{Value: "http://example.org/s"},
    P: rdf.IRI{Value: "http://example.org/p"},
    O: rdf.Literal{Lexical: "v"},
})
_ = enc.Flush()
```

### Parse (handler)

```go
count := 0
err := rdf.Parse(context.Background(), strings.NewReader(input), rdf.FormatNTriples,
    rdf.HandlerFunc(func(q rdf.Quad) error {
        count++
        return nil
    }),
)
```

### ParseChan (channel)

```go
out, errs := rdf.ParseChan(context.Background(), strings.NewReader(input), rdf.FormatNTriples)
for q := range out {
    _ = q
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
stmt := rdf.Quad{
    S: quoted,
    P: rdf.IRI{Value: "http://example.org/said"},
    O: rdf.Literal{Lexical: "true"},
}
```

## Format Selection

```go
format, ok := rdf.ParseFormat("nt")
if !ok {
    // fallback
}
```

## Notes

- The API is intentionally small and favors streaming. For large inputs,
  prefer `NewDecoder` or `Parse` instead of buffering all results.
- For any format that is not supported, `NewDecoder`/`NewEncoder` returns
  `rdf.ErrUnsupportedFormat`.

