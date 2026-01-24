package rdf

import (
	"testing"
)

// Test common utility functions for maximum coverage

func TestResolveIRI_Simple(t *testing.T) {
	base := "http://example.org/"
	relative := "path"

	result := resolveIRI(base, relative)
	expected := "http://example.org/path"
	if result != expected {
		t.Errorf("resolveIRI = %q, want %q", result, expected)
	}
}

func TestResolveIRI_Absolute(t *testing.T) {
	base := "http://example.org/"
	relative := "http://other.org/path"

	result := resolveIRI(base, relative)
	if result != relative {
		t.Errorf("resolveIRI should return absolute IRI unchanged: %q", result)
	}
}

func TestResolveIRI_NoBase(t *testing.T) {
	base := ""
	relative := "http://example.org/path"

	result := resolveIRI(base, relative)
	if result != relative {
		t.Errorf("resolveIRI should return relative unchanged when base is empty: %q", result)
	}
}

func TestResolveIRI_RelativeWithSlash(t *testing.T) {
	base := "http://example.org/"
	relative := "/path"

	result := resolveIRI(base, relative)
	expected := "http://example.org/path"
	if result != expected {
		t.Errorf("resolveIRI = %q, want %q", result, expected)
	}
}

func TestResolveIRI_BaseWithPath(t *testing.T) {
	base := "http://example.org/base/"
	relative := "path"

	result := resolveIRI(base, relative)
	expected := "http://example.org/base/path"
	if result != expected {
		t.Errorf("resolveIRI = %q, want %q", result, expected)
	}
}

func TestResolveIRI_BaseWithoutSlash(t *testing.T) {
	base := "http://example.org/base"
	relative := "path"

	result := resolveIRI(base, relative)
	expected := "http://example.org/path"
	if result != expected {
		t.Errorf("resolveIRI = %q, want %q", result, expected)
	}
}

func TestResolveIRI_RelativeWithDot(t *testing.T) {
	base := "http://example.org/base/"
	relative := "./path"

	result := resolveIRI(base, relative)
	expected := "http://example.org/base/path"
	if result != expected {
		t.Errorf("resolveIRI = %q, want %q", result, expected)
	}
}

func TestResolveIRI_RelativeWithDotDot(t *testing.T) {
	base := "http://example.org/base/sub/"
	relative := "../path"

	result := resolveIRI(base, relative)
	expected := "http://example.org/base/path"
	if result != expected {
		t.Errorf("resolveIRI = %q, want %q", result, expected)
	}
}

func TestGenerateBlankNodeID(t *testing.T) {
	result := generateBlankNodeID(1)
	expected := "b1"
	if result != expected {
		t.Errorf("generateBlankNodeID(1) = %q, want %q", result, expected)
	}

	result = generateBlankNodeID(42)
	expected = "b42"
	if result != expected {
		t.Errorf("generateBlankNodeID(42) = %q, want %q", result, expected)
	}

	result = generateBlankNodeID(0)
	expected = "b0"
	if result != expected {
		t.Errorf("generateBlankNodeID(0) = %q, want %q", result, expected)
	}
}

func TestGenerateBlankNodeID_Large(t *testing.T) {
	result := generateBlankNodeID(999999)
	expected := "b999999"
	if result != expected {
		t.Errorf("generateBlankNodeID(999999) = %q, want %q", result, expected)
	}
}
