package rdf

import (
	"encoding/json"
	"io"
)

// CanonicalizeJSONLD canonicalizes a JSON-LD document using the JSON Canonicalization Scheme (JCS).
// This produces deterministic output by sorting object keys and normalizing JSON structure.
//
// Note: This function requires reading the entire JSON-LD document into memory.
// For streaming JSON-LD encoding, canonicalization is not practical as it would
// require buffering the entire document.
//
// Example usage:
//
//	var buf bytes.Buffer
//	enc, _ := NewWriter(&buf, FormatJSONLD)
//	enc.Write(stmt)
//	enc.Close()
//
//	canonical, err := CanonicalizeJSONLD(buf.Bytes())
//	if err != nil {
//		return err
//	}
//	// Use canonical JSON-LD
func CanonicalizeJSONLD(jsonData []byte) ([]byte, error) {
	// First, parse the JSON to ensure it's valid
	var data interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, err
	}

	// Re-marshal to get a normalized structure
	normalized, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	// Apply JSON Canonicalization Scheme
	canonical, err := canonicalizeJSONText(normalized)
	if err != nil {
		return nil, err
	}

	return canonical, nil
}

// CanonicalizeJSONLDReader reads JSON-LD from a reader and returns canonicalized output.
// This is a convenience function that reads all data before canonicalizing.
func CanonicalizeJSONLDReader(r io.Reader) ([]byte, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return CanonicalizeJSONLD(data)
}

// CanonicalizeJSONLDWriter writes canonicalized JSON-LD to a writer.
// This reads from the reader, canonicalizes, and writes to the writer.
func CanonicalizeJSONLDWriter(w io.Writer, r io.Reader) error {
	canonical, err := CanonicalizeJSONLDReader(r)
	if err != nil {
		return err
	}
	_, err = w.Write(canonical)
	return err
}
