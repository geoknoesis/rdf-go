package rdf

import "fmt"

// TermKind identifies RDF term types.
type TermKind uint8

const (
	// TermIRI represents an IRI term.
	TermIRI TermKind = iota
	// TermBlankNode represents a blank node term.
	TermBlankNode
	// TermLiteral represents a literal term.
	TermLiteral
	// TermTriple represents an RDF-star triple term.
	TermTriple
)

// Term is a value that can appear in RDF statements.
type Term interface {
	Kind() TermKind
	String() string
}

// IRI represents an RDF IRI.
type IRI struct {
	// Value is the IRI string value.
	Value string
}

// Kind returns TermIRI.
func (i IRI) Kind() TermKind { return TermIRI }

// String returns the IRI value.
func (i IRI) String() string { return i.Value }

// BlankNode represents an RDF blank node.
type BlankNode struct {
	// ID is the blank node identifier.
	ID string
}

// Kind returns TermBlankNode.
func (b BlankNode) Kind() TermKind { return TermBlankNode }

// String returns the blank node identifier prefixed with "_:".
func (b BlankNode) String() string { return "_:" + b.ID }

// Literal represents an RDF literal.
type Literal struct {
	// Lexical is the lexical form of the literal.
	Lexical string
	// Datatype is the datatype IRI, if any.
	Datatype IRI
	// Lang is the language tag, if any.
	Lang string
}

// Kind returns TermLiteral.
func (l Literal) Kind() TermKind { return TermLiteral }

// String returns a string representation of the literal.
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
	// S is the subject of the quoted triple.
	S Term
	// P is the predicate of the quoted triple.
	P IRI
	// O is the object of the quoted triple.
	O Term
}

// Kind returns TermTriple.
func (t TripleTerm) Kind() TermKind { return TermTriple }

// String returns a string representation of the triple term.
func (t TripleTerm) String() string {
	return fmt.Sprintf("<<%s %s %s>>", t.S.String(), t.P.String(), t.O.String())
}

// Triple is an RDF triple.
type Triple struct {
	// S is the subject.
	S Term
	// P is the predicate.
	P IRI
	// O is the object.
	O Term
}

// Statement represents an RDF statement, which can be either a triple or a quad.
// If G is nil, it represents a triple. If G is non-nil, it represents a quad.
type Statement struct {
	// S is the subject.
	S Term
	// P is the predicate.
	P IRI
	// O is the object.
	O Term
	// G is the graph name, or nil for triples (default graph).
	G Term
}

// IsQuad reports whether the statement is a quad (has a graph).
func (s Statement) IsQuad() bool {
	return s.G != nil
}

// IsTriple reports whether the statement is a triple (no graph).
func (s Statement) IsTriple() bool {
	return s.G == nil
}

// AsTriple returns the statement as a triple (ignores graph).
func (s Statement) AsTriple() Triple {
	return Triple{S: s.S, P: s.P, O: s.O}
}

// AsQuad returns the statement as a quad.
func (s Statement) AsQuad() Quad {
	return Quad{S: s.S, P: s.P, O: s.O, G: s.G}
}

// ToStatement converts a triple to a statement.
func (t Triple) ToStatement() Statement {
	return Statement{S: t.S, P: t.P, O: t.O, G: nil}
}

// ToStatement converts a quad to a statement.
func (q Quad) ToStatement() Statement {
	return Statement{S: q.S, P: q.P, O: q.O, G: q.G}
}

// NewTriple creates a Statement representing a triple (G is nil).
// This is a convenience function for creating triple statements.
func NewTriple(s Term, p IRI, o Term) Statement {
	return Statement{S: s, P: p, O: o, G: nil}
}

// NewQuad creates a Statement representing a quad with the specified graph.
// This is a convenience function for creating quad statements.
func NewQuad(s Term, p IRI, o Term, g Term) Statement {
	return Statement{S: s, P: p, O: o, G: g}
}

// Quad is an RDF quad (triple + optional graph name).
type Quad struct {
	// S is the subject.
	S Term
	// P is the predicate.
	P IRI
	// O is the object.
	O Term
	// G is the graph name, or nil for the default graph.
	G Term
}

// IsZero reports whether the quad has no subject/predicate/object.
func (q Quad) IsZero() bool {
	return q.S == nil && q.P.Value == "" && q.O == nil && q.G == nil
}

// ToTriple extracts the triple from a quad (ignores graph).
func (q Quad) ToTriple() Triple {
	return Triple{S: q.S, P: q.P, O: q.O}
}

// InDefaultGraph reports whether the quad is in the default graph (no named graph).
func (q Quad) InDefaultGraph() bool {
	return q.G == nil
}

// ToQuad converts a triple to a quad in the default graph.
func (t Triple) ToQuad() Quad {
	return Quad{S: t.S, P: t.P, O: t.O, G: nil}
}

// ToQuadInGraph converts a triple to a quad in a named graph.
func (t Triple) ToQuadInGraph(graph Term) Quad {
	return Quad{S: t.S, P: t.P, O: t.O, G: graph}
}
