package rdf

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strings"

	ld "github.com/piprate/json-gold/ld"
)

const jsonLiteralPlaceholderIRI = "urn:json:literal"

// JSONLDOptions configures JSON-LD processing.
type JSONLDOptions struct {
	// Context cancels JSON-LD decoding when done.
	Context context.Context
	// BaseIRI resolves relative IRIs.
	BaseIRI string
	// Base overrides the document base IRI when set (manifest option "base").
	Base string
	// ProcessingMode controls JSON-LD version semantics: "json-ld-1.0" or "json-ld-1.1".
	ProcessingMode string

	// ExpandContext provides an external context for expansion.
	ExpandContext interface{}
	// CompactArrays controls compaction of single-element arrays.
	CompactArrays bool

	// RDF conversion flags.
	UseNativeTypes        bool
	UseRdfType            bool
	ProduceGeneralizedRdf bool

	// Optional RDF direction handling (JSON-LD 1.1).
	RdfDirection string

	// Normative indicates if the test is normative (W3C manifests).
	Normative bool
	// SafeMode toggles strict JSON-LD error handling.
	SafeMode bool

	// Remote document loading.
	DocumentLoader DocumentLoader

	// MaxInputBytes limits the size of JSON-LD input when decoding. Zero means unlimited.
	MaxInputBytes int64
	// MaxNodes limits the number of JSON-LD nodes processed. Zero means unlimited.
	MaxNodes int
	// MaxGraphItems limits the number of items in a @graph array. Zero means unlimited.
	MaxGraphItems int
	// MaxQuads limits the number of emitted quads. Zero means unlimited.
	MaxQuads int
}

// DocumentLoader resolves remote contexts/documents.
type DocumentLoader interface {
	LoadDocument(ctx context.Context, iri string) (RemoteDocument, error)
}

// RemoteDocument represents a fetched JSON-LD document.
type RemoteDocument struct {
	DocumentURL string
	Document    interface{}
	ContextURL  string
	Profile     string
}

// JSONLDProcessor exposes JSON-LD algorithms.
type JSONLDProcessor interface {
	Expand(ctx context.Context, input interface{}, opts JSONLDOptions) (interface{}, error)
	Compact(ctx context.Context, input interface{}, context interface{}, opts JSONLDOptions) (interface{}, error)
	Flatten(ctx context.Context, input interface{}, context interface{}, opts JSONLDOptions) (interface{}, error)
	ToRDF(ctx context.Context, input interface{}, opts JSONLDOptions) ([]Quad, error)
	FromRDF(ctx context.Context, quads []Quad, opts JSONLDOptions) (interface{}, error)
}

type defaultJSONLDProcessor struct{}

// NewJSONLDProcessor returns the default JSON-LD processor.
func NewJSONLDProcessor() JSONLDProcessor {
	return &defaultJSONLDProcessor{}
}

func (p *defaultJSONLDProcessor) Expand(ctx context.Context, input interface{}, opts JSONLDOptions) (interface{}, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	proc := ld.NewJsonLdProcessor()
	goldOpts := newJSONGoldOptions(ctx, opts)
	return proc.Expand(input, goldOpts)
}

func (p *defaultJSONLDProcessor) Compact(ctx context.Context, input interface{}, context interface{}, opts JSONLDOptions) (interface{}, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	proc := ld.NewJsonLdProcessor()
	goldOpts := newJSONGoldOptions(ctx, opts)
	return proc.Compact(input, context, goldOpts)
}

func (p *defaultJSONLDProcessor) Flatten(ctx context.Context, input interface{}, context interface{}, opts JSONLDOptions) (interface{}, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	proc := ld.NewJsonLdProcessor()
	goldOpts := newJSONGoldOptions(ctx, opts)
	return proc.Flatten(input, context, goldOpts)
}

func (p *defaultJSONLDProcessor) ToRDF(ctx context.Context, input interface{}, opts JSONLDOptions) ([]Quad, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	prepared, err := prepareJSONLDForToRDF(ctx, input, opts)
	if err != nil {
		return nil, err
	}
	proc := ld.NewJsonLdProcessor()
	goldOpts := newJSONGoldOptions(ctx, opts)
	result, err := proc.ToRDF(prepared, goldOpts)
	if err != nil {
		return nil, err
	}
	dataset, ok := result.(*ld.RDFDataset)
	if !ok {
		return nil, fmt.Errorf("jsonld: unexpected ToRDF result %T", result)
	}
	if err := canonicalizeJSONLiteralDataset(dataset); err != nil {
		return nil, err
	}
	serializer := &ld.NQuadRDFSerializer{}
	serialized, err := serializer.Serialize(dataset)
	if err != nil {
		return nil, err
	}
	nquads, ok := serialized.(string)
	if !ok {
		return nil, fmt.Errorf("jsonld: unexpected N-Quads result %T", serialized)
	}
	return parseNQuadsString(ctx, nquads)
}

