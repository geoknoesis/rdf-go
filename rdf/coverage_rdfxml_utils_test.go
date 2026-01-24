package rdf

import (
	"encoding/xml"
	"strings"
	"testing"
)

// Test RDF/XML utility functions for additional coverage

func TestFindPrefix_Existing(t *testing.T) {
	dec := &rdfxmltripleDecoder{
		namespaces: map[string]string{
			"ex": "http://example.org/",
		},
	}

	prefix := dec.findPrefix("http://example.org/")
	if prefix != "ex" {
		t.Errorf("findPrefix = %q, want %q", prefix, "ex")
	}
}

func TestFindPrefix_NotFound(t *testing.T) {
	dec := &rdfxmltripleDecoder{
		namespaces: map[string]string{},
	}

	prefix := dec.findPrefix("http://example.org/")
	if prefix != "" {
		t.Errorf("findPrefix = %q, want empty", prefix)
	}
}

func TestResolveQName_WithPrefix(t *testing.T) {
	dec := &rdfxmltripleDecoder{
		namespaces: map[string]string{
			"ex": "http://example.org/",
		},
	}

	result := dec.resolveQName("http://example.org/", "p")
	if result != "http://example.org/p" {
		t.Errorf("resolveQName = %q, want %q", result, "http://example.org/p")
	}
}

func TestResolveQName_NoPrefix(t *testing.T) {
	dec := &rdfxmltripleDecoder{
		namespaces: map[string]string{},
	}

	result := dec.resolveQName("", "p")
	if result != "p" {
		t.Errorf("resolveQName = %q, want %q", result, "p")
	}
}

func TestAttrValue_Found(t *testing.T) {
	dec := &rdfxmltripleDecoder{}
	attrs := []xml.Attr{
		{Name: xml.Name{Space: rdfXMLNS, Local: "about"}, Value: "http://example.org/s"},
	}

	value := dec.attrValue(attrs, rdfXMLNS, "about")
	if value != "http://example.org/s" {
		t.Errorf("attrValue = %q, want %q", value, "http://example.org/s")
	}
}

func TestAttrValue_NotFound(t *testing.T) {
	dec := &rdfxmltripleDecoder{}
	attrs := []xml.Attr{}

	value := dec.attrValue(attrs, rdfXMLNS, "about")
	if value != "" {
		t.Errorf("attrValue = %q, want empty", value)
	}
}

func TestRdfAttrValue_Found(t *testing.T) {
	dec := &rdfxmltripleDecoder{}
	attrs := []xml.Attr{
		{Name: xml.Name{Space: rdfXMLNS, Local: "about"}, Value: "http://example.org/s"},
	}

	value := dec.rdfAttrValue(attrs, "about")
	if value != "http://example.org/s" {
		t.Errorf("rdfAttrValue = %q, want %q", value, "http://example.org/s")
	}
}

func TestRdfAttrValue_NotFound(t *testing.T) {
	dec := &rdfxmltripleDecoder{}
	attrs := []xml.Attr{}

	value := dec.rdfAttrValue(attrs, "about")
	if value != "" {
		t.Errorf("rdfAttrValue = %q, want empty", value)
	}
}

func TestEscapeXML_Simple(t *testing.T) {
	result := escapeXML("test")
	if result != "test" {
		t.Errorf("escapeXML = %q, want %q", result, "test")
	}
}

func TestEscapeXML_SpecialChars(t *testing.T) {
	result := escapeXML("<test&>")
	if !strings.Contains(result, "&lt;") {
		t.Error("escapeXML should escape <")
	}
	if !strings.Contains(result, "&amp;") {
		t.Error("escapeXML should escape &")
	}
	if !strings.Contains(result, "&gt;") {
		t.Error("escapeXML should escape >")
	}
}

func TestEscapeXMLAttr_Simple(t *testing.T) {
	result := escapeXMLAttr("test")
	if result != "test" {
		t.Errorf("escapeXMLAttr = %q, want %q", result, "test")
	}
}

func TestEscapeXMLAttr_Quotes(t *testing.T) {
	result := escapeXMLAttr(`test"value`)
	if !strings.Contains(result, "&quot;") {
		t.Error("escapeXMLAttr should escape quotes")
	}
}
