package rdf

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestRoundTripTriplesIsomorphic(t *testing.T) {
	triples := []Triple{
		{S: IRI{Value: "http://example.org/s"}, P: IRI{Value: "http://example.org/p"}, O: Literal{Lexical: "v", Lang: "en"}},
		{S: BlankNode{ID: "b1"}, P: IRI{Value: "http://example.org/p2"}, O: IRI{Value: "http://example.org/o"}},
		{S: IRI{Value: "http://example.org/s2"}, P: IRI{Value: "http://example.org/p3"}, O: Literal{Lexical: "1", Datatype: IRI{Value: "http://www.w3.org/2001/XMLSchema#integer"}}},
	}

	formats := []TripleFormat{TripleFormatNTriples, TripleFormatTurtle, TripleFormatRDFXML, TripleFormatJSONLD}
	for _, format := range formats {
		var buf bytes.Buffer
		enc, err := NewTripleEncoder(&buf, format)
		if err != nil {
			t.Fatalf("format %s: %v", format, err)
		}
		for _, triple := range triples {
			if err := enc.Write(triple); err != nil {
				t.Fatalf("format %s: write error %v", format, err)
			}
		}
		if err := enc.Close(); err != nil {
			t.Fatalf("format %s: close error %v", format, err)
		}

		dec, err := NewTripleDecoder(strings.NewReader(buf.String()), format)
		if err != nil {
			t.Fatalf("format %s: %v", format, err)
		}
		parsed, err := collectTriples(dec)
		if err != nil {
			t.Fatalf("format %s: decode error %v", format, err)
		}
		if err := dec.Close(); err != nil {
			t.Fatalf("format %s: decoder close error %v", format, err)
		}

		want := triplesToQuads(triples)
		got := triplesToQuads(parsed)
		if !isomorphicQuads(want, got) {
			t.Fatalf("format %s: roundtrip graphs are not isomorphic", format)
		}
	}
}

func TestRoundTripQuadsIsomorphic(t *testing.T) {
	quads := []Quad{
		{S: IRI{Value: "http://example.org/s"}, P: IRI{Value: "http://example.org/p"}, O: Literal{Lexical: "v"}},
		{S: BlankNode{ID: "b1"}, P: IRI{Value: "http://example.org/p2"}, O: IRI{Value: "http://example.org/o"}, G: IRI{Value: "http://example.org/g"}},
		{S: IRI{Value: "http://example.org/s2"}, P: IRI{Value: "http://example.org/p3"}, O: BlankNode{ID: "b2"}},
	}

	formats := []QuadFormat{QuadFormatNQuads, QuadFormatTriG}
	for _, format := range formats {
		var buf bytes.Buffer
		enc, err := NewQuadEncoder(&buf, format)
		if err != nil {
			t.Fatalf("format %s: %v", format, err)
		}
		for _, quad := range quads {
			if err := enc.Write(quad); err != nil {
				t.Fatalf("format %s: write error %v", format, err)
			}
		}
		if err := enc.Close(); err != nil {
			t.Fatalf("format %s: close error %v", format, err)
		}

		dec, err := NewQuadDecoder(strings.NewReader(buf.String()), format)
		if err != nil {
			t.Fatalf("format %s: %v", format, err)
		}
		parsed, err := collectQuads(dec)
		if err != nil {
			t.Fatalf("format %s: decode error %v", format, err)
		}
		if err := dec.Close(); err != nil {
			t.Fatalf("format %s: decoder close error %v", format, err)
		}

		if !isomorphicQuads(quads, parsed) {
			t.Fatalf("format %s: roundtrip graphs are not isomorphic", format)
		}
	}
}

func collectTriples(dec TripleDecoder) ([]Triple, error) {
	var triples []Triple
	for {
		triple, err := dec.Next()
		if err != nil {
			if err == io.EOF {
				return triples, nil
			}
			return nil, err
		}
		triples = append(triples, triple)
	}
}

func collectQuads(dec QuadDecoder) ([]Quad, error) {
	var quads []Quad
	for {
		quad, err := dec.Next()
		if err != nil {
			if err == io.EOF {
				return quads, nil
			}
			return nil, err
		}
		if quad.IsZero() {
			continue
		}
		quads = append(quads, quad)
	}
}

