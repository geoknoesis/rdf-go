package rdf

import "testing"

func TestParseFormat(t *testing.T) {
	cases := []struct {
		input  string
		want   Format
		expect bool
	}{
		{"turtle", FormatTurtle, true},
		{"ttl", FormatTurtle, true},
		{"ntriples", FormatNTriples, true},
		{"nt", FormatNTriples, true},
		{"rdfxml", FormatRDFXML, true},
		{"rdf", FormatRDFXML, true},
		{"xml", FormatRDFXML, true},
		{"jsonld", FormatJSONLD, true},
		{"json-ld", FormatJSONLD, true},
		{"json", FormatJSONLD, true},
		{"unknown", "", false},
		{"trig", FormatTriG, true},
		{"nquads", FormatNQuads, true},
	}
	for _, c := range cases {
		got, ok := ParseFormat(c.input)
		if ok != c.expect {
			t.Fatalf("input %q ok=%v want %v", c.input, ok, c.expect)
		}
		if got != c.want {
			t.Fatalf("input %q got %v want %v", c.input, got, c.want)
		}
	}
}

func TestParseFormat_QuadFormats(t *testing.T) {
	cases := []struct {
		input  string
		want   Format
		expect bool
	}{
		{"trig", FormatTriG, true},
		{"nquads", FormatNQuads, true},
		{"nq", FormatNQuads, true},
		{"unknown", "", false},
	}
	for _, c := range cases {
		got, ok := ParseFormat(c.input)
		if ok != c.expect {
			t.Fatalf("input %q ok=%v want %v", c.input, ok, c.expect)
		}
		if got != c.want {
			t.Fatalf("input %q got %v want %v", c.input, got, c.want)
		}
	}
}
