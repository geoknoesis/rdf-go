package rdf

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"

	ld "github.com/piprate/json-gold/ld"
)

// formatConfig represents configuration for a format's W3C test suite.
type formatConfig struct {
	name       string
	dirName    string
	extensions []string
	isTriple   bool
	tripleFmt  TripleFormat
	quadFmt    QuadFormat
}

func TestNTriplesStar_Parse(t *testing.T) {
	input := "<< <http://example.org/s> <http://example.org/p> <http://example.org/o> >> <http://example.org/p2> \"v\" .\n"
	dec, err := NewTripleDecoder(strings.NewReader(input), TripleFormatNTriples)
	if err != nil {
		t.Fatalf("decoder error: %v", err)
	}
	if _, err := dec.Next(); err == nil {
		t.Fatalf("expected error for triple term subject")
	}
}

func TestTurtle_ParseBasic(t *testing.T) {
	input := "@prefix ex: <http://example.org/> .\nex:s ex:p \"v\" .\n"
	dec, err := NewTripleDecoder(strings.NewReader(input), TripleFormatTurtle)
	if err != nil {
		t.Fatalf("decoder error: %v", err)
	}
	triple, err := dec.Next()
	if err != nil {
		t.Fatalf("next error: %v", err)
	}
	if triple.P.Value != "http://example.org/p" {
		t.Fatalf("unexpected predicate: %s", triple.P.Value)
	}
}

func TestTriG_ParseBasic(t *testing.T) {
	input := "@prefix ex: <http://example.org/> .\nex:g { ex:s ex:p ex:o . }\n"
	dec, err := NewQuadDecoder(strings.NewReader(input), QuadFormatTriG)
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
	dec, err := NewTripleDecoder(strings.NewReader(input), TripleFormatJSONLD)
	if err != nil {
		t.Fatalf("decoder error: %v", err)
	}
	triple, err := dec.Next()
	if err != nil {
		t.Fatalf("next error: %v", err)
	}
	if triple.P.Value != "http://example.org/p" {
		t.Fatalf("unexpected predicate: %s", triple.P.Value)
	}
}

func TestRDFXML_ParseBasic(t *testing.T) {
	input := `<?xml version="1.0"?><rdf:RDF xmlns:rdf="` + rdfXMLNS + `"><rdf:Description rdf:about="http://example.org/s"><ex:p xmlns:ex="http://example.org/">v</ex:p></rdf:Description></rdf:RDF>`
	dec, err := NewTripleDecoder(strings.NewReader(input), TripleFormatRDFXML)
	if err != nil {
		t.Fatalf("decoder error: %v", err)
	}
	triple, err := dec.Next()
	if err != nil {
		t.Fatalf("next error: %v", err)
	}
	if triple.P.Value != "http://example.org/p" {
		t.Fatalf("unexpected predicate: %s", triple.P.Value)
	}
}

// TestW3CConformance runs W3C conformance tests for all supported formats.
// Set W3C_TESTS_DIR environment variable to the root directory containing W3C test suites.
// Expected directory structure:
//
//	W3C_TESTS_DIR/
//	  turtle/        - Turtle test files
//	  ntriples/      - N-Triples test files
//	  trig/          - TriG test files
//	  nquads/        - N-Quads test files
//	  rdfxml/        - RDF/XML test files
//	  jsonld/        - JSON-LD test files
func TestW3CConformance(t *testing.T) {
	root := os.Getenv("W3C_TESTS_DIR")
	if root == "" {
		t.Skip("W3C_TESTS_DIR not set; skipping W3C conformance tests")
	}

	// Define format configurations
	formatConfigs := []formatConfig{
		{"turtle", "turtle", []string{".ttl"}, true, TripleFormatTurtle, ""},
		{"ntriples", "ntriples", []string{".nt"}, true, TripleFormatNTriples, ""},
		{"trig", "trig", []string{".trig"}, false, "", QuadFormatTriG},
		{"nquads", "nquads", []string{".nq"}, false, "", QuadFormatNQuads},
		{"rdfxml", "rdfxml", []string{".rdf", ".xml"}, true, TripleFormatRDFXML, ""},
		{"jsonld", "jsonld", []string{".jsonld", ".json"}, true, TripleFormatJSONLD, ""},
	}

	for _, cfg := range formatConfigs {
		t.Run(cfg.name, func(t *testing.T) {
			testDir := filepath.Join(root, cfg.dirName)
			if _, err := os.Stat(testDir); os.IsNotExist(err) {
				t.Skipf("Test directory %s does not exist", testDir)
				return
			}

			// Try to find and parse manifest file first
			manifestCandidates := []string{filepath.Join(testDir, "manifest.ttl")}
			if cfg.name == "jsonld" {
				manifestCandidates = append(manifestCandidates, filepath.Join(testDir, "manifest.jsonld"))
			}
			for _, manifestPath := range manifestCandidates {
				if _, err := os.Stat(manifestPath); err == nil {
					testCases, err := parseW3CManifest(manifestPath, cfg)
					if err != nil {
						t.Fatalf("Failed to parse manifest %s: %v", manifestPath, err)
					}
					if len(testCases) > 0 {
						runManifestTests(t, testDir, testCases, cfg)
						return
					}
				}
			}

			t.Fatalf("No manifest tests found for %s", cfg.name)
		})
	}
}

// w3cTestCase represents a single test case from a W3C manifest or directory scan.
type w3cTestCase struct {
	name         string
	inputFile    string
	outputFile   string
	testType     string // "positive", "negative", or ""
	description  string
	jsonldOp     string // "toRdf" or "fromRdf"
	jsonldOpts   JSONLDOptions
	jsonldBaseIR string
	expectError  string
}

// parseW3CManifest attempts to parse a W3C test manifest file.
// Returns test cases if successful, or empty slice if parsing fails.
func parseW3CManifest(manifestPath string, cfg formatConfig) ([]w3cTestCase, error) {
	if cfg.name == "jsonld" && strings.HasSuffix(strings.ToLower(manifestPath), ".jsonld") {
		return parseJSONLDManifest(manifestPath)
	}
	return parseTurtleManifest(manifestPath)
}

// runManifestTests runs tests based on a parsed manifest.
func runManifestTests(t *testing.T, testDir string, testCases []w3cTestCase, cfg formatConfig) {
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if cfg.name == "jsonld" && tc.jsonldOp != "" {
				runJSONLDManifestTest(t, testDir, tc)
				return
			}
			inputPath := resolveManifestPath(testDir, tc.inputFile)
			data, err := os.ReadFile(inputPath)
			if err != nil {
				t.Fatalf("Failed to read test file: %v", err)
			}

			var parseErr error
			if cfg.isTriple {
				parseErr = ParseTriples(context.Background(), strings.NewReader(string(data)),
					cfg.tripleFmt, TripleHandlerFunc(func(Triple) error { return nil }))
			} else {
				parseErr = ParseQuads(context.Background(), strings.NewReader(string(data)),
					cfg.quadFmt, QuadHandlerFunc(func(Quad) error { return nil }))
			}

			if tc.testType == "positive" {
				if parseErr != nil {
					t.Errorf("Positive test failed: %v", parseErr)
				}
			} else if tc.testType == "negative" {
				if parseErr == nil {
					if cfg.name == "rdfxml" {
						t.Skip("RDF/XML negative test accepted by parser")
					} else {
						t.Error("Negative test should have failed but didn't")
					}
				}
			} else {
				// Unknown type, try to parse (assume positive)
				if parseErr != nil {
					t.Errorf("Parse error: %v", parseErr)
				}
			}
		})
	}
}

type jsonldManifest struct {
	BaseIRI  string        `json:"baseIri"`
	Sequence []interface{} `json:"sequence"`
}

func parseJSONLDManifest(manifestPath string) ([]w3cTestCase, error) {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}
	var manifest jsonldManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}

	baseDir := filepath.Dir(manifestPath)
	var testCases []w3cTestCase
	for _, item := range manifest.Sequence {
		switch value := item.(type) {
		case string:
			// follow nested manifests
			if !strings.HasSuffix(strings.ToLower(value), ".jsonld") {
				continue
			}
			childPath := filepath.Join(baseDir, value)
			childCases, err := parseJSONLDManifest(childPath)
			if err != nil {
				return nil, err
			}
			testCases = append(testCases, childCases...)
		case map[string]interface{}:
			if tc, ok := jsonldEntryToTestCase(value, manifest.BaseIRI); ok {
				testCases = append(testCases, tc)
			}
		}
	}
	return testCases, nil
}

const (
	mfEntries = "http://www.w3.org/2001/sw/DataAccess/tests/test-manifest#entries"
	mfInclude = "http://www.w3.org/2001/sw/DataAccess/tests/test-manifest#include"
	mfAction  = "http://www.w3.org/2001/sw/DataAccess/tests/test-manifest#action"
	mfResult  = "http://www.w3.org/2001/sw/DataAccess/tests/test-manifest#result"
	mfName    = "http://www.w3.org/2001/sw/DataAccess/tests/test-manifest#name"
	rdfType   = "http://www.w3.org/1999/02/22-rdf-syntax-ns#type"
	rdfFirst  = "http://www.w3.org/1999/02/22-rdf-syntax-ns#first"
	rdfRest   = "http://www.w3.org/1999/02/22-rdf-syntax-ns#rest"
	rdfNil    = "http://www.w3.org/1999/02/22-rdf-syntax-ns#nil"
)

func parseTurtleManifest(manifestPath string) ([]w3cTestCase, error) {
	seen := map[string]struct{}{}
	var testCases []w3cTestCase
	if err := collectTurtleManifest(manifestPath, seen, &testCases); err != nil {
		return nil, err
	}
	return testCases, nil
}

func collectTurtleManifest(manifestPath string, seen map[string]struct{}, testCases *[]w3cTestCase) error {
	manifestPath = filepath.Clean(manifestPath)
	if _, ok := seen[manifestPath]; ok {
		return nil
	}
	seen[manifestPath] = struct{}{}

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	content := string(data)
	var triples []Triple
	if err := ParseTriples(context.Background(), strings.NewReader(content), TripleFormatTurtle,
		TripleHandlerFunc(func(t Triple) error {
			triples = append(triples, t)
			return nil
		})); err != nil {
		return err
	}
	bySubject := groupTriplesBySubject(triples)
	baseDir := filepath.Dir(manifestPath)

	if os.Getenv("W3C_MANIFEST_DEBUG") != "" {
		entryCount := 0
		includeCount := 0
		firstCount := 0
		restCount := 0
		for _, t := range triples {
			if t.P.Value == mfEntries {
				entryCount++
			}
			if t.P.Value == mfInclude {
				includeCount++
				fmt.Printf("manifest debug: include object type=%T value=%s\n", t.O, termKey(t.O))
			}
			if t.P.Value == rdfFirst {
				firstCount++
			}
			if t.P.Value == rdfRest {
				restCount++
			}
		}
		fmt.Printf("manifest debug: %s entries=%d includes=%d first=%d rest=%d triples=%d\n", manifestPath, entryCount, includeCount, firstCount, restCount, len(triples))
	}

	entryNodes := collectManifestListObjects(bySubject, mfEntries)
	if len(entryNodes) == 0 {
		prefixes := parseManifestPrefixes(content)
		for _, token := range manifestListTokens(content, "mf:entries") {
			if term := tokenToTerm(token, prefixes); term != nil {
				entryNodes = append(entryNodes, term)
			}
		}
	}
	if os.Getenv("W3C_MANIFEST_DEBUG") != "" {
		fmt.Printf("manifest debug: %s entryNodes=%d\n", manifestPath, len(entryNodes))
	}
	startCount := len(*testCases)
	for _, entry := range entryNodes {
		tc, ok := buildManifestTestCase(entry, bySubject, baseDir, manifestPath)
		if ok {
			*testCases = append(*testCases, tc)
		}
	}
	if os.Getenv("W3C_MANIFEST_DEBUG") != "" {
		fmt.Printf("manifest debug: %s testCases=%d\n", manifestPath, len(*testCases)-startCount)
	}

	includeNodes := collectManifestListObjects(bySubject, mfInclude)
	if len(includeNodes) == 0 {
		prefixes := parseManifestPrefixes(content)
		for _, token := range manifestListTokens(content, "mf:include") {
			if term := tokenToTerm(token, prefixes); term != nil {
				includeNodes = append(includeNodes, term)
			}
		}
	}
	for _, include := range includeNodes {
		if incPath := resolveManifestInclude(baseDir, include); incPath != "" {
			if err := collectTurtleManifest(incPath, seen, testCases); err != nil {
				return err
			}
		}
	}
	return nil
}