func triplesToQuads(triples []Triple) []Quad {
	quads := make([]Quad, len(triples))
	for i, t := range triples {
		quads[i] = t.ToQuad()
	}
	return quads
}

func isomorphicQuads(a, b []Quad) bool {
	if len(a) != len(b) {
		return false
	}
	aBNodes := isoCollectBlankNodes(a)
	bBNodes := isoCollectBlankNodes(b)
	if len(aBNodes) != len(bBNodes) {
		return false
	}
	bCounts := quadCountMap(b, nil)
	if len(aBNodes) == 0 {
		return quadCountEquals(quadCountMap(a, nil), bCounts)
	}
	mapping := map[string]string{}
	used := map[string]bool{}

	var search func(idx int) bool
	search = func(idx int) bool {
		if idx == len(aBNodes) {
			return quadCountEquals(quadCountMap(a, mapping), bCounts)
		}
		source := aBNodes[idx]
		for _, target := range bBNodes {
			if used[target] {
				continue
			}
			mapping[source] = target
			if mappingConsistent(a, mapping, bCounts) {
				used[target] = true
				if search(idx + 1) {
					return true
				}
				used[target] = false
			}
			delete(mapping, source)
		}
		return false
	}

	return search(0)
}

func mappingConsistent(quads []Quad, mapping map[string]string, targetCounts map[string]int) bool {
	counts := map[string]int{}
	for _, quad := range quads {
		key, ok := isoQuadKey(quad, mapping, true)
		if !ok {
			continue
		}
		counts[key]++
		if counts[key] > targetCounts[key] {
			return false
		}
	}
	return true
}

func quadCountMap(quads []Quad, mapping map[string]string) map[string]int {
	counts := map[string]int{}
	for _, quad := range quads {
		key, ok := isoQuadKey(quad, mapping, mapping != nil)
		if !ok {
			return map[string]int{}
		}
		counts[key]++
	}
	return counts
}

func quadCountEquals(a, b map[string]int) bool {
	if len(a) != len(b) {
		return false
	}
	for key, count := range a {
		if b[key] != count {
			return false
		}
	}
	return true
}

func isoQuadKey(q Quad, mapping map[string]string, requireMapped bool) (string, bool) {
	subject, ok := isoTermKey(q.S, mapping, requireMapped)
	if !ok {
		return "", false
	}
	predicate := "I:" + q.P.Value
	object, ok := isoTermKey(q.O, mapping, requireMapped)
	if !ok {
		return "", false
	}
	graph := "G:default"
	if q.G != nil {
		graphTerm, ok := isoTermKey(q.G, mapping, requireMapped)
		if !ok {
			return "", false
		}
		graph = "G:" + graphTerm
	}
	return subject + "|" + predicate + "|" + object + "|" + graph, true
}

func isoTermKey(term Term, mapping map[string]string, requireMapped bool) (string, bool) {
	switch value := term.(type) {
	case IRI:
		return "I:" + value.Value, true
	case BlankNode:
		if mapping == nil {
			return "B:" + value.ID, true
		}
		mapped, ok := mapping[value.ID]
		if !ok {
			return "", !requireMapped
		}
		return "B:" + mapped, true
	case Literal:
		return "L:" + value.Lexical + "|lang:" + value.Lang + "|dt:" + value.Datatype.Value, true
	case TripleTerm:
		subject, ok := isoTermKey(value.S, mapping, requireMapped)
		if !ok {
			return "", false
		}
		object, ok := isoTermKey(value.O, mapping, requireMapped)
		if !ok {
			return "", false
		}
		return "T:" + subject + "|P:" + value.P.Value + "|O:" + object, true
	default:
		return "", false
	}
}

func isoCollectBlankNodes(quads []Quad) []string {
	seen := map[string]bool{}
	for _, quad := range quads {
		isoCollectBlankNodesFromTerm(quad.S, seen)
		isoCollectBlankNodesFromTerm(quad.O, seen)
		if quad.G != nil {
			isoCollectBlankNodesFromTerm(quad.G, seen)
		}
	}
	out := make([]string, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	return out
}

func isoCollectBlankNodesFromTerm(term Term, seen map[string]bool) {
	switch value := term.(type) {
	case BlankNode:
		seen[value.ID] = true
	case TripleTerm:
		isoCollectBlankNodesFromTerm(value.S, seen)
		isoCollectBlankNodesFromTerm(value.O, seen)
	}
}
