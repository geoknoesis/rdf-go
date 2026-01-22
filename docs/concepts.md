# Concepts

This document explains the core concepts in `grit/rdf-go`, including RDF terms, triples, quads, and how they're represented in the library.

## RDF Terms

RDF terms are the basic building blocks of RDF data. The library provides several term types that implement the `Term` interface.

### IRI (Internationalized Resource Identifier)

An IRI identifies a resource. In RDF, IRIs are used for subjects, predicates, and sometimes objects.

```go
subject := rdf.IRI{Value: "http://example.org/person/alice"}
predicate := rdf.IRI{Value: "http://www.w3.org/1999/02/22-rdf-syntax-ns#type"}
```

### Blank Node

A blank node represents an anonymous resource. Blank nodes are useful when you don't need to give a resource a specific IRI.

```go
bnode := rdf.BlankNode{ID: "b1"}
```

### Literal

A literal represents a data value, such as a string or number. Literals can have:
- A lexical value (the actual string)
- An optional datatype IRI
- An optional language tag

```go
// Simple string literal
name := rdf.Literal{Lexical: "Alice"}

// Literal with datatype
age := rdf.Literal{
    Lexical:  "30",
    Datatype: rdf.IRI{Value: "http://www.w3.org/2001/XMLSchema#integer"},
}

// Literal with language tag
title := rdf.Literal{
    Lexical: "Hello",
    Lang:    "en",
}
```

### TripleTerm (RDF-star)

A `TripleTerm` represents a quoted triple in RDF-star. This allows you to make statements about statements.

```go
quoted := rdf.TripleTerm{
    S: rdf.IRI{Value: "http://example.org/alice"},
    P: rdf.IRI{Value: "http://example.org/said"},
    O: rdf.Literal{Lexical: "Hello"},
}

// Use the quoted triple as a subject
stmt := rdf.Quad{
    S: quoted,
    P: rdf.IRI{Value: "http://example.org/asserted"},
    O: rdf.Literal{Lexical: "true"},
}
```

## Triples and Quads

### Triple

A triple consists of three components:
- **Subject (S)**: The resource the statement is about
- **Predicate (P)**: The property or relationship
- **Object (O)**: The value or target resource

```go
triple := rdf.Triple{
    S: rdf.IRI{Value: "http://example.org/alice"},
    P: rdf.IRI{Value: "http://example.org/name"},
    O: rdf.Literal{Lexical: "Alice"},
}
```

### Quad

A quad extends a triple with an optional fourth component:
- **Graph (G)**: The named graph containing the triple

```go
quad := rdf.Quad{
    S: rdf.IRI{Value: "http://example.org/alice"},
    P: rdf.IRI{Value: "http://example.org/name"},
    O: rdf.Literal{Lexical: "Alice"},
    G: rdf.IRI{Value: "http://example.org/graph1"},
}
```

If the graph component is `nil`, the quad represents a triple in the default graph.

## Term Interface

All term types implement the `Term` interface:

```go
type Term interface {
    Kind() TermKind
    String() string
}
```

The `Kind()` method returns the type of term:
- `TermIRI`
- `TermBlankNode`
- `TermLiteral`
- `TermTriple`

## Streaming Model

The library uses a streaming model for efficient processing:

- **Decoders** use a pull-style API: call `Next()` to get the next quad
- **Encoders** use a push-style API: call `Write()` to output a quad

This design minimizes memory usage and allows processing of large RDF datasets without loading everything into memory.

## Format Support

The library supports multiple RDF serialization formats:

- **Turtle** (`FormatTurtle`): Human-readable format with prefixes
- **TriG** (`FormatTriG`): Turtle with named graphs
- **N-Triples** (`FormatNTriples`): Simple line-based format
- **N-Quads** (`FormatNQuads`): N-Triples with graph names
- **RDF/XML** (`FormatRDFXML`): XML-based serialization
- **JSON-LD** (`FormatJSONLD`): JSON-based linked data format

Each format can be used for both reading (decoding) and writing (encoding).