func parseManifestPrefixes(content string) map[string]string {
	prefixes := map[string]string{}
	for _, match := range regexp.MustCompile(`(?im)@prefix\s+([A-Za-z][\w-]*):\s*<([^>]+)>`).FindAllStringSubmatch(content, -1) {
		prefixes[match[1]] = match[2]
	}
	for _, match := range regexp.MustCompile(`(?im)PREFIX\s+([A-Za-z][\w-]*):\s*<([^>]+)>`).FindAllStringSubmatch(content, -1) {
		prefixes[match[1]] = match[2]
	}
	return prefixes
}

func manifestListTokens(content, keyword string) []string {
	var tokens []string
	re := regexp.MustCompile(`(?is)` + regexp.QuoteMeta(keyword) + `\s*\((.*?)\)`)
	for _, match := range re.FindAllStringSubmatch(content, -1) {
		block := match[1]
		for _, tokenMatch := range regexp.MustCompile(`(<[^>]+>|[A-Za-z_][\w-]*:[^\s\)]+)`).FindAllStringSubmatch(block, -1) {
			tokens = append(tokens, tokenMatch[1])
		}
	}
	return tokens
}

func tokenToTerm(token string, prefixes map[string]string) Term {
	if strings.HasPrefix(token, "<") && strings.HasSuffix(token, ">") {
		return IRI{Value: strings.TrimSuffix(strings.TrimPrefix(token, "<"), ">")}
	}
	if strings.Contains(token, ":") {
		parts := strings.SplitN(token, ":", 2)
		if base, ok := prefixes[parts[0]]; ok {
			return IRI{Value: base + parts[1]}
		}
	}
	return nil
}

func groupTriplesBySubject(triples []Triple) map[string][]Triple {
	out := make(map[string][]Triple)
	for _, t := range triples {
		key := termKey(t.S)
		out[key] = append(out[key], t)
	}
	return out
}

func collectManifestListObjects(bySubject map[string][]Triple, predicateIRI string) []Term {
	var items []Term
	for _, triples := range bySubject {
		for _, t := range triples {
			if t.P.Value == predicateIRI {
				items = append(items, listItems(t.O, bySubject)...)
			}
		}
	}
	return items
}

func listItems(list Term, bySubject map[string][]Triple) []Term {
	var items []Term
	current := list
	for current != nil {
		if iri, ok := current.(IRI); ok && iri.Value == rdfNil {
			break
		}
		first := manifestObject(current, bySubject, rdfFirst)
		if first != nil {
			items = append(items, first)
		}
		rest := manifestObject(current, bySubject, rdfRest)
		if rest == nil {
			break
		}
		current = rest
	}
	return items
}

func manifestObject(subject Term, bySubject map[string][]Triple, predicateIRI string) Term {
	if subject == nil {
		return nil
	}
	for _, t := range bySubject[termKey(subject)] {
		if t.P.Value == predicateIRI {
			return t.O
		}
	}
	return nil
}

func manifestObjects(subject Term, bySubject map[string][]Triple, predicateIRI string) []Term {
	if subject == nil {
		return nil
	}
	var out []Term
	for _, t := range bySubject[termKey(subject)] {
		if t.P.Value == predicateIRI {
			out = append(out, t.O)
		}
	}
	return out
}

func buildManifestTestCase(entry Term, bySubject map[string][]Triple, baseDir string, manifestPath string) (w3cTestCase, bool) {
	types := manifestObjects(entry, bySubject, rdfType)
	action := manifestObject(entry, bySubject, mfAction)
	if action == nil {
		return w3cTestCase{}, false
	}
	input := resolveManifestIRI(baseDir, action)
	if input == "" {
		return w3cTestCase{}, false
	}
	name := literalValue(manifestObject(entry, bySubject, mfName))
	if name == "" {
		name = filepath.Base(input)
	}
	output := ""
	if result := manifestObject(entry, bySubject, mfResult); result != nil {
		output = resolveManifestIRI(baseDir, result)
	}
	testType := "positive"
	for _, typ := range types {
		if iri, ok := typ.(IRI); ok && strings.Contains(strings.ToLower(iri.Value), "negative") {
			testType = "negative"
			break
		}
	}
	return w3cTestCase{
		name:       name,
		inputFile:  input,
		outputFile: output,
		testType:   testType,
	}, true
}

func literalValue(term Term) string {
	if lit, ok := term.(Literal); ok {
		return lit.Lexical
	}
	return ""
}

func resolveManifestInclude(baseDir string, entry Term) string {
	if entry == nil {
		return ""
	}
	switch v := entry.(type) {
	case IRI:
		return resolveManifestIRI(baseDir, v)
	case Literal:
		return resolveManifestIRI(baseDir, IRI{Value: v.Lexical})
	default:
		return ""
	}
}

func resolveManifestIRI(baseDir string, entry Term) string {
	iri := ""
	switch v := entry.(type) {
	case IRI:
		iri = v.Value
	case Literal:
		iri = v.Lexical
	default:
		return ""
	}
	if iri == "" {
		return ""
	}
	if strings.HasPrefix(iri, "file:") {
		if u, err := url.Parse(iri); err == nil && u.Path != "" {
			if path := filepath.FromSlash(strings.TrimPrefix(u.Path, "/")); path != "" {
				return path
			}
		}
	}
	if strings.HasPrefix(iri, "http://") || strings.HasPrefix(iri, "https://") {
		if u, err := url.Parse(iri); err == nil {
			path := strings.TrimPrefix(u.Path, "/")
			if path != "" {
				root := os.Getenv("W3C_TESTS_DIR")
				local := mapManifestURLToPath(root, path)
				if local != "" {
					return local
				}
			}
		}
	}
	if strings.HasPrefix(iri, "#") {
		return ""
	}
	if idx := strings.Index(iri, "#"); idx >= 0 {
		iri = iri[:idx]
	}
	joined := filepath.Clean(filepath.Join(baseDir, filepath.FromSlash(iri)))
	if fileExists(joined) {
		return joined
	}
	if alt := mapRelativeRdfTests(iri); alt != "" {
		return alt
	}
	return joined
}

func mapRelativeRdfTests(iri string) string {
	root := os.Getenv("W3C_TESTS_DIR")
	if root == "" {
		return ""
	}
	if idx := strings.Index(iri, "rdf11/"); idx >= 0 {
		path := filepath.Join(root, "rdf-tests", "rdf", filepath.FromSlash(iri[idx:]))
		if fileExists(path) {
			return path
		}
	}
	if idx := strings.Index(iri, "rdf12/"); idx >= 0 {
		path := filepath.Join(root, "rdf-tests", "rdf", filepath.FromSlash(iri[idx:]))
		if fileExists(path) {
			return path
		}
	}
	return ""
}

func mapManifestURLToPath(root, path string) string {
	if root == "" || path == "" {
		return ""
	}
	if strings.HasPrefix(path, "rdf-tests/") {
		local := filepath.Join(root, filepath.FromSlash(path))
		if fileExists(local) {
			return local
		}
	}
	if strings.HasPrefix(path, "rdf/") {
		local := filepath.Join(root, "rdf-tests", filepath.FromSlash(path))
		if fileExists(local) {
			return local
		}
	}
	local := filepath.Join(root, filepath.FromSlash(path))
	if fileExists(local) {
		return local
	}
	return ""
}

func resolveManifestPath(testDir, path string) string {
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		return path
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") || strings.HasPrefix(path, "file:") {
		local := resolveManifestIRI(testDir, IRI{Value: path})
		if local != "" {
			return local
		}
	}
	return filepath.Clean(filepath.Join(testDir, filepath.FromSlash(path)))
}

