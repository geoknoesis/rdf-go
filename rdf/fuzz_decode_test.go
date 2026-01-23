package rdf

import (
	"bytes"
	"io"
	"testing"
)

const (
	fuzzMaxLineBytes      = 8 << 10
	fuzzMaxStatementBytes = 32 << 10
	fuzzMaxJSONLDBytes    = 64 << 10
)

func FuzzDecodeNTriples(f *testing.F) {
	f.Add([]byte(`<http://example.org/s> <http://example.org/p> "v" .`))
	f.Fuzz(func(t *testing.T, data []byte) {
		dec, err := NewReader(bytes.NewReader(data), FormatNTriples,
			OptMaxLineBytes(fuzzMaxLineBytes),
			OptMaxStatementBytes(fuzzMaxStatementBytes))
		if err != nil {
			return
		}
		drainTriples(dec)
	})
}

func FuzzDecodeNQuads(f *testing.F) {
	f.Add([]byte(`<http://example.org/s> <http://example.org/p> "v" <http://example.org/g> .`))
	f.Fuzz(func(t *testing.T, data []byte) {
		dec, err := NewReader(bytes.NewReader(data), FormatNQuads,
			OptMaxLineBytes(fuzzMaxLineBytes),
			OptMaxStatementBytes(fuzzMaxStatementBytes))
		if err != nil {
			return
		}
		drainQuads(dec)
	})
}

func FuzzDecodeTurtle(f *testing.F) {
	f.Add([]byte(`@prefix ex: <http://example.org/> . ex:s ex:p "v" .`))
	f.Fuzz(func(t *testing.T, data []byte) {
		dec, err := NewReader(bytes.NewReader(data), FormatTurtle,
			OptMaxLineBytes(fuzzMaxLineBytes),
			OptMaxStatementBytes(fuzzMaxStatementBytes))
		if err != nil {
			return
		}
		drainTriples(dec)
	})
}

func FuzzDecodeTriG(f *testing.F) {
	f.Add([]byte(`@prefix ex: <http://example.org/> . ex:g { ex:s ex:p ex:o . }`))
	f.Fuzz(func(t *testing.T, data []byte) {
		dec, err := NewReader(bytes.NewReader(data), FormatTriG,
			OptMaxLineBytes(fuzzMaxLineBytes),
			OptMaxStatementBytes(fuzzMaxStatementBytes))
		if err != nil {
			return
		}
		drainQuads(dec)
	})
}

func FuzzDecodeRDFXML(f *testing.F) {
	f.Add([]byte(`<?xml version="1.0"?><rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#"><rdf:Description rdf:about="http://example.org/s"><rdf:type rdf:resource="http://example.org/t"/></rdf:Description></rdf:RDF>`))
	f.Fuzz(func(t *testing.T, data []byte) {
		limited := io.LimitedReader{R: bytes.NewReader(data), N: fuzzMaxStatementBytes}
		dec, err := NewReader(&limited, FormatRDFXML)
		if err != nil {
			return
		}
		drainTriples(dec)
	})
}

func FuzzDecodeJSONLD(f *testing.F) {
	f.Add([]byte(`{"@graph":[{"@id":"http://example.org/s","http://example.org/p":{"@value":"v"}}]}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		// JSON-LD uses unified decoder
		dec, err := NewReader(bytes.NewReader(data), FormatJSONLD)
		if err != nil {
			return
		}
		drainTriples(dec)
	})
}

func drainTriples(dec Reader) {
	for {
		_, err := dec.Next()
		if err != nil {
			break
		}
	}
	_ = dec.Close()
}

func drainQuads(dec Reader) {
	for {
		_, err := dec.Next()
		if err != nil {
			break
		}
	}
	_ = dec.Close()
}