func (p *defaultJSONLDProcessor) FromRDF(ctx context.Context, quads []Quad, opts JSONLDOptions) (interface{}, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	if err := validateJSONLiteralQuads(quads); err != nil {
		return nil, err
	}
	nquads, err := quadsToNQuads(quads)
	if err != nil {
		return nil, err
	}
	proc := ld.NewJsonLdProcessor()
	goldOpts := newJSONGoldOptions(ctx, opts)
	goldOpts.Format = "application/n-quads"
	output, err := proc.FromRDF(nquads, goldOpts)
	if err != nil {
		return nil, err
	}
	normalized, err := normalizeJSONLDJSONLiterals(output)
	if err != nil {
		return nil, err
	}
	return normalized, nil
}

// NewJSONLDTripleDecoder creates a JSON-LD triple decoder with options.
func NewJSONLDTripleDecoder(r io.Reader, opts JSONLDOptions) TripleDecoder {
	return newJSONLDTripleDecoderWithOptions(r, opts)
}

// NewJSONLDQuadDecoder creates a JSON-LD quad decoder with options.
func NewJSONLDQuadDecoder(r io.Reader, opts JSONLDOptions) QuadDecoder {
	return newJSONLDQuadDecoderWithOptions(r, opts)
}

// NewJSONLDTripleEncoder creates a JSON-LD triple encoder with options.
func NewJSONLDTripleEncoder(w io.Writer, opts JSONLDOptions) TripleEncoder {
	return newJSONLDTripleEncoderWithOptions(w, opts)
}

// NewJSONLDQuadEncoder creates a JSON-LD quad encoder with options.
func NewJSONLDQuadEncoder(w io.Writer, opts JSONLDOptions) QuadEncoder {
	return newJSONLDQuadEncoderWithOptions(w, opts)
}

// ParseJSONLDTriples streams JSON-LD triples to a handler.
func ParseJSONLDTriples(ctx context.Context, r io.Reader, opts JSONLDOptions, handler TripleHandler) error {
	decoder := NewJSONLDTripleDecoder(r, opts)
	defer decoder.Close()
	return parseTriplesWithDecoder(ctx, decoder, handler)
}

// ParseJSONLDQuads streams JSON-LD quads to a handler.
func ParseJSONLDQuads(ctx context.Context, r io.Reader, opts JSONLDOptions, handler QuadHandler) error {
	decoder := NewJSONLDQuadDecoder(r, opts)
	defer decoder.Close()
	return parseQuadsWithDecoder(ctx, decoder, handler)
}

type jsonGoldDocumentLoader struct {
	ctx   context.Context
	inner DocumentLoader
}

func (l jsonGoldDocumentLoader) LoadDocument(iri string) (*ld.RemoteDocument, error) {
	if l.inner == nil {
		return ld.NewDefaultDocumentLoader(nil).LoadDocument(iri)
	}
	remote, err := l.inner.LoadDocument(l.ctx, iri)
	if err != nil {
		return nil, err
	}
	return &ld.RemoteDocument{
		DocumentURL: remote.DocumentURL,
		Document:    remote.Document,
		ContextURL:  remote.ContextURL,
	}, nil
}

func newJSONGoldOptions(ctx context.Context, opts JSONLDOptions) *ld.JsonLdOptions {
	goldOpts := ld.NewJsonLdOptions(opts.BaseIRI)
	base := opts.BaseIRI
	if opts.Base != "" {
		base = opts.Base
	}
	if base != "" {
		goldOpts.Base = base
	}
	if opts.ProcessingMode != "" {
		goldOpts.ProcessingMode = opts.ProcessingMode
	}
	if opts.ExpandContext != nil {
		goldOpts.ExpandContext = opts.ExpandContext
	}
	if opts.CompactArrays {
		goldOpts.CompactArrays = opts.CompactArrays
	}
	if opts.UseNativeTypes {
		goldOpts.UseNativeTypes = opts.UseNativeTypes
	}
	if opts.UseRdfType {
		goldOpts.UseRdfType = opts.UseRdfType
	}
	if opts.ProduceGeneralizedRdf {
		goldOpts.ProduceGeneralizedRdf = opts.ProduceGeneralizedRdf
	}
	goldOpts.SafeMode = opts.SafeMode
	if opts.DocumentLoader != nil {
		goldOpts.DocumentLoader = jsonGoldDocumentLoader{ctx: ctx, inner: opts.DocumentLoader}
	}
	return goldOpts
}

