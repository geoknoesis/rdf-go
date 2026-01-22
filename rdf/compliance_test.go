package rdf

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNTriplesStar_Parse(t *testing.T) {
	input := "<< <http://example.org/s> <http://example.org/p> <http://example.org/o> >> <http://example.org/p2> \"v\" .\n"
	dec, err := NewDecoder(strings.NewReader(input), FormatNTriples)
	if err != nil {
		t.Fatalf("decoder error: %v", err)
	}
	quad, err := dec.Next()
	if err != nil {
		t.Fatalf("next error: %v", err)
	}
	if quad.S == nil || quad.P.Value == "" || quad.O == nil {
		t.Fatalf("unexpected quad: %#v", quad)
	}
}

func TestTurtle_ParseBasic(t *testing.T) {
	input := "@prefix ex: <http://example.org/> .\nex:s ex:p \"v\" .\n"
	dec, err := NewDecoder(strings.NewReader(input), FormatTurtle)
	if err != nil {
		t.Fatalf("decoder error: %v", err)
	}
	quad, err := dec.Next()
	if err != nil {
		t.Fatalf("next error: %v", err)
	}
	if quad.P.Value != "http://example.org/p" {
		t.Fatalf("unexpected predicate: %s", quad.P.Value)
	}
}

func TestTriG_ParseBasic(t *testing.T) {
	input := "@prefix ex: <http://example.org/> .\nex:g { ex:s ex:p ex:o . }\n"
	dec, err := NewDecoder(strings.NewReader(input), FormatTriG)
	if err != nil {
		t.Fatalf("decoder error: %v", err)
	}
	quad, err := dec.Next()
	if err != nil {
		t.Fatalf("next error: %v", err)
	}
	if quad.G == nil {
		t.Fatalf("expected graph term")
	}
}

func TestJSONLD_ParseBasic(t *testing.T) {
	input := `{"@context":{"ex":"http://example.org/"},"@id":"ex:s","ex:p":"v"}`
	dec, err := NewDecoder(strings.NewReader(input), FormatJSONLD)
	if err != nil {
		t.Fatalf("decoder error: %v", err)
	}
	quad, err := dec.Next()
	if err != nil {
		t.Fatalf("next error: %v", err)
	}
	if quad.P.Value != "http://example.org/p" {
		t.Fatalf("unexpected predicate: %s", quad.P.Value)
	}
}

func TestRDFXML_ParseBasic(t *testing.T) {
	input := `<?xml version="1.0"?><rdf:RDF xmlns:rdf="` + rdfXMLNS + `"><rdf:Description rdf:about="http://example.org/s"><ex:p xmlns:ex="http://example.org/">v</ex:p></rdf:Description></rdf:RDF>`
	dec, err := NewDecoder(strings.NewReader(input), FormatRDFXML)
	if err != nil {
		t.Fatalf("decoder error: %v", err)
	}
	quad, err := dec.Next()
	if err != nil {
		t.Fatalf("next error: %v", err)
	}
	if quad.P.Value != "http://example.org/p" {
		t.Fatalf("unexpected predicate: %s", quad.P.Value)
	}
}

func TestW3CManifestsOptional(t *testing.T) {
	root := os.Getenv("W3C_TESTS_DIR")
	if root == "" {
		t.Skip("W3C_TESTS_DIR not set; skipping W3C manifest scan")
	}
	paths := []string{
		filepath.Join(root, "turtle"),
		filepath.Join(root, "ntriples"),
	}
	for _, dir := range paths {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read dir %s: %v", dir, err)
		}
		for _, entry := range entries {
			if !strings.HasSuffix(entry.Name(), ".ttl") && !strings.HasSuffix(entry.Name(), ".nt") {
				continue
			}
			path := filepath.Join(dir, entry.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read file %s: %v", path, err)
			}
			format := FormatTurtle
			if strings.HasSuffix(entry.Name(), ".nt") {
				format = FormatNTriples
			}
			if err := Parse(context.Background(), strings.NewReader(string(data)), format, HandlerFunc(func(Quad) error { return nil })); err != nil {
				t.Fatalf("parse error %s: %v", path, err)
			}
		}
	}
}