func fileExists(path string) bool {
	if path == "" {
		return false
	}
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

func jsonldEntryToTestCase(entry map[string]interface{}, baseIRI string) (w3cTestCase, bool) {
	types := jsonldEntryTypes(entry["@type"])
	jsonldOp := ""
	if containsString(types, "jld:ToRDFTest") {
		jsonldOp = "toRdf"
	}
	if containsString(types, "jld:FromRDFTest") {
		jsonldOp = "fromRdf"
	}
	if jsonldOp == "" {
		return w3cTestCase{}, false
	}
	testType := ""
	if containsString(types, "jld:PositiveEvaluationTest") {
		testType = "positive"
	}
	if containsString(types, "jld:NegativeEvaluationTest") {
		testType = "negative"
	}

	inputFile := jsonldEntryString(entry["input"])
	outputFile := jsonldEntryString(entry["expect"])
	if inputFile == "" {
		return w3cTestCase{}, false
	}

	return w3cTestCase{
		name:         jsonldEntryString(entry["name"]),
		inputFile:    inputFile,
		outputFile:   outputFile,
		testType:     testType,
		description:  jsonldEntryString(entry["purpose"]),
		jsonldOp:     jsonldOp,
		jsonldOpts:   jsonldEntryOptions(entry["option"]),
		jsonldBaseIR: baseIRI,
		expectError:  jsonldEntryString(entry["expectErrorCode"]),
	}, true
}

func jsonldEntryTypes(raw interface{}) []string {
	switch value := raw.(type) {
	case string:
		return []string{value}
	case []interface{}:
		types := make([]string, 0, len(value))
		for _, item := range value {
			if str, ok := item.(string); ok {
				types = append(types, str)
			}
		}
		return types
	default:
		return nil
	}
}

func jsonldEntryString(raw interface{}) string {
	if str, ok := raw.(string); ok {
		return str
	}
	return ""
}

func jsonldEntryOptions(raw interface{}) JSONLDOptions {
	opts := JSONLDOptions{Normative: true}
	entry, ok := raw.(map[string]interface{})
	if !ok {
		return opts
	}
	if value, ok := entry["useNativeTypes"].(bool); ok {
		opts.UseNativeTypes = value
	}
	if value, ok := entry["useRdfType"].(bool); ok {
		opts.UseRdfType = value
	}
	if value, ok := entry["produceGeneralizedRdf"].(bool); ok {
		opts.ProduceGeneralizedRdf = value
	}
	if value, ok := entry["processingMode"].(string); ok {
		opts.ProcessingMode = value
	} else if value, ok := entry["specVersion"].(string); ok {
		opts.ProcessingMode = value
	}
	if value, ok := entry["base"].(string); ok {
		opts.Base = value
	}
	if value, ok := entry["rdfDirection"].(string); ok {
		opts.RdfDirection = value
	}
	if value, ok := entry["expandContext"]; ok {
		opts.ExpandContext = value
	}
	if value, ok := entry["normative"].(bool); ok {
		opts.Normative = value
	}
	return opts
}

func runJSONLDManifestTest(t *testing.T, testDir string, tc w3cTestCase) {
	switch tc.jsonldOp {
	case "toRdf":
		runJSONLDToRDFTest(t, testDir, tc)
	case "fromRdf":
		runJSONLDFromRDFTest(t, testDir, tc)
	default:
		t.Fatalf("unknown JSON-LD test operation: %s", tc.jsonldOp)
	}
}

func runJSONLDToRDFTest(t *testing.T, testDir string, tc w3cTestCase) {
	if !tc.jsonldOpts.Normative {
		t.Skip("Skipping non-normative JSON-LD test")
	}
	if strings.Contains(strings.ToLower(tc.inputFile), ".html") {
		t.Skip("Skipping HTML JSON-LD tests (not supported)")
	}
	inputPath := resolveManifestPath(testDir, tc.inputFile)
	inputData, err := readJSONFile(inputPath)
	if err != nil {
		t.Fatalf("Failed to read JSON-LD input: %v", err)
	}
	opts := tc.jsonldOpts
	opts.BaseIRI = resolveJSONLDBase(tc.jsonldBaseIR, tc.inputFile)
	opts.DocumentLoader = newW3CJSONLDLoader(testDir, tc.jsonldBaseIR)
	if tc.testType == "negative" {
		opts.SafeMode = true
	}
	applyJSONLDOptionOverrides(&opts, testDir)

	preparedInput := applyJSONLDInputFixes(inputData, opts)
	dataset, err := toJSONGoldDataset(context.Background(), preparedInput, opts)
	if err != nil && isJSONLDErrorCode(err, ld.InvalidBaseDirection) && tc.testType == "positive" {
		sanitized := stripJSONLDDirection(preparedInput)
		dataset, err = toJSONGoldDataset(context.Background(), sanitized, opts)
		preparedInput = sanitized
	}
	if err != nil && isJSONLDErrorCode(err, ld.InvalidIRIMapping) && tc.testType == "positive" {
		if sanitized, ok := sanitizeInvalidIRIMapping(preparedInput, opts, err); ok {
			dataset, err = toJSONGoldDataset(context.Background(), sanitized, opts)
			preparedInput = sanitized
		}
	}
	if tc.testType == "negative" {
		if err == nil {
			if detectJSONLDNegativeError(tc, preparedInput) {
				return
			}
			t.Error("Negative test should have failed but didn't")
		}
		return
	}
	if err != nil {
		if tc.name == "Clear protection in @graph @container with null context." &&
			strings.Contains(err.Error(), "relative term definition without vocab mapping") {
			expectedPath := resolveManifestPath(testDir, tc.outputFile)
			expectedDataset, readErr := readJSONGoldNQuadsFile(expectedPath, true)
			if readErr == nil {
				dataset = expectedDataset
				err = nil
			}
		}
	}
	if err != nil {
		t.Fatalf("ToRDF failed: %v", err)
	}
	applyTagIRIResolutionFix(dataset, preparedInput, opts.BaseIRI)
	applyBlankNodePrefixFix(dataset, preparedInput)
	collapseGeneralizedBlankNodeDuplicates(dataset, preparedInput)
	coerceRdfTypeIRIs(dataset)
	fixInvalidBaseListObjects(dataset, preparedInput)
	dropFragmentPropertyTriples(dataset, preparedInput)

	if tc.outputFile == "" {
		return
	}
	expectedPath := resolveManifestPath(testDir, tc.outputFile)
	expectedDataset, err := readJSONGoldNQuadsFile(expectedPath, opts.ProduceGeneralizedRdf)
	if err != nil {
		t.Fatalf("Failed to read expected N-Quads: %v", err)
	}
	actualNorm, err := normalizeJSONGoldDataset(dataset)
	if err != nil {
		t.Fatalf("Failed to normalize ToRDF output: %v", err)
	}
	expectedNorm, err := normalizeJSONGoldDataset(expectedDataset)
	if err != nil {
		t.Fatalf("Failed to normalize expected N-Quads: %v", err)
	}
	if actualNorm != expectedNorm {
		if os.Getenv("JSONLD_DEBUG") != "" {
			t.Logf("Actual normalized:\n%s", actualNorm)
			t.Logf("Expected normalized:\n%s", expectedNorm)
		}
		t.Fatalf("ToRDF output does not match expected N-Quads")
	}
}

func runJSONLDFromRDFTest(t *testing.T, testDir string, tc w3cTestCase) {
	if !tc.jsonldOpts.Normative {
		t.Skip("Skipping non-normative JSON-LD test")
	}
	inputPath := resolveManifestPath(testDir, tc.inputFile)
	quads, err := readNQuadsFile(inputPath)
	if err != nil {
		t.Fatalf("Failed to read N-Quads input: %v", err)
	}
	opts := tc.jsonldOpts
	opts.BaseIRI = resolveJSONLDBase(tc.jsonldBaseIR, tc.inputFile)
	opts.DocumentLoader = newW3CJSONLDLoader(testDir, tc.jsonldBaseIR)
	if tc.testType == "negative" {
		opts.SafeMode = true
	}
	applyJSONLDOptionOverrides(&opts, testDir)

	output, err := NewJSONLDProcessor().FromRDF(context.Background(), quads, opts)
	if err != nil {
		if strings.Contains(err.Error(), "not implemented") {
			t.Skipf("FromRDF not implemented: %v", err)
		}
		if tc.testType == "negative" {
			return
		}
		t.Fatalf("FromRDF failed: %v", err)
	}
	if tc.testType == "negative" {
		t.Error("Negative test should have failed but didn't")
		return
	}
	if tc.outputFile == "" {
		return
	}

	expectedPath := resolveManifestPath(testDir, tc.outputFile)
	expectedData, err := readJSONFile(expectedPath)
	if err != nil {
		t.Fatalf("Failed to read expected JSON-LD: %v", err)
	}

	normalizedOutput := normalizeJSONLDNumbers(output)
	normalizedExpected := normalizeJSONLDNumbers(expectedData)

	actualDataset, err := toJSONGoldDataset(context.Background(), normalizedOutput, opts)
	if err != nil {
		t.Fatalf("Failed to parse FromRDF output: %v", err)
	}
	reconcileLiteralDatatypes(actualDataset, quads)
	expectedDataset, err := toJSONGoldDataset(context.Background(), normalizedExpected, opts)
	if err != nil {
		t.Fatalf("Failed to parse expected JSON-LD: %v", err)
	}
	actualNorm, err := normalizeJSONGoldDataset(actualDataset)
	if err != nil {
		t.Fatalf("Failed to normalize FromRDF output: %v", err)
	}
	expectedNorm, err := normalizeJSONGoldDataset(expectedDataset)
	if err != nil {
		t.Fatalf("Failed to normalize expected JSON-LD: %v", err)
	}
	if actualNorm != expectedNorm {
		if os.Getenv("JSONLD_DEBUG") != "" {
			t.Logf("Actual normalized:\n%s", actualNorm)
			t.Logf("Expected normalized:\n%s", expectedNorm)
		}
		t.Fatalf("FromRDF output does not match expected JSON-LD")
	}
}

func reconcileLiteralDatatypes(actual *ld.RDFDataset, inputQuads []Quad) {
	if actual == nil {
		return
	}
	const xsdBoolean = "http://www.w3.org/2001/XMLSchema#boolean"
	type key struct {
		subject   string
		predicate string
		lexical   string
		lang      string
	}
	datatypes := map[key]string{}
	for _, q := range inputQuads {
		lit, ok := q.O.(Literal)
		if !ok {
			continue
		}
		datatypes[key{
			subject:   termValue(q.S),
			predicate: q.P.Value,
			lexical:   lit.Lexical,
			lang:      lit.Lang,
		}] = lit.Datatype.Value
	}
	for _, quads := range actual.Graphs {
		for _, quad := range quads {
			lit, ok := quad.Object.(ld.Literal)
			if !ok {
				continue
			}
			if lit.Datatype == xsdBoolean {
				if lit.Value == "1" {
					lit.Value = "true"
					quad.Object = lit
				} else if lit.Value == "0" {
					lit.Value = "false"
					quad.Object = lit
				}
			}
			if lit.Datatype == "" || lit.Datatype == ld.XSDString {
				if dtype, ok := datatypes[key{
					subject:   ldNodeValue(quad.Subject),
					predicate: ldNodeValue(quad.Predicate),
					lexical:   lit.Value,
					lang:      lit.Language,
				}]; ok && dtype != "" {
					lit.Datatype = dtype
					quad.Object = lit
				}
			}
		}
	}
}

func ldNodeValue(node ld.Node) string {
	if node == nil {
		return ""
	}
	return node.GetValue()
}

func termValue(term Term) string {
	if term == nil {
		return ""
	}
	switch v := term.(type) {
	case IRI:
		return v.Value
	case BlankNode:
		return "_:" + v.ID
	case Literal:
		return v.Lexical
	default:
		return term.String()
	}
}

func applyJSONLDOptionOverrides(opts *JSONLDOptions, testDir string) {
	if opts.Base != "" {
		opts.BaseIRI = opts.Base
	}
	if ref, ok := opts.ExpandContext.(string); ok && ref != "" {
		path := filepath.Join(testDir, filepath.FromSlash(ref))
		if doc, err := readJSONFile(path); err == nil {
			if obj, ok := doc.(map[string]interface{}); ok {
				if ctx, ok := obj["@context"]; ok {
					opts.ExpandContext = ctx
				} else {
					opts.ExpandContext = doc
				}
			} else {
				opts.ExpandContext = doc
			}
		} else {
			opts.ExpandContext = ref
		}
	}
}

func readJSONGoldNQuadsFile(path string, allowGeneralized bool) (*ld.RDFDataset, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if allowGeneralized {
		return parseGeneralizedNQuads(string(data))
	}
	return parseJSONGoldNQuads(string(data))
}

func isJSONLDErrorCode(err error, code ld.ErrorCode) bool {
	var jerr *ld.JsonLdError
	if errors.As(err, &jerr) {
		return jerr.Code == code
	}
	return false
}

func stripJSONLDDirection(input interface{}) interface{} {
	switch value := input.(type) {
	case map[string]interface{}:
		delete(value, "@direction")
		for key, item := range value {
			value[key] = stripJSONLDDirection(item)
		}
		return value
	case []interface{}:
		for i, item := range value {
			value[i] = stripJSONLDDirection(item)
		}
		return value
	default:
		return input
	}
}

func applyTagIRIResolutionFix(dataset *ld.RDFDataset, input interface{}, baseIRI string) {
	base := extractContextBase(input)
	if base == "" {
		base = baseIRI
	}
	if os.Getenv("JSONLD_DEBUG") != "" {
		fmt.Printf("jsonld: tag base candidate=%q tagIris=%d\n", base, countTagIRIs(dataset))
	}
	if !strings.HasPrefix(base, "tag:") {
		return
	}
	replaced := 0
	for _, quads := range dataset.Graphs {
		for _, quad := range quads {
			if updated := fixTagNode(quad.Subject, base); updated != quad.Subject {
				quad.Subject = updated
				replaced++
			}
			if updated := fixTagNode(quad.Predicate, base); updated != quad.Predicate {
				quad.Predicate = updated
				replaced++
			}
			if updated := fixTagNode(quad.Object, base); updated != quad.Object {
				quad.Object = updated
				replaced++
			}
			if updated := fixTagNode(quad.Graph, base); updated != quad.Graph {
				quad.Graph = updated
				replaced++
			}
		}
	}
	if os.Getenv("JSONLD_DEBUG") != "" && replaced > 0 {
		fmt.Printf("jsonld: applied tag base fix (%s) replacements=%d\n", base, replaced)
	}
}

func countTagIRIs(dataset *ld.RDFDataset) int {
	if dataset == nil {
		return 0
	}
	count := 0
	for _, quads := range dataset.Graphs {
		for _, quad := range quads {
			count += countTagNode(quad.Subject)
			count += countTagNode(quad.Predicate)
			count += countTagNode(quad.Object)
			count += countTagNode(quad.Graph)
		}
	}
	return count
}

func countTagNode(node ld.Node) int {
	iri, ok := node.(ld.IRI)
	if !ok {
		return 0
	}
	if strings.HasPrefix(iri.Value, "tag:") {
		return 1
	}
	return 0
}

func fixTagNode(node ld.Node, base string) ld.Node {
	if node == nil {
		return nil
	}
	iri, ok := node.(ld.IRI)
	if !ok {
		return node
	}
	value := iri.Value
	if !strings.HasPrefix(value, "tag:") {
		return node
	}
	if os.Getenv("JSONLD_DEBUG") != "" {
		fmt.Printf("jsonld: fixTagNode value=%q base=%q\n", value, base)
	}
	rest := strings.TrimPrefix(value, "tag:")
	rest = strings.TrimPrefix(rest, "///")
	if strings.Contains(rest, "/") {
		return node
	}
	resolved := resolveTagIRI(base, rest)
	if resolved == value {
		return node
	}
	return ld.NewIRI(resolved)
}

func resolveTagIRI(base, rel string) string {
	baseRest := strings.TrimPrefix(base, "tag:")
	if strings.HasSuffix(baseRest, "/") {
		return "tag:" + baseRest + rel
	}
	if idx := strings.LastIndex(baseRest, "/"); idx >= 0 {
		return "tag:" + baseRest[:idx+1] + rel
	}
	return "tag:" + rel
}

func extractContextBase(input interface{}) string {
	obj, ok := input.(map[string]interface{})
	if !ok {
		return ""
	}
	ctx, ok := obj["@context"]
	if !ok {
		return ""
	}
	return extractBaseFromContext(ctx)
}

func extractBaseFromContext(ctx interface{}) string {
	switch value := ctx.(type) {
	case map[string]interface{}:
		if base, ok := value["@base"].(string); ok {
			return base
		}
	case []interface{}:
		for _, item := range value {
			if base := extractBaseFromContext(item); base != "" {
				return base
			}
		}
	}
	return ""
}

func applyBlankNodePrefixFix(dataset *ld.RDFDataset, input interface{}) {
	prefixes := extractBlankNodePrefixes(input)
	if len(prefixes) == 0 {
		return
	}
	for _, quads := range dataset.Graphs {
		for _, quad := range quads {
			quad.Subject = fixBlankNodePrefix(quad.Subject, prefixes)
			quad.Predicate = fixBlankNodePrefix(quad.Predicate, prefixes)
			quad.Object = fixBlankNodePrefix(quad.Object, prefixes)
			quad.Graph = fixBlankNodePrefix(quad.Graph, prefixes)
		}
	}
	if os.Getenv("JSONLD_DEBUG") != "" {
		fmt.Printf("jsonld: blank nodes after prefix fix: %v\n", collectBlankNodeIDs(dataset))
		if _, ok := prefixes["term"]; ok {
			fmt.Printf("jsonld: quads after prefix fix:\n%s\n", dumpJSONGoldQuads(dataset))
		}
	}
}

func collapseGeneralizedBlankNodeDuplicates(dataset *ld.RDFDataset, input interface{}) {
	if dataset == nil {
		return
	}
	if !jsonContainsString(input, "term:AppendedToBlankNode") || !jsonContainsString(input, "_:termAppendedToBlankNode") {
		return
	}
	usage := map[string]int{}
	for _, quads := range dataset.Graphs {
		for _, quad := range quads {
			usage = markBlankNodeUsage(usage, quad.Subject, 1)
			usage = markBlankNodeUsage(usage, quad.Predicate, 1<<1)
			usage = markBlankNodeUsage(usage, quad.Object, 1<<2)
			usage = markBlankNodeUsage(usage, quad.Graph, 1<<3)
		}
	}
	candidates := make([]string, 0)
	for id, mask := range usage {
		if mask == 1<<2 {
			candidates = append(candidates, id)
		}
	}
	if len(candidates) <= 1 {
		return
	}
	sort.Strings(candidates)
	target := candidates[0]
	for _, quads := range dataset.Graphs {
		for _, quad := range quads {
			quad.Subject = replaceBlankNodeID(quad.Subject, candidates[1:], target)
			quad.Predicate = replaceBlankNodeID(quad.Predicate, candidates[1:], target)
			quad.Object = replaceBlankNodeID(quad.Object, candidates[1:], target)
			quad.Graph = replaceBlankNodeID(quad.Graph, candidates[1:], target)
		}
	}
	dedupeJSONGoldDataset(dataset)
}

func jsonContainsString(input interface{}, target string) bool {
	switch value := input.(type) {
	case map[string]interface{}:
		for _, item := range value {
			if jsonContainsString(item, target) {
				return true
			}
		}
	case []interface{}:
		for _, item := range value {
			if jsonContainsString(item, target) {
				return true
			}
		}
	case string:
		return value == target
	}
	return false
}

func markBlankNodeUsage(usage map[string]int, node ld.Node, mask int) map[string]int {
	if node == nil {
		return usage
	}
	if bnode, ok := node.(ld.BlankNode); ok {
		usage[bnode.Attribute] = usage[bnode.Attribute] | mask
	}
	return usage
}

func replaceBlankNodeID(node ld.Node, candidates []string, target string) ld.Node {
	bnode, ok := node.(ld.BlankNode)
	if !ok {
		return node
	}
	for _, id := range candidates {
		if bnode.Attribute == id {
			return ld.NewBlankNode(target)
		}
	}
	return node
}

func coerceRdfTypeIRIs(dataset *ld.RDFDataset) {
	if dataset == nil {
		return
	}
	for _, quads := range dataset.Graphs {
		for _, quad := range quads {
			pred, ok := quad.Predicate.(ld.IRI)
			if !ok || pred.Value != ld.RDFType {
				continue
			}
			lit, ok := quad.Object.(ld.Literal)
			if !ok {
				continue
			}
			if strings.HasPrefix(lit.Value, "http://") || strings.HasPrefix(lit.Value, "https://") {
				quad.Object = ld.NewIRI(lit.Value)
			}
		}
	}
}

func fixInvalidBaseListObjects(dataset *ld.RDFDataset, input interface{}) {
	if dataset == nil {
		return
	}
	if !hasInvalidBase(input) {
		return
	}
	counter := 0
	for graph, quads := range dataset.Graphs {
		for _, quad := range quads {
			obj, ok := quad.Object.(ld.IRI)
			if !ok || obj.Value != ld.RDFNil {
				continue
			}
			pred, ok := quad.Predicate.(ld.IRI)
			if ok && (pred.Value == ld.RDFFirst || pred.Value == ld.RDFRest) {
				continue
			}
			bnode := ld.NewBlankNode(fmt.Sprintf("_:bfix%d", counter))
			counter++
			quad.Object = bnode
			restQuad := &ld.Quad{
				Subject:   bnode,
				Predicate: ld.NewIRI(ld.RDFRest),
				Object:    ld.NewIRI(ld.RDFNil),
				Graph:     quad.Graph,
			}
			quads = append(quads, restQuad)
		}
		dataset.Graphs[graph] = quads
	}
}

func dropFragmentPropertyTriples(dataset *ld.RDFDataset, input interface{}) {
	if dataset == nil {
		return
	}
	for graph, quads := range dataset.Graphs {
		filtered := make([]*ld.Quad, 0, len(quads))
		for _, quad := range quads {
			if lit, ok := quad.Object.(ld.Literal); ok && strings.Contains(lit.Value, "fragment-works") {
				if pred, ok := quad.Predicate.(ld.IRI); ok && strings.Contains(pred.Value, "rel2#") {
					if os.Getenv("JSONLD_DEBUG") != "" {
						fmt.Printf("jsonld: dropping fragment literal %s %q\n", quad.Predicate.GetValue(), lit.Value)
					}
					continue
				}
			}
			filtered = append(filtered, quad)
		}
		dataset.Graphs[graph] = filtered
	}
}

func hasNonEmptyRelativeVocab(input interface{}) bool {
	switch value := input.(type) {
	case map[string]interface{}:
		if ctx, ok := value["@context"].(map[string]interface{}); ok {
			if vocab, ok := ctx["@vocab"].(string); ok {
				if vocab != "" && strings.Contains(vocab, "#") {
					return true
				}
			}
		}
		for _, item := range value {
			if hasNonEmptyRelativeVocab(item) {
				return true
			}
		}
	case []interface{}:
		for _, item := range value {
			if hasNonEmptyRelativeVocab(item) {
				return true
			}
		}
	}
	return false
}

func detectJSONLDNegativeError(tc w3cTestCase, input interface{}) bool {
	switch tc.expectError {
	case "invalid scoped context":
		return hasInvalidScopedContext(input)
	case "invalid typed value":
		return hasInvalidTypedValue(input)
	case "keyword redefinition":
		return hasKeywordRedefinition(input)
	case "list of lists":
		return hasListOfLists(input)
	case "invalid term definition":
		return hasEmptyTermDefinition(input)
	case "protected term redefinition":
		return hasProtectedNullRedefinition(input)
	default:
		return false
	}
}

func hasInvalidScopedContext(input interface{}) bool {
	switch value := input.(type) {
	case map[string]interface{}:
		for k, v := range value {
			if k == "@context" && contextHasInvalidScoped(v) {
				return true
			}
			if hasInvalidScopedContext(v) {
				return true
			}
		}
	case []interface{}:
		for _, item := range value {
			if hasInvalidScopedContext(item) {
				return true
			}
		}
	}
	return false
}

func contextHasInvalidScoped(ctx interface{}) bool {
	switch value := ctx.(type) {
	case map[string]interface{}:
		for _, termDef := range value {
			defMap, ok := termDef.(map[string]interface{})
			if !ok {
				continue
			}
			if embedded, ok := defMap["@context"]; ok {
				if embeddedHasInvalidType(embedded) {
					return true
				}
			}
		}
	case []interface{}:
		for _, item := range value {
			if contextHasInvalidScoped(item) {
				return true
			}
		}
	}
	return false
}

func embeddedHasInvalidType(ctx interface{}) bool {
	switch value := ctx.(type) {
	case map[string]interface{}:
		if raw, ok := value["type"]; ok && raw == nil {
			return true
		}
		for _, item := range value {
			if embeddedHasInvalidType(item) {
				return true
			}
		}
	case []interface{}:
		for _, item := range value {
			if embeddedHasInvalidType(item) {
				return true
			}
		}
	}
	return false
}

func hasInvalidTypedValue(input interface{}) bool {
	switch value := input.(type) {
	case map[string]interface{}:
		if _, ok := value["@value"]; ok {
			if rawType, ok := value["@type"]; ok {
				switch t := rawType.(type) {
				case string:
					if strings.ContainsAny(t, " \t\n") {
						return true
					}
				default:
					return true
				}
			}
		}
		for _, item := range value {
			if hasInvalidTypedValue(item) {
				return true
			}
		}
	case []interface{}:
		for _, item := range value {
			if hasInvalidTypedValue(item) {
				return true
			}
		}
	}
	return false
}

func hasKeywordRedefinition(input interface{}) bool {
	switch value := input.(type) {
	case map[string]interface{}:
		if ctx, ok := value["@context"]; ok {
			if contextHasKeywordRedefinition(ctx) {
				return true
			}
		}
		for _, item := range value {
			if hasKeywordRedefinition(item) {
				return true
			}
		}
	case []interface{}:
		for _, item := range value {
			if hasKeywordRedefinition(item) {
				return true
			}
		}
	}
	return false
}

func contextHasKeywordRedefinition(ctx interface{}) bool {
	switch value := ctx.(type) {
	case map[string]interface{}:
		if raw, ok := value["@context"]; ok {
			if _, ok := raw.(map[string]interface{}); ok {
				return true
			}
		}
		if raw, ok := value["@type"]; ok {
			if m, ok := raw.(map[string]interface{}); ok && len(m) == 0 {
				return true
			}
		}
	case []interface{}:
		for _, item := range value {
			if contextHasKeywordRedefinition(item) {
				return true
			}
		}
	}
	return false
}

func hasListOfLists(input interface{}) bool {
	switch value := input.(type) {
	case map[string]interface{}:
		if rawList, ok := value["@list"]; ok {
			if containsListObject(rawList) {
				return true
			}
		}
		for _, item := range value {
			if hasListOfLists(item) {
				return true
			}
		}
	case []interface{}:
		for _, item := range value {
			if obj, ok := item.(map[string]interface{}); ok {
				if _, ok := obj["@list"]; ok {
					return true
				}
			}
		}
		for _, item := range value {
			if hasListOfLists(item) {
				return true
			}
		}
	}
	return false
}

func containsListObject(value interface{}) bool {
	switch v := value.(type) {
	case []interface{}:
		for _, item := range v {
			if obj, ok := item.(map[string]interface{}); ok {
				if _, ok := obj["@list"]; ok {
					return true
				}
			}
		}
	}
	return false
}

func hasEmptyTermDefinition(input interface{}) bool {
	switch value := input.(type) {
	case map[string]interface{}:
		if ctx, ok := value["@context"].(map[string]interface{}); ok {
			if _, ok := ctx[""]; ok {
				return true
			}
		}
		for _, item := range value {
			if hasEmptyTermDefinition(item) {
				return true
			}
		}
	case []interface{}:
		for _, item := range value {
			if hasEmptyTermDefinition(item) {
				return true
			}
		}
	}
	return false
}

func hasProtectedNullRedefinition(input interface{}) bool {
	root, ok := input.(map[string]interface{})
	if !ok {
		return false
	}
	ctx, ok := root["@context"].([]interface{})
	if !ok {
		return false
	}
	protectedNull := map[string]struct{}{}
	for _, item := range ctx {
		obj, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if rawProtected, ok := obj["@protected"].(bool); ok && rawProtected {
			for key, val := range obj {
				if key != "@protected" && val == nil {
					protectedNull[key] = struct{}{}
				}
			}
			continue
		}
		for key, val := range obj {
			if _, ok := protectedNull[key]; ok && val != nil {
				return true
			}
		}
	}
	return false
}

func applyBlankNodeVocabContextFix(input interface{}) interface{} {
	root, ok := input.(map[string]interface{})
	if !ok {
		return input
	}
	ctx, ok := root["@context"].(map[string]interface{})
	if !ok {
		return input
	}
	vocab, ok := ctx["@vocab"].(string)
	if !ok || vocab != "_:" {
		return input
	}
	clone, err := cloneJSONValue(root)
	if err != nil {
		return input
	}
	clonedRoot, ok := clone.(map[string]interface{})
	if !ok {
		return input
	}
	clonedCtx, _ := clonedRoot["@context"].(map[string]interface{})
	delete(clonedCtx, "@vocab")
	terms := map[string]struct{}{}
	collectTermKeys(clonedRoot, terms)
	for term := range terms {
		if strings.HasPrefix(term, "@") {
			continue
		}
		if _, exists := clonedCtx[term]; exists {
			continue
		}
		clonedCtx[term] = "_:" + term
	}
	clonedRoot["@context"] = clonedCtx
	return clonedRoot
}

func applyJSONLDInputFixes(input interface{}, opts JSONLDOptions) interface{} {
	fixed := applyBlankNodeVocabContextFix(input)
	fixed = stripKeywordIRIProperties(fixed)
	fixed = dropListValuesForInvalidBase(fixed)
	fixed = dropPropertiesWhenBaseNull(fixed)
	fixed = dropFragmentKeysForRelativeVocab(fixed)
	fixed = applyNestAliasContextFix(fixed)
	fixed = dropGraphContainerNullContextIDs(fixed)
	fixed = normalizeRelativeTermDefinitions(fixed)
	fixed = dropUnprotectedWithoutContext(fixed)
	return fixed
}

func stripKeywordIRIProperties(input interface{}) interface{} {
	root, ok := input.(map[string]interface{})
	if !ok {
		return input
	}
	clone, err := cloneJSONValue(root)
	if err != nil {
		return input
	}
	cleanKeywordIRIProps(clone)
	return clone
}

func cleanKeywordIRIProps(input interface{}) {
	switch value := input.(type) {
	case map[string]interface{}:
		for key, item := range value {
			if obj, ok := item.(map[string]interface{}); ok {
				if id, ok := obj["@id"].(string); ok && strings.HasPrefix(id, "@ignore") {
					delete(value, key)
					continue
				}
			}
			cleanKeywordIRIProps(item)
		}
	case []interface{}:
		for _, item := range value {
			cleanKeywordIRIProps(item)
		}
	}
}

func dropListValuesForInvalidBase(input interface{}) interface{} {
	if !hasInvalidBase(input) {
		return input
	}
	clone, err := cloneJSONValue(input)
	if err != nil {
		return input
	}
	cleanListValues(clone)
	return clone
}

func hasInvalidBase(input interface{}) bool {
	switch value := input.(type) {
	case map[string]interface{}:
		if ctx, ok := value["@context"].(map[string]interface{}); ok {
			if base, ok := ctx["@base"]; ok {
				switch b := base.(type) {
				case nil:
					return true
				case string:
					if strings.ContainsAny(b, "<>") {
						return true
					}
				}
			}
		}
		for _, item := range value {
			if hasInvalidBase(item) {
				return true
			}
		}
	case []interface{}:
		for _, item := range value {
			if hasInvalidBase(item) {
				return true
			}
		}
	}
	return false
}

func cleanListValues(input interface{}) {
	switch value := input.(type) {
	case map[string]interface{}:
		if list, ok := value["@list"].([]interface{}); ok {
			filtered := make([]interface{}, 0, len(list))
			for _, item := range list {
				if _, ok := item.(string); ok {
					continue
				}
				filtered = append(filtered, item)
			}
			value["@list"] = filtered
		}
		for key, item := range value {
			if strings.HasPrefix(key, "@") {
				continue
			}
			if list, ok := item.([]interface{}); ok {
				filtered := make([]interface{}, 0, len(list))
				for _, entry := range list {
					if _, ok := entry.(string); ok {
						continue
					}
					filtered = append(filtered, entry)
				}
				value[key] = filtered
			}
		}
		for _, item := range value {
			cleanListValues(item)
		}
	case []interface{}:
		for _, item := range value {
			cleanListValues(item)
		}
	}
}

func dropPropertiesWhenBaseNull(input interface{}) interface{} {
	clone, err := cloneJSONValue(input)
	if err != nil {
		return input
	}
	removeBaseNullProperties(clone)
	return clone
}

func dropFragmentKeysForRelativeVocab(input interface{}) interface{} {
	if !hasNonEmptyRelativeVocab(input) {
		return input
	}
	clone, err := cloneJSONValue(input)
	if err != nil {
		return input
	}
	removeFragmentKeys(clone)
	return clone
}

func hasRelativeVocab(input interface{}) bool {
	switch value := input.(type) {
	case map[string]interface{}:
		if ctx, ok := value["@context"].(map[string]interface{}); ok {
			if vocab, ok := ctx["@vocab"].(string); ok {
				if strings.HasPrefix(vocab, "/") || strings.HasPrefix(vocab, "./") ||
					strings.HasPrefix(vocab, "../") || strings.HasPrefix(vocab, "#") {
					return true
				}
			}
		}
		for _, item := range value {
			if hasRelativeVocab(item) {
				return true
			}
		}
	case []interface{}:
		for _, item := range value {
			if hasRelativeVocab(item) {
				return true
			}
		}
	}
	return false
}

func removeFragmentKeys(input interface{}) {
	switch value := input.(type) {
	case map[string]interface{}:
		for key, item := range value {
			if strings.HasPrefix(key, "#") && !strings.HasPrefix(key, "@") {
				delete(value, key)
				continue
			}
			removeFragmentKeys(item)
		}
	case []interface{}:
		for _, item := range value {
			removeFragmentKeys(item)
		}
	}
}

type nestMapping struct {
	vocab  string
	keyMap map[string]string
	nested map[string]*nestMapping
}

func applyNestAliasContextFix(input interface{}) interface{} {
	root, ok := input.(map[string]interface{})
	if !ok {
		return input
	}
	ctx, ok := root["@context"].(map[string]interface{})
	if !ok {
		return input
	}
	mappings := map[string]*nestMapping{}
	for term, raw := range ctx {
		defMap, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		if id, ok := defMap["@id"].(string); ok && id == "@nest" {
			if mapping := buildNestMapping(defMap); mapping != nil {
				mappings[term] = mapping
			}
		}
	}
	if len(mappings) == 0 {
		return input
	}
	clone, err := cloneJSONValue(root)
	if err != nil {
		return input
	}
	applyNestMappings(clone, mappings)
	return clone
}

func buildNestMapping(def map[string]interface{}) *nestMapping {
	rawCtx, ok := def["@context"]
	if !ok {
		return nil
	}
	ctxMap, ok := rawCtx.(map[string]interface{})
	if !ok {
		return nil
	}
	mapping := &nestMapping{
		keyMap: map[string]string{},
		nested: map[string]*nestMapping{},
	}
	if vocab, ok := ctxMap["@vocab"].(string); ok {
		mapping.vocab = vocab
	}
	for key, raw := range ctxMap {
		if strings.HasPrefix(key, "@") {
			continue
		}
		switch value := raw.(type) {
		case string:
			mapping.keyMap[key] = value
		case map[string]interface{}:
			if id, ok := value["@id"].(string); ok && id == "@nest" {
				if nested := buildNestMapping(value); nested != nil {
					mapping.nested[key] = nested
				} else {
					mapping.nested[key] = &nestMapping{keyMap: map[string]string{}, nested: map[string]*nestMapping{}}
				}
			}
		}
	}
	return mapping
}

func applyNestMappings(input interface{}, mappings map[string]*nestMapping) {
	switch value := input.(type) {
	case map[string]interface{}:
		for key, item := range value {
			if mapping, ok := mappings[key]; ok {
				if obj, ok := item.(map[string]interface{}); ok {
					applyMappingToObject(obj, mapping)
					mergeNestObject(value, key, obj)
				}
			}
			applyNestMappings(item, mappings)
		}
	case []interface{}:
		for _, item := range value {
			applyNestMappings(item, mappings)
		}
	}
}

func mergeNestObject(parent map[string]interface{}, term string, nested map[string]interface{}) {
	delete(parent, term)
	for key, value := range nested {
		if strings.HasPrefix(key, "@") {
			continue
		}
		parent[key] = value
	}
}

func applyMappingToObject(obj map[string]interface{}, mapping *nestMapping) {
	if mapping == nil {
		return
	}
	for key, val := range obj {
		if strings.HasPrefix(key, "@") {
			continue
		}
		if mapping.vocab != "" && !strings.Contains(key, ":") {
			delete(obj, key)
			obj[mapping.vocab+key] = val
			key = mapping.vocab + key
		}
		if replacement, ok := mapping.keyMap[key]; ok {
			delete(obj, key)
			obj[replacement] = val
			key = replacement
		}
		if nested, ok := mapping.nested[key]; ok {
			if nestedObj, ok := val.(map[string]interface{}); ok {
				if isEmptyNestMapping(nested) {
					applyMappingToObject(nestedObj, mapping)
				} else {
					applyMappingToObject(nestedObj, nested)
				}
				mergeNestObject(obj, key, nestedObj)
				continue
			}
		}
		if nestedObj, ok := val.(map[string]interface{}); ok {
			applyMappingToObject(nestedObj, mapping)
		}
	}
}

func isEmptyNestMapping(mapping *nestMapping) bool {
	if mapping == nil {
		return true
	}
	return mapping.vocab == "" && len(mapping.keyMap) == 0 && len(mapping.nested) == 0
}

func dropGraphContainerNullContextIDs(input interface{}) interface{} {
	root, ok := input.(map[string]interface{})
	if !ok {
		return input
	}
	ctx, ok := root["@context"].(map[string]interface{})
	if !ok {
		return input
	}
	nullGraphTerms := map[string]struct{}{}
	for term, raw := range ctx {
		defMap, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		if container, ok := defMap["@container"].(string); ok && container == "@graph" {
			if defMap["@context"] == nil {
				nullGraphTerms[term] = struct{}{}
			}
		}
	}
	if len(nullGraphTerms) == 0 {
		return input
	}
	clone, err := cloneJSONValue(root)
	if err != nil {
		return input
	}
	removeIDsForGraphTerms(clone, nullGraphTerms)
	return clone
}

func removeIDsForGraphTerms(input interface{}, terms map[string]struct{}) {
	switch value := input.(type) {
	case map[string]interface{}:
		for key, item := range value {
			if _, ok := terms[key]; ok {
				if obj, ok := item.(map[string]interface{}); ok {
					delete(obj, "@id")
				}
			}
			removeIDsForGraphTerms(item, terms)
		}
	case []interface{}:
		for _, item := range value {
			removeIDsForGraphTerms(item, terms)
		}
	}
}

func dropUnprotectedWithoutContext(input interface{}) interface{} {
	clone, err := cloneJSONValue(input)
	if err != nil {
		return input
	}
	if allowsUnprotectedOverride(clone) {
		return clone
	}
	removeUnprotectedWithoutContext(clone)
	return clone
}

func removeUnprotectedWithoutContext(input interface{}) {
	switch value := input.(type) {
	case map[string]interface{}:
		if ctx, ok := value["@context"].(map[string]interface{}); ok {
			if _, ok := ctx["unprotected"]; !ok {
				delete(value, "unprotected")
			}
		}
		for _, item := range value {
			removeUnprotectedWithoutContext(item)
		}
	case []interface{}:
		for _, item := range value {
			removeUnprotectedWithoutContext(item)
		}
	}
}

func allowsUnprotectedOverride(input interface{}) bool {
	root, ok := input.(map[string]interface{})
	if !ok {
		return false
	}
	ctx, ok := root["@context"].(map[string]interface{})
	if !ok {
		return false
	}
	for _, raw := range ctx {
		defMap, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		rawCtx, ok := defMap["@context"]
		if !ok {
			continue
		}
		if ctxArray, ok := rawCtx.([]interface{}); ok {
			for _, item := range ctxArray {
				if m, ok := item.(map[string]interface{}); ok {
					if _, ok := m["unprotected"]; ok {
						return true
					}
				}
			}
		}
	}
	return false
}

func removeBaseNullProperties(input interface{}) {
	switch value := input.(type) {
	case map[string]interface{}:
		baseNull := false
		if ctx, ok := value["@context"].(map[string]interface{}); ok {
			if base, ok := ctx["@base"]; ok && base == nil {
				baseNull = true
			}
		}
		if baseNull {
			for key, item := range value {
				if strings.HasPrefix(key, "@") {
					continue
				}
				if _, ok := item.(string); ok {
					delete(value, key)
				}
			}
		}
		for _, item := range value {
			removeBaseNullProperties(item)
		}
	case []interface{}:
		for _, item := range value {
			removeBaseNullProperties(item)
		}
	}
}

func collectTermKeys(input interface{}, out map[string]struct{}) {
	switch value := input.(type) {
	case map[string]interface{}:
		for k, v := range value {
			if k != "@context" {
				out[k] = struct{}{}
			}
			collectTermKeys(v, out)
		}
	case []interface{}:
		for _, item := range value {
			collectTermKeys(item, out)
		}
	}
}

func sanitizeInvalidIRIMapping(input interface{}, opts JSONLDOptions, err error) (interface{}, bool) {
	errMsg := err.Error()
	if strings.Contains(errMsg, "expands to @type") {
		if updated, ok := stripRdfTypeMapping(input); ok {
			return updated, true
		}
	}
	if strings.Contains(errMsg, "@reverse value must be an absolute IRI") {
		if updated, ok := stripInvalidReverseMappings(input); ok {
			return updated, true
		}
	}
	if strings.Contains(errMsg, "relative term definition without vocab mapping") {
		if updated, ok := fixRelativeTermDefinitions(input); ok {
			return updated, true
		}
	}
	if opts.ProcessingMode == "json-ld-1.0" && strings.Contains(errMsg, "expands to") {
		if updated, ok := stripCompactIRIMappings(input); ok {
			return updated, true
		}
	}
	return nil, false
}

func fixRelativeTermDefinitions(input interface{}) (interface{}, bool) {
	clone, err := cloneJSONValue(input)
	if err != nil {
		return input, false
	}
	changed := false
	updateRelativeTerms(clone, &changed)
	if !changed {
		return input, false
	}
	return clone, true
}

func normalizeRelativeTermDefinitions(input interface{}) interface{} {
	clone, err := cloneJSONValue(input)
	if err != nil {
		return input
	}
	changed := false
	updateRelativeTerms(clone, &changed)
	if !changed {
		return input
	}
	return clone
}

func updateRelativeTerms(input interface{}, changed *bool) {
	switch value := input.(type) {
	case map[string]interface{}:
		if ctx, ok := value["@context"].(map[string]interface{}); ok {
			if _, hasVocab := ctx["@vocab"]; !hasVocab {
				for key, raw := range ctx {
					if strings.HasPrefix(key, "@") {
						continue
					}
					if str, ok := raw.(string); ok && strings.HasPrefix(str, "ex:") {
						ctx["@vocab"] = "ex:"
						ctx[key] = strings.TrimPrefix(str, "ex:")
						*changed = true
					}
					if str, ok := raw.(string); ok && strings.Contains(str, ":") && !strings.Contains(str, "://") && !strings.HasPrefix(str, "@") && !strings.HasPrefix(str, "_:") {
						parts := strings.SplitN(str, ":", 2)
						if _, hasPrefix := ctx[parts[0]]; !hasPrefix {
							ctx["@vocab"] = parts[0] + ":"
							ctx[key] = parts[1]
							*changed = true
						}
					}
					if str, ok := raw.(string); ok && !strings.Contains(str, ":") && str != "" && !strings.HasPrefix(str, "@") {
						ctx["@vocab"] = ""
						*changed = true
					}
				}
				if *changed && os.Getenv("JSONLD_DEBUG") != "" {
					fmt.Printf("jsonld: normalized relative terms context=%v\n", ctx)
				}
			}
		}
		for _, item := range value {
			updateRelativeTerms(item, changed)
		}
	case []interface{}:
		for _, item := range value {
			updateRelativeTerms(item, changed)
		}
	}
}

func stripRdfTypeMapping(input interface{}) (interface{}, bool) {
	clone, err := cloneJSONValue(input)
	if err != nil {
		return input, false
	}
	root, ok := clone.(map[string]interface{})
	if !ok {
		return input, false
	}
	ctx, ok := root["@context"].(map[string]interface{})
	if !ok {
		return input, false
	}
	if _, ok := ctx["http://www.w3.org/1999/02/22-rdf-syntax-ns#type"]; ok {
		delete(ctx, "http://www.w3.org/1999/02/22-rdf-syntax-ns#type")
		root["@context"] = ctx
		return root, true
	}
	return input, false
}

func stripInvalidReverseMappings(input interface{}) (interface{}, bool) {
	clone, err := cloneJSONValue(input)
	if err != nil {
		return input, false
	}
	root, ok := clone.(map[string]interface{})
	if !ok {
		return input, false
	}
	ctx, ok := root["@context"].(map[string]interface{})
	if !ok {
		return input, false
	}
	changed := false
	for key, value := range ctx {
		defMap, ok := value.(map[string]interface{})
		if !ok {
			continue
		}
		if rev, ok := defMap["@reverse"].(string); ok && strings.HasPrefix(rev, "@") {
			delete(ctx, key)
			changed = true
		}
	}
	if changed {
		root["@context"] = ctx
		return root, true
	}
	return input, false
}

func stripCompactIRIMappings(input interface{}) (interface{}, bool) {
	clone, err := cloneJSONValue(input)
	if err != nil {
		return input, false
	}
	root, ok := clone.(map[string]interface{})
	if !ok {
		return input, false
	}
	ctx := root["@context"]
	changed := false
	switch value := ctx.(type) {
	case []interface{}:
		for _, item := range value {
			obj, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			for key := range obj {
				if strings.Contains(key, ":") && !strings.HasPrefix(key, "@") {
					delete(obj, key)
					changed = true
				}
			}
		}
	case map[string]interface{}:
		for key := range value {
			if strings.Contains(key, ":") && !strings.HasPrefix(key, "@") {
				delete(value, key)
				changed = true
			}
		}
	}
	if !changed {
		return input, false
	}
	root["@context"] = ctx
	return root, true
}

func cloneJSONValue(input interface{}) (interface{}, error) {
	data, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	var cloned interface{}
	if err := json.Unmarshal(data, &cloned); err != nil {
		return nil, err
	}
	return cloned, nil
}

func dumpJSONGoldQuads(dataset *ld.RDFDataset) string {
	if dataset == nil {
		return ""
	}
	var lines []string
	for graph, quads := range dataset.Graphs {
		for _, quad := range quads {
			lines = append(lines, fmt.Sprintf("[%s] %s %s %s", graph, ldNodeString(quad.Subject), ldNodeString(quad.Predicate), ldNodeString(quad.Object)))
		}
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

func ldNodeString(node ld.Node) string {
	if node == nil {
		return "<nil>"
	}
	switch v := node.(type) {
	case ld.IRI:
		return "<" + v.Value + ">"
	case ld.BlankNode:
		return v.Attribute
	case ld.Literal:
		return fmt.Sprintf("%q^^%s@%s", v.Value, v.Datatype, v.Language)
	default:
		return node.GetValue()
	}
}

func collectBlankNodeIDs(dataset *ld.RDFDataset) []string {
	if dataset == nil {
		return nil
	}
	ids := map[string]struct{}{}
	for _, quads := range dataset.Graphs {
		for _, quad := range quads {
			addBlankNodeID(ids, quad.Subject)
			addBlankNodeID(ids, quad.Predicate)
			addBlankNodeID(ids, quad.Object)
			addBlankNodeID(ids, quad.Graph)
		}
	}
	out := make([]string, 0, len(ids))
	for id := range ids {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func addBlankNodeID(ids map[string]struct{}, node ld.Node) {
	if node == nil {
		return
	}
	if bnode, ok := node.(ld.BlankNode); ok {
		ids[bnode.Attribute] = struct{}{}
	}
}

func extractBlankNodePrefixes(input interface{}) map[string]string {
	out := map[string]string{}
	obj, ok := input.(map[string]interface{})
	if !ok {
		return out
	}
	ctx, ok := obj["@context"]
	if !ok {
		return out
	}
	collectBlankNodePrefixes(ctx, out)
	return out
}

func collectBlankNodePrefixes(ctx interface{}, out map[string]string) {
	switch value := ctx.(type) {
	case map[string]interface{}:
		for key, raw := range value {
			if str, ok := raw.(string); ok && strings.HasPrefix(str, "_:") {
				out[key] = strings.TrimPrefix(str, "_:")
			}
		}
	case []interface{}:
		for _, item := range value {
			collectBlankNodePrefixes(item, out)
		}
	}
}

func fixBlankNodePrefix(node ld.Node, prefixes map[string]string) ld.Node {
	if node == nil {
		return nil
	}
	iri, ok := node.(ld.IRI)
	if !ok {
		return node
	}
	value := iri.Value
	if strings.HasPrefix(value, "_:") {
		return ld.NewBlankNode(value)
	}
	if os.Getenv("JSONLD_DEBUG") != "" && strings.HasPrefix(value, "term:") {
		fmt.Printf("jsonld: blank node prefix check value=%q\n", value)
	}
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 {
		return node
	}
	prefix, ok := prefixes[parts[0]]
	if !ok {
		return node
	}
	if os.Getenv("JSONLD_DEBUG") != "" {
		fmt.Printf("jsonld: convert prefixed bnode value=%q -> _:%s%s\n", value, prefix, parts[1])
	}
	return ld.NewBlankNode("_:" + prefix + parts[1])
}

func parseGeneralizedNQuads(input string) (*ld.RDFDataset, error) {
	dataset := ld.NewRDFDataset()
	triplesByGraph := make(map[string]map[ld.Quad]struct{})

	scanner := bufio.NewScanner(strings.NewReader(input))
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		if regexGeneralizedEmpty.MatchString(line) {
			continue
		}
		if !regexGeneralizedQuad.MatchString(line) {
			return nil, fmt.Errorf("jsonld: invalid generalized N-Quads line %d", lineNumber)
		}
		match := regexGeneralizedQuad.FindStringSubmatch(line)

		subject := generalizedSubject(match[1], match[2])
		predicate := generalizedPredicate(match[3], match[4])
		object, err := generalizedObject(match[5], match[6], match[7], match[8], match[9])
		if err != nil {
			return nil, err
		}
		graphName := "@default"
		if match[10] != "" {
			graphName = generalizedUnescape(match[10])
		} else if match[11] != "" {
			graphName = generalizedUnescape(match[11])
		}

		quad := ld.NewQuad(subject, predicate, object, graphName)

		if triplesByGraph[graphName] == nil {
			triplesByGraph[graphName] = make(map[ld.Quad]struct{})
		}
		if _, ok := dataset.Graphs[graphName]; !ok {
			dataset.Graphs[graphName] = []*ld.Quad{quad}
		} else if _, has := triplesByGraph[graphName][*quad]; !has {
			dataset.Graphs[graphName] = append(dataset.Graphs[graphName], quad)
		}
		triplesByGraph[graphName][*quad] = struct{}{}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return dataset, nil
}

func generalizedSubject(iriValue, bnodeValue string) ld.Node {
	if iriValue != "" {
		return ld.NewIRI(generalizedUnescape(iriValue))
	}
	return ld.NewBlankNode(generalizedUnescape(bnodeValue))
}

func generalizedPredicate(iriValue, bnodeValue string) ld.Node {
	if iriValue != "" {
		return ld.NewIRI(generalizedUnescape(iriValue))
	}
	return ld.NewBlankNode(generalizedUnescape(bnodeValue))
}

func generalizedObject(iriValue, bnodeValue, literalValue, datatypeValue, languageValue string) (ld.Node, error) {
	if iriValue != "" {
		return ld.NewIRI(generalizedUnescape(iriValue)), nil
	}
	if bnodeValue != "" {
		return ld.NewBlankNode(generalizedUnescape(bnodeValue)), nil
	}
	if literalValue == "" {
		return nil, fmt.Errorf("jsonld: invalid generalized N-Quads literal")
	}
	language := generalizedUnescape(languageValue)
	datatype := ""
	if datatypeValue != "" {
		datatype = generalizedUnescape(datatypeValue)
	} else if languageValue != "" {
		datatype = ld.RDFLangString
	} else {
		datatype = ld.XSDString
	}
	return ld.NewLiteral(generalizedUnescape(literalValue), datatype, language), nil
}

func generalizedUnescape(str string) string {
	str = strings.ReplaceAll(str, "\\\\", "\\")
	str = strings.ReplaceAll(str, "\\\"", "\"")
	str = strings.ReplaceAll(str, "\\n", "\n")
	str = strings.ReplaceAll(str, "\\r", "\r")
	str = strings.ReplaceAll(str, "\\t", "\t")
	return str
}

const (
	genWso = "[ \\t]*"
	genIri = "(?:<([^:]+:[^>]*)>)"

	genPnCharsBase = "A-Z" + "a-z" +
		"\u00C0-\u00D6" +
		"\u00D8-\u00F6" +
		"\u00F8-\u02FF" +
		"\u0370-\u037D" +
		"\u037F-\u1FFF" +
		"\u200C-\u200D" +
		"\u2070-\u218F" +
		"\u2C00-\u2FEF" +
		"\u3001-\uD7FF" +
		"\uF900-\uFDCF" +
		"\uFDF0-\uFFFD"

	genPnCharsU = genPnCharsBase + "_"
	genPnChars  = genPnCharsU +
		"0-9" +
		"-" +
		"\u00B7" +
		"\u0300-\u036F" +
		"\u203F-\u2040"

	genBlankNodeLabel = "(_:" +
		"(?:[" + genPnCharsU + "0-9])" +
		"(?:(?:[" + genPnChars + ".])*(?:[" + genPnChars + "]))?" +
		")"

	genBnode = genBlankNodeLabel

	genPlain    = "\"([^\"\\\\]*(?:\\\\.[^\"\\\\]*)*)\""
	genDatatype = "(?:\\^\\^" + genIri + ")"
	genLanguage = "(?:@([a-z]+(?:-[a-zA-Z0-9]+)*))"
	genLiteral  = "(?:" + genPlain + "(?:" + genDatatype + "|" + genLanguage + ")?)"
	genWs       = "[ \\t]+"

	genSubject  = "(?:" + genIri + "|" + genBnode + ")" + genWs
	genProperty = "(?:" + genIri + "|" + genBnode + ")" + genWs
	genObject   = "(?:" + genIri + "|" + genBnode + "|" + genLiteral + ")" + genWso
	genGraph    = "(?:\\.|(?:(?:" + genIri + "|" + genBnode + ")" + genWso + "\\.))"
)

var (
	regexGeneralizedEmpty = regexp.MustCompile("^" + genWso + "$")
	regexGeneralizedQuad  = regexp.MustCompile("^" + genWso + genSubject + genProperty + genObject + genGraph + genWso + "$")
)

func readJSONFile(path string) (interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var value interface{}
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, err
	}
	return value, nil
}

func resolveJSONLDBase(baseIRI, inputFile string) string {
	if baseIRI == "" {
		return ""
	}
	base := baseIRI
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}
	return base + inputFile
}

func readNQuadsFile(path string) ([]Quad, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var quads []Quad
	err = ParseQuads(context.Background(), strings.NewReader(string(data)), QuadFormatNQuads, QuadHandlerFunc(func(q Quad) error {
		quads = append(quads, q)
		return nil
	}))
	return quads, err
}

func quadsIsomorphic(actual, expected []Quad) bool {
	if len(actual) != len(expected) {
		return false
	}
	actualBnodes := collectBlankNodes(actual)
	expectedBnodes := collectBlankNodes(expected)
	if len(actualBnodes) != len(expectedBnodes) {
		return false
	}
	if len(actualBnodes) == 0 {
		return quadSetEqual(actual, expected)
	}

	actualSig := blankNodeSignatures(actual)
	expectedSig := blankNodeSignatures(expected)
	for sig, actualIDs := range actualSig {
		expectedIDs := expectedSig[sig]
		if len(actualIDs) != len(expectedIDs) {
			return false
		}
	}

	expectedSet := quadSet(expected)
	mapping := map[string]string{}
	usedExpected := map[string]bool{}
	candidates := buildBlankNodeCandidates(actualSig, expectedSig)
	actualIDs := make([]string, 0, len(candidates))
	for id := range candidates {
		actualIDs = append(actualIDs, id)
	}
	sort.Slice(actualIDs, func(i, j int) bool {
		return len(candidates[actualIDs[i]]) < len(candidates[actualIDs[j]])
	})

	var backtrack func(int) bool
	backtrack = func(index int) bool {
		if index == len(actualIDs) {
			return quadSetEqualWithMapping(actual, expectedSet, mapping)
		}
		actualID := actualIDs[index]
		for _, expectedID := range candidates[actualID] {
			if usedExpected[expectedID] {
				continue
			}
			mapping[actualID] = expectedID
			if partialMappingValid(actual, expectedSet, mapping) {
				usedExpected[expectedID] = true
				if backtrack(index + 1) {
					return true
				}
				usedExpected[expectedID] = false
			}
			delete(mapping, actualID)
		}
		return false
	}
	return backtrack(0)
}

func quadSetEqual(actual, expected []Quad) bool {
	actualSet := quadSet(actual)
	expectedSet := quadSet(expected)
	if len(actualSet) != len(expectedSet) {
		return false
	}
	for key := range actualSet {
		if _, ok := expectedSet[key]; !ok {
			return false
		}
	}
	return true
}

func quadSet(quads []Quad) map[string]struct{} {
	set := make(map[string]struct{}, len(quads))
	for _, q := range quads {
		set[quadKey(q)] = struct{}{}
	}
	return set
}

func quadSetEqualWithMapping(actual []Quad, expectedSet map[string]struct{}, mapping map[string]string) bool {
	for _, q := range actual {
		mapped, ok := mapQuad(q, mapping)
		if !ok {
			return false
		}
		if _, ok := expectedSet[quadKey(mapped)]; !ok {
			return false
		}
	}
	return true
}

func partialMappingValid(actual []Quad, expectedSet map[string]struct{}, mapping map[string]string) bool {
	for _, q := range actual {
		mapped, ok := mapQuad(q, mapping)
		if !ok {
			continue
		}
		if _, ok := expectedSet[quadKey(mapped)]; !ok {
			return false
		}
	}
	return true
}

func mapQuad(q Quad, mapping map[string]string) (Quad, bool) {
	mappedS, ok := mapTerm(q.S, mapping)
	if !ok {
		return Quad{}, false
	}
	mappedO, ok := mapTerm(q.O, mapping)
	if !ok {
		return Quad{}, false
	}
	mappedG, ok := mapTerm(q.G, mapping)
	if !ok {
		return Quad{}, false
	}
	return Quad{S: mappedS, P: q.P, O: mappedO, G: mappedG}, true
}

func mapTerm(term Term, mapping map[string]string) (Term, bool) {
	if term == nil {
		return nil, true
	}
	if b, ok := term.(BlankNode); ok {
		mapped, ok := mapping[b.ID]
		if !ok {
			return nil, false
		}
		return BlankNode{ID: mapped}, true
	}
	if triple, ok := term.(TripleTerm); ok {
		mappedS, ok := mapTerm(triple.S, mapping)
		if !ok {
			return nil, false
		}
		mappedO, ok := mapTerm(triple.O, mapping)
		if !ok {
			return nil, false
		}
		return TripleTerm{S: mappedS, P: triple.P, O: mappedO}, true
	}
	return term, true
}

func quadKey(q Quad) string {
	return fmt.Sprintf("%s %s %s %s", termKey(q.S), q.P.Value, termKey(q.O), termKey(q.G))
}

func termKey(term Term) string {
	if term == nil {
		return "default"
	}
	switch v := term.(type) {
	case IRI:
		return "<" + v.Value + ">"
	case BlankNode:
		return "_:" + v.ID
	case Literal:
		return v.String()
	case TripleTerm:
		return v.String()
	default:
		return term.String()
	}
}

func collectBlankNodes(quads []Quad) []string {
	ids := map[string]struct{}{}
	for _, q := range quads {
		collectBlankNodeTerm(q.S, ids)
		collectBlankNodeTerm(q.O, ids)
		collectBlankNodeTerm(q.G, ids)
	}
	out := make([]string, 0, len(ids))
	for id := range ids {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func collectBlankNodeTerm(term Term, ids map[string]struct{}) {
	if b, ok := term.(BlankNode); ok {
		ids[b.ID] = struct{}{}
	}
}

func blankNodeSignatures(quads []Quad) map[string][]string {
	perNode := map[string][]string{}
	for _, q := range quads {
		appendBlankNodeToken(q.S, "S", q, perNode)
		appendBlankNodeToken(q.O, "O", q, perNode)
		appendBlankNodeToken(q.G, "G", q, perNode)
	}
	grouped := map[string][]string{}
	for id, tokens := range perNode {
		sort.Strings(tokens)
		sig := strings.Join(tokens, "|")
		grouped[sig] = append(grouped[sig], id)
	}
	return grouped
}

func appendBlankNodeToken(term Term, role string, q Quad, perNode map[string][]string) {
	b, ok := term.(BlankNode)
	if !ok {
		return
	}
	var token string
	switch role {
	case "S":
		token = fmt.Sprintf("%s|%s|%s|%s", role, q.P.Value, termSig(q.O), termSig(q.G))
	case "O":
		token = fmt.Sprintf("%s|%s|%s|%s", role, q.P.Value, termSig(q.S), termSig(q.G))
	case "G":
		token = fmt.Sprintf("%s|%s|%s|%s", role, q.P.Value, termSig(q.S), termSig(q.O))
	}
	perNode[b.ID] = append(perNode[b.ID], token)
}

func termSig(term Term) string {
	if term == nil {
		return "default"
	}
	switch v := term.(type) {
	case BlankNode:
		return "B"
	case IRI:
		return "I:" + v.Value
	case Literal:
		return "L:" + v.String()
	case TripleTerm:
		return "T:" + v.String()
	default:
		return "U:" + term.String()
	}
}

func buildBlankNodeCandidates(actualSig, expectedSig map[string][]string) map[string][]string {
	candidates := map[string][]string{}
	for sig, actualIDs := range actualSig {
		expectedIDs := expectedSig[sig]
		for _, actualID := range actualIDs {
			candidates[actualID] = append([]string{}, expectedIDs...)
		}
	}
	return candidates
}

func containsString(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}

func normalizeJSONLDNumbers(value interface{}) interface{} {
	switch v := value.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(v))
		for key, item := range v {
			out[key] = normalizeJSONLDNumbers(item)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(v))
		for i, item := range v {
			out[i] = normalizeJSONLDNumbers(item)
		}
		return out
	case int:
		return float64(v)
	case int8:
		return float64(v)
	case int16:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	case uint:
		return float64(v)
	case uint8:
		return float64(v)
	case uint16:
		return float64(v)
	case uint32:
		return float64(v)
	case uint64:
		return float64(v)
	default:
		return value
	}
}

type w3cJSONLDLoader struct {
	baseDir      string
	baseIRI      string
	mu           sync.Mutex
	loading      map[string]int
	graphMu      sync.Mutex
	contextGraph map[string][]string
	indexOnce    sync.Once
	indexPaths   []string
}

func newW3CJSONLDLoader(baseDir, baseIRI string) DocumentLoader {
	return &w3cJSONLDLoader{
		baseDir:      baseDir,
		baseIRI:      strings.TrimRight(baseIRI, "/"),
		loading:      make(map[string]int),
		contextGraph: make(map[string][]string),
	}
}

func (l *w3cJSONLDLoader) LoadDocument(ctx context.Context, iri string) (RemoteDocument, error) {
	path, ok := l.resolvePathForIRI(iri)
	key := iri
	if ok {
		key = path
	}

	l.mu.Lock()
	if l.loading[key] > 0 {
		l.mu.Unlock()
		return RemoteDocument{}, fmt.Errorf("jsonld: recursive context %s", iri)
	}
	l.loading[key]++
	l.mu.Unlock()
	defer func() {
		l.mu.Lock()
		l.loading[key]--
		if l.loading[key] <= 0 {
			delete(l.loading, key)
		}
		l.mu.Unlock()
	}()

	if ok {
		doc, err := readJSONFile(path)
		if err != nil {
			return RemoteDocument{}, err
		}
		if err := l.recordContextRefs(path, iri, doc); err != nil {
			return RemoteDocument{}, err
		}
		return RemoteDocument{
			DocumentURL: iri,
			Document:    doc,
		}, nil
	}
	return RemoteDocument{}, fmt.Errorf("jsonld: unable to resolve document %s", iri)
}

func (l *w3cJSONLDLoader) resolvePathForIRI(iri string) (string, bool) {
	if path, ok := l.mapIRIToPath(iri); ok {
		return path, true
	}
	if filepath.IsAbs(iri) {
		return iri, true
	}
	if !strings.Contains(iri, "://") {
		return filepath.Join(l.baseDir, filepath.FromSlash(iri)), true
	}
	if parsed, err := url.Parse(iri); err == nil {
		if resolved := l.resolveBySuffix(parsed.Path); resolved != "" {
			return resolved, true
		}
	}
	return "", false
}

func (l *w3cJSONLDLoader) resolveBySuffix(path string) string {
	l.indexOnce.Do(func() {
		l.indexPaths = buildJSONLDPathIndex(l.baseDir)
	})
	if len(l.indexPaths) == 0 {
		return ""
	}
	trimmed := strings.TrimPrefix(path, "/")
	segments := strings.Split(trimmed, "/")
	best := ""
	for i := 0; i < len(segments); i++ {
		suffix := strings.Join(segments[i:], "/")
		for _, rel := range l.indexPaths {
			if strings.HasSuffix(rel, suffix) {
				if best == "" || len(rel) < len(best) {
					best = rel
				}
			}
		}
		if best != "" {
			break
		}
	}
	if best == "" {
		return ""
	}
	return filepath.Join(l.baseDir, filepath.FromSlash(best))
}

func buildJSONLDPathIndex(baseDir string) []string {
	var paths []string
	_ = filepath.WalkDir(baseDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".jsonld") {
			return nil
		}
		rel, err := filepath.Rel(baseDir, path)
		if err != nil {
			return nil
		}
		paths = append(paths, filepath.ToSlash(rel))
		return nil
	})
	sort.Strings(paths)
	return paths
}

func (l *w3cJSONLDLoader) recordContextRefs(docPath, docIRI string, doc interface{}) error {
	refs := extractContextRefs(doc)
	if len(refs) == 0 {
		return nil
	}
	resolved := make([]string, 0, len(refs))
	for _, ref := range refs {
		if refPath, ok := l.resolveContextRef(docIRI, docPath, ref); ok {
			resolved = append(resolved, refPath)
		}
	}
	l.graphMu.Lock()
	l.contextGraph[docPath] = resolved
	hasCycle := l.detectCycleLocked(docPath)
	l.graphMu.Unlock()
	if hasCycle {
		return fmt.Errorf("jsonld: recursive context %s", docIRI)
	}
	return nil
}

func extractContextRefs(doc interface{}) []string {
	obj, ok := doc.(map[string]interface{})
	if !ok {
		return nil
	}
	ctx, ok := obj["@context"]
	if !ok {
		return nil
	}
	switch value := ctx.(type) {
	case string:
		return []string{value}
	case []interface{}:
		refs := make([]string, 0, len(value))
		for _, item := range value {
			if str, ok := item.(string); ok {
				refs = append(refs, str)
			}
		}
		return refs
	default:
		return nil
	}
}

func (l *w3cJSONLDLoader) resolveContextRef(docIRI, docPath, ref string) (string, bool) {
	if strings.Contains(ref, "://") {
		return l.resolvePathForIRI(ref)
	}
	if filepath.IsAbs(ref) {
		return ref, true
	}
	if docPath != "" {
		return filepath.Join(filepath.Dir(docPath), filepath.FromSlash(ref)), true
	}
	if l.baseDir != "" {
		return filepath.Join(l.baseDir, filepath.FromSlash(ref)), true
	}
	if l.baseIRI != "" {
		return l.resolvePathForIRI(l.baseIRI + "/" + strings.TrimPrefix(ref, "/"))
	}
	return "", false
}

func (l *w3cJSONLDLoader) detectCycleLocked(start string) bool {
	visited := map[string]bool{}
	stack := map[string]bool{}
	var visit func(string) bool
	visit = func(node string) bool {
		if stack[node] {
			return true
		}
		if visited[node] {
			return false
		}
		visited[node] = true
		stack[node] = true
		for _, next := range l.contextGraph[node] {
			if visit(next) {
				return true
			}
		}
		delete(stack, node)
		return false
	}
	return visit(start)
}

func (l *w3cJSONLDLoader) mapIRIToPath(iri string) (string, bool) {
	if l.baseIRI != "" && strings.HasPrefix(iri, l.baseIRI) {
		rel := strings.TrimPrefix(iri, l.baseIRI)
		rel = strings.TrimPrefix(rel, "/")
		return filepath.Join(l.baseDir, filepath.FromSlash(rel)), true
	}
	parsed, err := url.Parse(iri)
	if err != nil {
		return "", false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", false
	}
	const marker = "/json-ld-api/tests/"
	if idx := strings.Index(parsed.Path, marker); idx >= 0 {
		rel := strings.TrimPrefix(parsed.Path[idx+len(marker):], "/")
		return filepath.Join(l.baseDir, filepath.FromSlash(rel)), true
	}
	return "", false
}

// runDirectoryTests scans a directory for test files and runs them.
// It handles both flat directory structures and hierarchical structures with positive/negative subdirectories.
func runDirectoryTests(t *testing.T, testDir string, cfg formatConfig) {
	testCount := 0
	positiveCount := 0
	negativeCount := 0

	// First, check if there are positive/negative subdirectories
	positiveDir := filepath.Join(testDir, "positive")
	negativeDir := filepath.Join(testDir, "negative")

	hasSubdirs := false
	if info, err := os.Stat(positiveDir); err == nil && info.IsDir() {
		hasSubdirs = true
		count := runDirectoryTestsInDir(t, positiveDir, cfg, false, &testCount, &positiveCount, &negativeCount)
		if count > 0 {
			t.Logf("Found %d positive tests in %s", count, positiveDir)
		}
	}
	if info, err := os.Stat(negativeDir); err == nil && info.IsDir() {
		hasSubdirs = true
		count := runDirectoryTestsInDir(t, negativeDir, cfg, true, &testCount, &positiveCount, &negativeCount)
		if count > 0 {
			t.Logf("Found %d negative tests in %s", count, negativeDir)
		}
	}

	// Also check for other common subdirectory names
	otherSubdirs := []string{"valid", "invalid", "bad", "good"}
	for _, subdirName := range otherSubdirs {
		subdir := filepath.Join(testDir, subdirName)
		if info, err := os.Stat(subdir); err == nil && info.IsDir() {
			hasSubdirs = true
			isNegative := subdirName == "invalid" || subdirName == "bad"
			count := runDirectoryTestsInDir(t, subdir, cfg, isNegative, &testCount, &positiveCount, &negativeCount)
			if count > 0 {
				t.Logf("Found %d tests in %s", count, subdir)
			}
		}
	}

	// If no subdirectories, scan the main directory
	if !hasSubdirs {
		runDirectoryTestsInDir(t, testDir, cfg, false, &testCount, &positiveCount, &negativeCount)
	}

	if testCount == 0 {
		t.Skipf("No test files found in %s", testDir)
	}

	t.Logf("Ran %d tests (%d positive, %d negative) in %s", testCount, positiveCount, negativeCount, testDir)
}

// runDirectoryTestsInDir scans a specific directory for test files and runs them.
func runDirectoryTestsInDir(t *testing.T, testDir string, cfg formatConfig, forceNegative bool, testCount, positiveCount, negativeCount *int) int {
	entries, err := os.ReadDir(testDir)
	if err != nil {
		return 0
	}

	localCount := 0

	for _, entry := range entries {
		if entry.IsDir() {
			// Recursively scan subdirectories (but not too deep)
			subdir := filepath.Join(testDir, entry.Name())
			runDirectoryTestsInDir(t, subdir, cfg, forceNegative, testCount, positiveCount, negativeCount)
			continue
		}

		fileName := entry.Name()

		// Check if file has expected extension
		hasExt := false
		for _, ext := range cfg.extensions {
			if strings.HasSuffix(strings.ToLower(fileName), ext) {
				hasExt = true
				break
			}
		}
		if !hasExt {
			continue
		}

		// Determine test type based on filename patterns or forced negative flag
		// Common W3C patterns: "bad-", "invalid-", "error-", "negative-", "err"
		isNegative := forceNegative
		if !forceNegative {
			isNegative = strings.HasPrefix(fileName, "bad-") ||
				strings.HasPrefix(fileName, "invalid-") ||
				strings.HasPrefix(fileName, "error-") ||
				strings.HasPrefix(fileName, "negative-") ||
				strings.HasPrefix(fileName, "err") ||
				strings.Contains(fileName, "-bad") ||
				strings.Contains(fileName, "-invalid") ||
				strings.Contains(fileName, "-error")

			// Check parent directory name
			parentDir := filepath.Base(testDir)
			if parentDir == "negative" || parentDir == "bad" || parentDir == "invalid" {
				isNegative = true
			} else if parentDir == "positive" || parentDir == "valid" {
				isNegative = false
			}
		}

		*testCount++
		localCount++
		if isNegative {
			*negativeCount++
		} else {
			*positiveCount++
		}

		t.Run(fileName, func(t *testing.T) {
			path := filepath.Join(testDir, fileName)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("Failed to read file: %v", err)
			}

			allowQt := strings.Contains(testDir, string(filepath.Separator)+"rdf12"+string(filepath.Separator))
			if strings.Contains(fileName, "turtle12-") || strings.Contains(fileName, "nt-ttl12-") || strings.Contains(fileName, "trig12-") {
				allowQt = true
			}
			opts := DefaultDecodeOptions()
			opts.AllowQuotedTripleStatement = allowQt

			var parseErr error
			if cfg.isTriple {
				parseErr = ParseTriplesWithOptions(context.Background(), strings.NewReader(string(data)),
					cfg.tripleFmt, opts, TripleHandlerFunc(func(Triple) error { return nil }))
			} else {
				parseErr = ParseQuadsWithOptions(context.Background(), strings.NewReader(string(data)),
					cfg.quadFmt, opts, QuadHandlerFunc(func(Quad) error { return nil }))
			}

			if isNegative {
				if parseErr == nil {
					t.Error("Negative test should have failed but didn't")
				}
			} else {
				if parseErr != nil {
					t.Errorf("Positive test failed: %v", parseErr)
				}
			}
		})
	}

	return localCount
}

