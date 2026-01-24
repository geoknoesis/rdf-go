package rdf

import (
	"context"
	"testing"
)

// Test decoder_options.go functions for coverage

func TestDefaultDecodeOptions(t *testing.T) {
	opts := defaultDecodeOptions()

	if opts.MaxLineBytes != DefaultMaxLineBytes {
		t.Errorf("defaultDecodeOptions().MaxLineBytes = %d, want %d", opts.MaxLineBytes, DefaultMaxLineBytes)
	}
	if opts.MaxStatementBytes != DefaultMaxStatementBytes {
		t.Errorf("defaultDecodeOptions().MaxStatementBytes = %d, want %d", opts.MaxStatementBytes, DefaultMaxStatementBytes)
	}
	if opts.MaxDepth != DefaultMaxDepth {
		t.Errorf("defaultDecodeOptions().MaxDepth = %d, want %d", opts.MaxDepth, DefaultMaxDepth)
	}
	if opts.MaxTriples != DefaultMaxTriples {
		t.Errorf("defaultDecodeOptions().MaxTriples = %d, want %d", opts.MaxTriples, DefaultMaxTriples)
	}
	if !opts.ExpandRDFXMLContainers {
		t.Error("defaultDecodeOptions().ExpandRDFXMLContainers should be true")
	}
}

func TestSafeDecodeOptions_Coverage(t *testing.T) {
	opts := safeDecodeOptions()

	expectedMaxLineBytes := 64 << 10
	if opts.MaxLineBytes != expectedMaxLineBytes {
		t.Errorf("safeDecodeOptions().MaxLineBytes = %d, want %d", opts.MaxLineBytes, expectedMaxLineBytes)
	}

	expectedMaxStatementBytes := 256 << 10
	if opts.MaxStatementBytes != expectedMaxStatementBytes {
		t.Errorf("safeDecodeOptions().MaxStatementBytes = %d, want %d", opts.MaxStatementBytes, expectedMaxStatementBytes)
	}

	expectedMaxDepth := 50
	if opts.MaxDepth != expectedMaxDepth {
		t.Errorf("safeDecodeOptions().MaxDepth = %d, want %d", opts.MaxDepth, expectedMaxDepth)
	}

	expectedMaxTriples := int64(1_000_000)
	if opts.MaxTriples != expectedMaxTriples {
		t.Errorf("safeDecodeOptions().MaxTriples = %d, want %d", opts.MaxTriples, expectedMaxTriples)
	}
}

func TestNormalizeDecodeOptions_ZeroValues(t *testing.T) {
	opts := decodeOptions{
		MaxLineBytes:      0,
		MaxStatementBytes: 0,
		MaxDepth:          0,
		MaxTriples:        0,
	}

	normalized := normalizeDecodeOptions(opts)

	if normalized.MaxLineBytes != DefaultMaxLineBytes {
		t.Errorf("normalizeDecodeOptions().MaxLineBytes = %d, want %d", normalized.MaxLineBytes, DefaultMaxLineBytes)
	}
	if normalized.MaxStatementBytes != DefaultMaxStatementBytes {
		t.Errorf("normalizeDecodeOptions().MaxStatementBytes = %d, want %d", normalized.MaxStatementBytes, DefaultMaxStatementBytes)
	}
	if normalized.MaxDepth != DefaultMaxDepth {
		t.Errorf("normalizeDecodeOptions().MaxDepth = %d, want %d", normalized.MaxDepth, DefaultMaxDepth)
	}
	if normalized.MaxTriples != DefaultMaxTriples {
		t.Errorf("normalizeDecodeOptions().MaxTriples = %d, want %d", normalized.MaxTriples, DefaultMaxTriples)
	}
}

func TestNormalizeDecodeOptions_NonZeroValues(t *testing.T) {
	opts := decodeOptions{
		MaxLineBytes:      1000,
		MaxStatementBytes: 2000,
		MaxDepth:          50,
		MaxTriples:        5000,
	}

	normalized := normalizeDecodeOptions(opts)

	if normalized.MaxLineBytes != 1000 {
		t.Errorf("normalizeDecodeOptions().MaxLineBytes = %d, want 1000", normalized.MaxLineBytes)
	}
	if normalized.MaxStatementBytes != 2000 {
		t.Errorf("normalizeDecodeOptions().MaxStatementBytes = %d, want 2000", normalized.MaxStatementBytes)
	}
	if normalized.MaxDepth != 50 {
		t.Errorf("normalizeDecodeOptions().MaxDepth = %d, want 50", normalized.MaxDepth)
	}
	if normalized.MaxTriples != 5000 {
		t.Errorf("normalizeDecodeOptions().MaxTriples = %d, want 5000", normalized.MaxTriples)
	}
}

func TestNormalizeDecodeOptions_ExpandRDFXMLContainers(t *testing.T) {
	// Test that ExpandRDFXMLContainers is preserved when false
	opts := decodeOptions{
		ExpandRDFXMLContainers: false,
	}

	normalized := normalizeDecodeOptions(opts)

	if normalized.ExpandRDFXMLContainers {
		t.Error("normalizeDecodeOptions() should preserve ExpandRDFXMLContainers=false")
	}

	// Test that ExpandRDFXMLContainers is preserved when true
	opts.ExpandRDFXMLContainers = true
	normalized = normalizeDecodeOptions(opts)

	if !normalized.ExpandRDFXMLContainers {
		t.Error("normalizeDecodeOptions() should preserve ExpandRDFXMLContainers=true")
	}
}

func TestDecodeOptions_AllFields(t *testing.T) {
	ctx := context.Background()
	opts := decodeOptions{
		MaxLineBytes:               1000,
		MaxStatementBytes:          2000,
		MaxDepth:                   50,
		MaxTriples:                 5000,
		AllowQuotedTripleStatement: true,
		DebugStatements:            true,
		AllowEnvOverrides:          true,
		Context:                    ctx,
		StrictIRIValidation:        true,
		ExpandRDFXMLContainers:     true,
	}

	// Verify all fields are set
	if opts.MaxLineBytes != 1000 {
		t.Error("MaxLineBytes not set correctly")
	}
	if opts.MaxStatementBytes != 2000 {
		t.Error("MaxStatementBytes not set correctly")
	}
	if opts.MaxDepth != 50 {
		t.Error("MaxDepth not set correctly")
	}
	if opts.MaxTriples != 5000 {
		t.Error("MaxTriples not set correctly")
	}
	if !opts.AllowQuotedTripleStatement {
		t.Error("AllowQuotedTripleStatement not set correctly")
	}
	if !opts.DebugStatements {
		t.Error("DebugStatements not set correctly")
	}
	if !opts.AllowEnvOverrides {
		t.Error("AllowEnvOverrides not set correctly")
	}
	if opts.Context != ctx {
		t.Error("Context not set correctly")
	}
	if !opts.StrictIRIValidation {
		t.Error("StrictIRIValidation not set correctly")
	}
	if !opts.ExpandRDFXMLContainers {
		t.Error("ExpandRDFXMLContainers not set correctly")
	}
}