func parseNQuadsString(ctx context.Context, nquads string) ([]Quad, error) {
	var quads []Quad
	err := ParseQuads(ctx, strings.NewReader(nquads), QuadFormatNQuads, QuadHandlerFunc(func(q Quad) error {
		quads = append(quads, q)
		return nil
	}))
	return quads, err
}

func quadsToNQuads(quads []Quad) (string, error) {
	var buf bytes.Buffer
	enc, err := NewQuadEncoder(&buf, QuadFormatNQuads)
	if err != nil {
		return "", err
	}
	for _, q := range quads {
		if err := enc.Write(q); err != nil {
			_ = enc.Close()
			return "", err
		}
	}
	if err := enc.Close(); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func toJSONGoldDataset(ctx context.Context, input interface{}, opts JSONLDOptions) (*ld.RDFDataset, error) {
	prepared, err := prepareJSONLDForToRDF(ctx, input, opts)
	if err != nil {
		return nil, err
	}
	proc := ld.NewJsonLdProcessor()
	goldOpts := newJSONGoldOptions(ctx, opts)
	result, err := proc.ToRDF(prepared, goldOpts)
	if err != nil {
		return nil, err
	}
	dataset, ok := result.(*ld.RDFDataset)
	if !ok {
		return nil, fmt.Errorf("jsonld: unexpected ToRDF result %T", result)
	}
	if err := canonicalizeJSONLiteralDataset(dataset); err != nil {
		return nil, err
	}
	return dataset, nil
}

func parseJSONGoldNQuads(input string) (*ld.RDFDataset, error) {
	serializer := &ld.NQuadRDFSerializer{}
	return serializer.Parse(input)
}

func normalizeJSONGoldDataset(dataset *ld.RDFDataset) (string, error) {
	api := ld.NewJsonLdApi()
	opts := ld.NewJsonLdOptions("")
	opts.Format = "application/n-quads"
	opts.Algorithm = ld.AlgorithmURDNA2015
	dedupeJSONGoldDataset(dataset)
	normalizeBlankNodeIDs(dataset)
	normalizeDatasetIRIs(dataset)
	if err := canonicalizeJSONLiteralDataset(dataset); err != nil {
		return "", err
	}
	normalized, err := api.Normalize(dataset, opts)
	if err != nil {
		return "", err
	}
	value, ok := normalized.(string)
	if !ok {
		return "", fmt.Errorf("jsonld: unexpected normalization result %T", normalized)
	}
	return value, nil
}

func normalizeBlankNodeIDs(dataset *ld.RDFDataset) {
	if dataset == nil {
		return
	}
	ids := map[string]struct{}{}
	for _, quads := range dataset.Graphs {
		for _, quad := range quads {
			collectBlankNodeID(ids, quad.Subject)
			collectBlankNodeID(ids, quad.Predicate)
			collectBlankNodeID(ids, quad.Object)
			collectBlankNodeID(ids, quad.Graph)
		}
	}
	if len(ids) == 0 {
		return
	}
	ordered := make([]string, 0, len(ids))
	for id := range ids {
		ordered = append(ordered, id)
	}
	sort.Strings(ordered)
	mapping := map[string]string{}
	for i, id := range ordered {
		mapping[id] = fmt.Sprintf("_:b%d", i)
	}
	for _, quads := range dataset.Graphs {
		for _, quad := range quads {
			quad.Subject = remapBlankNodeID(quad.Subject, mapping)
			quad.Predicate = remapBlankNodeID(quad.Predicate, mapping)
			quad.Object = remapBlankNodeID(quad.Object, mapping)
			quad.Graph = remapBlankNodeID(quad.Graph, mapping)
		}
	}
}

func collectBlankNodeID(ids map[string]struct{}, node ld.Node) {
	if node == nil {
		return
	}
	if bnode, ok := node.(ld.BlankNode); ok {
		ids[bnode.Attribute] = struct{}{}
	}
}

func remapBlankNodeID(node ld.Node, mapping map[string]string) ld.Node {
	bnode, ok := node.(ld.BlankNode)
	if !ok {
		return node
	}
	if mapped, ok := mapping[bnode.Attribute]; ok {
		return ld.NewBlankNode(mapped)
	}
	return node
}

func dedupeJSONGoldDataset(dataset *ld.RDFDataset) {
	if dataset == nil {
		return
	}
	for graphName, quads := range dataset.Graphs {
		seen := map[string]struct{}{}
		out := make([]*ld.Quad, 0, len(quads))
		for _, quad := range quads {
			if quad == nil {
				continue
			}
			key := ldQuadKey(quad, graphName)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, quad)
		}
		dataset.Graphs[graphName] = out
	}
}

