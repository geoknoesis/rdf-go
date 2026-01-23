package rdf

import "testing"

func TestParseTripleFormat(t *testing.T) {
	cases := []struct {
		input  string
		want   TripleFormat
		expect bool
	}{
		{"turtle", TripleFormatTurtle, true},
		{"ttl", TripleFormatTurtle, true},
		{"ntriples", TripleFormatNTriples, true},
		{"nt", TripleFormatNTriples, true},
		{"rdfxml", TripleFormatRDFXML, true},
		{"rdf", TripleFormatRDFXML, true},
		{"xml", TripleFormatRDFXML, true},
		{"jsonld", TripleFormatJSONLD, true},
		{"json-ld", TripleFormatJSONLD, true},
		{"json", TripleFormatJSONLD, true},
		{"unknown", "", false},
		{"trig", "", false},
		{"nquads", "", false},
	}
	for _, c := range cases {
		got, ok := ParseTripleFormat(c.input)
		if ok != c.expect {
			t.Fatalf("input %q ok=%v want %v", c.input, ok, c.expect)
		}
		if got != c.want {
			t.Fatalf("input %q got %v want %v", c.input, got, c.want)
		}
	}
}

func TestParseQuadFormat(t *testing.T) {
	cases := []struct {
		input  string
		want   QuadFormat
		expect bool
	}{
		{"trig", QuadFormatTriG, true},
		{"nquads", QuadFormatNQuads, true},
		{"nq", QuadFormatNQuads, true},
		{"unknown", "", false},
		{"turtle", "", false},
		{"ntriples", "", false},
	}
	for _, c := range cases {
		got, ok := ParseQuadFormat(c.input)
		if ok != c.expect {
			t.Fatalf("input %q ok=%v want %v", c.input, ok, c.expect)
		}
		if got != c.want {
			t.Fatalf("input %q got %v want %v", c.input, got, c.want)
		}
	}
}
