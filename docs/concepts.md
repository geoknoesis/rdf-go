# Concepts

This document explains the core concepts in `rdf-go`, including RDF terms, statements, and how they're represented in the library.

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
stmt := rdf.Statement{
    S: quoted,
    P: rdf.IRI{Value: "http://example.org/asserted"},
    O: rdf.Literal{Lexical: "true"},
    G: nil,
}
```

## Statements

### Statement

A `Statement` represents an RDF statement, which can be either a triple or a quad. This is the primary type used in the unified API.

```go
// A triple (G is nil)
stmt := rdf.Statement{
    S: rdf.IRI{Value: "http://example.org/alice"},
    P: rdf.IRI{Value: "http://example.org/name"},
    O: rdf.Literal{Lexical: "Alice"},
    G: nil, // nil indicates a triple
}

// A quad (G is non-nil)
stmt := rdf.Statement{
    S: rdf.IRI{Value: "http://example.org/alice"},
    P: rdf.IRI{Value: "http://example.org/name"},
    O: rdf.Literal{Lexical: "Alice"},
    G: rdf.IRI{Value: "http://example.org/graph1"}, // non-nil indicates a quad
}

// Check statement type
if stmt.IsTriple() {
    // This is a triple
}
if stmt.IsQuad() {
    // This is a quad
}

// Convert if needed
triple := stmt.AsTriple()
quad := stmt.AsQuad()
```

### Triple

A `Triple` consists of three components:
- **Subject (S)**: The resource the statement is about
- **Predicate (P)**: The property or relationship
- **Object (O)**: The value or target resource

```go
triple := rdf.Triple{
    S: rdf.IRI{Value: "http://example.org/alice"},
    P: rdf.IRI{Value: "http://example.org/name"},
    O: rdf.Literal{Lexical: "Alice"},
}

// Convert to Statement
stmt := triple.ToStatement()
```

### Quad

A `Quad` extends a triple with an optional fourth component:
- **Graph (G)**: The named graph containing the triple

```go
quad := rdf.Quad{
    S: rdf.IRI{Value: "http://example.org/alice"},
    P: rdf.IRI{Value: "http://example.org/name"},
    O: rdf.Literal{Lexical: "Alice"},
    G: rdf.IRI{Value: "http://example.org/graph1"},
}

// Convert to Statement
stmt := quad.ToStatement()

// Check if in default graph
if quad.InDefaultGraph() {
    // G is nil
}
```

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

- **Readers** use a pull-style API: call `Next()` to get the next statement
- **Writers** use a push-style API: call `Write()` to output a statement

This design minimizes memory usage and allows processing of large RDF datasets without loading everything into memory.

## Format Support

The library supports multiple RDF serialization formats using a unified `Format` type:

**Triple formats:**
- **Turtle** (`FormatTurtle`): Human-readable format with prefixes
- **N-Triples** (`FormatNTriples`): Simple line-based format
- **RDF/XML** (`FormatRDFXML`): XML-based serialization
- **JSON-LD** (`FormatJSONLD`): JSON-based linked data format

**Quad formats:**
- **TriG** (`FormatTriG`): Turtle with named graphs
- **N-Quads** (`FormatNQuads`): N-Triples with graph names

**Auto-detection:**
- **FormatAuto**: Automatically detect format from input

All formats work with the unified `Reader` and `Writer` interfaces. The `Statement` type is used for all formats, with `G` being `nil` for triple-only formats.

Each format can be used for both reading (decoding) and writing (encoding).