func ldQuadKey(quad *ld.Quad, graphName string) string {
	return strings.Join([]string{
		ldNodeKey(quad.Subject),
		ldNodeKey(quad.Predicate),
		ldNodeKey(quad.Object),
		graphName,
	}, "|")
}

func ldNodeKey(node ld.Node) string {
	if node == nil {
		return ""
	}
	if lit, ok := node.(ld.Literal); ok {
		return strings.Join([]string{lit.Value, lit.Datatype, lit.Language}, "::")
	}
	return node.GetValue()
}

func normalizeDatasetIRIs(dataset *ld.RDFDataset) {
	if dataset == nil {
		return
	}
	for _, quads := range dataset.Graphs {
		for _, quad := range quads {
			if quad == nil {
				continue
			}
			quad.Subject = normalizeJSONGoldNodeIRI(quad.Subject)
			quad.Predicate = normalizeJSONGoldNodeIRI(quad.Predicate)
			quad.Object = normalizeJSONGoldNodeIRI(quad.Object)
			quad.Graph = normalizeJSONGoldNodeIRI(quad.Graph)
		}
	}
}

func normalizeJSONGoldNodeIRI(node ld.Node) ld.Node {
	if node == nil {
		return nil
	}
	iri, ok := node.(ld.IRI)
	if !ok {
		return node
	}
	normalized := normalizeIRIPath(iri.Value)
	if normalized == iri.Value {
		return node
	}
	return ld.NewIRI(normalized)
}

func normalizeIRIPath(value string) string {
	parsed, err := url.Parse(value)
	if err != nil {
		return value
	}
	if parsed.Scheme != "" {
		prefix := parsed.Scheme + ":///"
		if strings.HasPrefix(value, prefix) {
			trimmed := strings.TrimPrefix(value, prefix)
			if trimmed != "" {
				return parsed.Scheme + ":" + trimmed
			}
		}
	}
	if parsed.Scheme != "" && parsed.Opaque == "" && parsed.Host == "" && strings.HasPrefix(parsed.Path, "///") {
		trimmed := strings.TrimPrefix(parsed.Path, "///")
		if trimmed != "" {
			return parsed.Scheme + ":" + trimmed
		}
	}
	if parsed.Opaque != "" || parsed.Scheme == "" {
		return value
	}
	if parsed.Path == "" {
		return value
	}
	needsCleanup := strings.Contains(parsed.Path, "/./") || strings.Contains(parsed.Path, "/../") ||
		strings.HasSuffix(parsed.Path, "/.") || strings.HasSuffix(parsed.Path, "/..") ||
		(parsed.Fragment != "" && strings.HasSuffix(parsed.Path, "/") && len(parsed.Path) > 1) ||
		strings.Contains(parsed.Path, "//")
	if !needsCleanup {
		return value
	}
	parsed.Path = removeDotSegments(parsed.Path)
	parsed.Path = collapseSlashes(parsed.Path)
	if parsed.Fragment != "" && strings.HasSuffix(parsed.Path, "/") && len(parsed.Path) > 1 {
		parsed.Path = strings.TrimSuffix(parsed.Path, "/")
	}
	return parsed.String()
}

func removeDotSegments(path string) string {
	var output []string
	segments := strings.Split(path, "/")
	for _, segment := range segments {
		switch segment {
		case ".":
			continue
		case "..":
			if len(output) > 0 {
				output = output[:len(output)-1]
			}
		default:
			output = append(output, segment)
		}
	}
	normalized := strings.Join(output, "/")
	if strings.HasPrefix(path, "/") {
		normalized = "/" + normalized
	}
	if normalized == "" {
		return "/"
	}
	return normalized
}

func collapseSlashes(path string) string {
	for strings.Contains(path, "//") {
		path = strings.ReplaceAll(path, "//", "/")
	}
	return path
}

