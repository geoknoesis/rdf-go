package rdf

import "fmt"

// TermKind identifies RDF term types.
type TermKind uint8

const (
	TermIRI TermKind = iota
	TermBlankNode
	TermLiteral
	TermTriple
)

// Term is a value that can appear in RDF statements.
type Term interface {
	Kind() TermKind
	String() string
}

// IRI represents an RDF IRI.
type IRI struct {
	Value string
}

func (i IRI) Kind() TermKind { return TermIRI }
func (i IRI) String() string { return i.Value }

// BlankNode represents an RDF blank node.
type BlankNode struct {
	ID string
}

func (b BlankNode) Kind() TermKind { return TermBlankNode }
func (b BlankNode) String() string { return "_:" + b.ID }

// Literal represents an RDF literal.
type Literal struct {
	Lexical  string
	Datatype IRI
	Lang     string
}

func (l Literal) Kind() TermKind { return TermLiteral }
func (l Literal) String() string {
	if l.Lang != "" {
		return fmt.Sprintf("%q@%s", l.Lexical, l.Lang)
	}
	if l.Datatype.Value != "" {
		return fmt.Sprintf("%q^^<%s>", l.Lexical, l.Datatype.Value)
	}
	return fmt.Sprintf("%q", l.Lexical)
}

// TripleTerm is an RDF-star quoted triple term.
type TripleTerm struct {
	S Term
	P IRI
	O Term
}

func (t TripleTerm) Kind() TermKind { return TermTriple }
func (t TripleTerm) String() string {
	return fmt.Sprintf("<<%s %s %s>>", t.S.String(), t.P.String(), t.O.String())
}

// Triple is an RDF triple.
type Triple struct {
	S Term
	P IRI
	O Term
}

// Quad is an RDF quad (triple + optional graph name).
type Quad struct {
	S Term
	P IRI
	O Term
	G Term
}

// IsZero reports whether the quad has no subject/predicate/object.
func (q Quad) IsZero() bool {
	return q.S == nil && q.P.Value == "" && q.O == nil && q.G == nil
}