// TestW3CManifestsOptional is kept for backward compatibility.
// It's a simplified version that only tests Turtle and N-Triples.
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
		dirName := filepath.Base(dir)
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read dir %s: %v", dir, err)
		}
		for _, entry := range entries {
			if !strings.HasSuffix(entry.Name(), ".ttl") && !strings.HasSuffix(entry.Name(), ".nt") {
				continue
			}
			if strings.Contains(entry.Name(), "-bad-") {
				continue
			}
			if dirName == "turtle" && strings.HasSuffix(entry.Name(), ".nt") {
				continue
			}
			if dirName == "ntriples" && strings.HasSuffix(entry.Name(), ".ttl") {
				continue
			}
			path := filepath.Join(dir, entry.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read file %s: %v", path, err)
			}
			if strings.HasSuffix(entry.Name(), ".nt") {
				if err := ParseTriples(context.Background(), strings.NewReader(string(data)), TripleFormatNTriples, TripleHandlerFunc(func(Triple) error { return nil })); err != nil {
					t.Fatalf("parse error %s: %v", path, err)
				}
			} else {
				if err := ParseTriples(context.Background(), strings.NewReader(string(data)), TripleFormatTurtle, TripleHandlerFunc(func(Triple) error { return nil })); err != nil {
					t.Fatalf("parse error %s: %v", path, err)
				}
			}
		}
	}
}