func canonicalizeJSONLiteralDataset(dataset *ld.RDFDataset) error {
	if dataset == nil {
		return nil
	}
	for _, quads := range dataset.Graphs {
		for _, quad := range quads {
			if quad == nil || quad.Object == nil {
				return fmt.Errorf("jsonld: invalid quad in dataset")
			}
			literal, ok := quad.Object.(ld.Literal)
			if !ok {
				continue
			}
			if literal.Datatype == jsonLiteralPlaceholderIRI {
				literal.Datatype = ld.RDFJSONLiteral
			}
			if literal.Datatype == ld.RDFJSONLiteral {
				canonical, err := canonicalizeJSONLiteralString(literal.Value)
				if err != nil {
					return err
				}
				literal.Value = canonical
				quad.Object = literal
			}
		}
	}
	return nil
}

func canonicalizeJSONLiteralString(raw string) (string, error) {
	normalized, err := canonicalizeJSONText([]byte(raw))
	if err != nil {
		return "", fmt.Errorf("jsonld: invalid JSON literal: %w", err)
	}
	return string(normalized), nil
}

func canonicalizeJSONLiteralValue(value interface{}) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	canonical, err := canonicalizeJSONText(data)
	if err != nil {
		return "", err
	}
	return string(canonical), nil
}

func validateJSONLiteralQuads(quads []Quad) error {
	for _, q := range quads {
		literal, ok := q.O.(Literal)
		if !ok {
			continue
		}
		if literal.Datatype.Value != ld.RDFJSONLiteral {
			continue
		}
		if _, err := canonicalizeJSONLiteralString(literal.Lexical); err != nil {
			return err
		}
	}
	return nil
}

func normalizeJSONLDJSONLiterals(input interface{}) (interface{}, error) {
	switch value := input.(type) {
	case map[string]interface{}:
		if jsonValue, ok := value["@value"]; ok {
			if jsonType, ok := value["@type"]; ok && jsonTypeIncludes(jsonType, ld.RDFJSONLiteral, "@json") {
				parsed, err := parseJSONLiteralValue(jsonValue)
				if err != nil {
					return nil, err
				}
				value["@type"] = "@json"
				value["@value"] = parsed
			}
		}
		for key, item := range value {
			normalized, err := normalizeJSONLDJSONLiterals(item)
			if err != nil {
				return nil, err
			}
			value[key] = normalized
		}
		return value, nil
	case []interface{}:
		for i, item := range value {
			normalized, err := normalizeJSONLDJSONLiterals(item)
			if err != nil {
				return nil, err
			}
			value[i] = normalized
		}
		return value, nil
	default:
		return input, nil
	}
}

func parseJSONLiteralValue(value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case string:
		var parsed interface{}
		if _, err := canonicalizeJSONText([]byte(v)); err != nil {
			return nil, fmt.Errorf("jsonld: invalid JSON literal: %w", err)
		}
		if err := json.Unmarshal([]byte(v), &parsed); err != nil {
			return nil, fmt.Errorf("jsonld: invalid JSON literal: %w", err)
		}
		return parsed, nil
	default:
		return v, nil
	}
}

func prepareJSONLDForToRDF(ctx context.Context, input interface{}, opts JSONLDOptions) (interface{}, error) {
	expanded, err := expandJSONLDInput(ctx, input, opts)
	if err != nil {
		return nil, err
	}
	return replaceJSONLiteralValues(expanded)
}

func expandJSONLDInput(ctx context.Context, input interface{}, opts JSONLDOptions) (interface{}, error) {
	proc := ld.NewJsonLdProcessor()
	goldOpts := newJSONGoldOptions(ctx, opts)
	return proc.Expand(input, goldOpts)
}

func replaceJSONLiteralValues(input interface{}) (interface{}, error) {
	switch value := input.(type) {
	case map[string]interface{}:
		if jsonType, ok := value["@type"]; ok && jsonTypeIncludes(jsonType, "@json", ld.RDFJSONLiteral) {
			if jsonValue, ok := value["@value"]; ok {
				canonical, err := canonicalizeJSONLiteralValue(jsonValue)
				if err != nil {
					return nil, err
				}
				value["@value"] = canonical
				value["@type"] = jsonLiteralPlaceholderIRI
			}
		}
		for key, item := range value {
			prepared, err := replaceJSONLiteralValues(item)
			if err != nil {
				return nil, err
			}
			value[key] = prepared
		}
		return value, nil
	case []interface{}:
		for i, item := range value {
			prepared, err := replaceJSONLiteralValues(item)
			if err != nil {
				return nil, err
			}
			value[i] = prepared
		}
		return value, nil
	default:
		return input, nil
	}
}

func jsonTypeIncludes(raw interface{}, values ...string) bool {
	switch v := raw.(type) {
	case string:
		for _, value := range values {
			if v == value {
				return true
			}
		}
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok {
				for _, value := range values {
					if s == value {
						return true
					}
				}
			}
		}
	}
	return false
}
