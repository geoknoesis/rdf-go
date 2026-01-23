# Comprehensive Code Review: rdf-go

**Review Date:** 2026  
**Reviewer:** World-Class Go & Semantic Web Systems Engineer  
**Target:** Reference-quality, production-grade Go codec library for Linked Data formats

---

## 1) High-Level Verdict

**Is this library close to "reference-quality"?**

**Status:** **Good foundation, but needs significant hardening for production use.**

The library demonstrates solid architectural choices (type-safe format separation, streaming-first design, clean interfaces) and shows evidence of W3C test suite integration. However, critical gaps in security, error diagnostics, and streaming completeness prevent it from being production-ready for untrusted inputs.

### Top 3 Strengths

1. **Clean API Design**: The separation of `TripleFormat` vs `QuadFormat` with type-safe constructors (`NewTripleDecoder`, `NewQuadDecoder`) is excellent. The streaming interfaces (`TripleDecoder`, `QuadDecoder`) are minimal and idiomatic Go.

2. **Streaming Architecture**: Most formats (N-Triples, N-Quads, Turtle, TriG, RDF/XML) properly stream with pull-style decoders. The `Next()` pattern is familiar and efficient.

3. **W3C Test Suite Integration**: Evidence of compliance testing infrastructure (`compliance_test.go`, `w3c-tests/` directory) shows commitment to standards alignment.

### Top 3 Risks

1. **JSON-LD Not Streaming**: The JSON-LD decoder loads the entire document into memory (`jsonldTripleDecoder.load()`), defeating the streaming promise. This is a critical flaw for large datasets.

2. **No Security Limits**: Zero protections against resource exhaustion (unbounded recursion, unlimited line length, no max depth, no timeouts). Malicious inputs can cause DoS.

3. **Poor Error Diagnostics**: Errors lack line numbers, column positions, or context excerpts. Messages like `"ntriples: unexpected end of line"` are unhelpful for debugging real-world files.

---

## 2) Public API & DX Review

### What's Clean

- **Type Safety**: `TripleFormat`/`QuadFormat` separation prevents format mismatches at compile time. Excellent.
- **Interface Minimalism**: `TripleDecoder` and `QuadDecoder` interfaces are exactly right—three methods, clear responsibilities.
- **Helper Functions**: `ParseTriples`, `ParseQuads`, `ParseTriplesChan`, `ParseQuadsChan` provide convenient entry points without cluttering the core API.
- **Format Parsing**: `ParseTripleFormat` and `ParseQuadFormat` handle common aliases gracefully.

### What's Confusing

1. **Inconsistent Streaming Promise**: The README claims "streaming decoders" but JSON-LD loads everything upfront. Users will be surprised.

2. **No Options Pattern**: All decoders/encoders are created with zero configuration. There's no way to:
   - Set base IRI for JSON-LD
   - Configure limits (max depth, max line length)
   - Control error recovery behavior
   - Set timeouts

3. **Error Handling Ambiguity**: `Err()` method exists but its relationship to `Next()` errors is unclear. When should users check `Err()` vs the error from `Next()`?

4. **Missing Format Detection**: No `DetectFormat(io.Reader) (Format, error)` function. Users must know the format ahead of time.

### Suggested API Reshaping

**Option 1: Functional Options (Recommended)**

```go
// Add options for all decoders/encoders
type DecoderOption func(*DecoderConfig)

type DecoderConfig struct {
    BaseIRI     string
    MaxDepth     int  // 0 = unlimited
    MaxLineLen   int  // 0 = unlimited
    MaxTriples   int64 // 0 = unlimited
    StrictMode   bool
}

func WithBaseIRI(base string) DecoderOption { ... }
func WithMaxDepth(depth int) DecoderOption { ... }
func WithStrictMode() DecoderOption { ... }

func NewTripleDecoder(r io.Reader, format TripleFormat, opts ...DecoderOption) (TripleDecoder, error)
```

**Option 2: Config Struct (Alternative)**

```go
type DecoderConfig struct {
    BaseIRI     string
    MaxDepth     int
    MaxLineLen   int
    // ... other options
}

func NewTripleDecoderWithConfig(r io.Reader, format TripleFormat, cfg *DecoderConfig) (TripleDecoder, error)
```

**Rationale**: Options allow incremental addition of features without breaking changes. Start with safe defaults (unlimited), add limits as needed.

**Format Detection Addition:**

```go
type Format interface {
    IsTriple() bool
    IsQuad() bool
    String() string
}

func DetectFormat(r io.Reader) (Format, error)
```

---

## 3) Standards & Semantic Correctness Review

### What Appears Correct

- **RDF Term Modeling**: `IRI`, `BlankNode`, `Literal`, `TripleTerm` correctly model RDF terms. `Term` interface is clean.
- **RDF-star Support**: `TripleTerm` allows quoted triples as subjects/objects. Implementation looks correct.
- **N-Triples/N-Quads**: Strict validation (absolute IRIs required, proper escaping) appears aligned with W3C specs.
- **Turtle/TriG**: Prefix handling, base IRI resolution, predicate-object lists seem correct.

### Likely Spec Mismatches or Ambiguous Areas

1. **JSON-LD Incomplete**: 
   - Missing `@nest`, `@container`, `@reverse`, `@index` support
   - No `@list` expansion
   - No remote context loading (security risk if added without limits)
   - Context merging is simplistic (no cycle detection)
   - No `@base` handling

2. **RDF/XML Entity Expansion**: 
   - Uses Go's `xml.Decoder` which disables external entities by default (good), but no explicit `Strict` mode configuration
   - No protection against billion laughs via internal entities
   - Should document that external entities are disabled

3. **Turtle Base IRI Resolution**: 
   - `resolveIRI()` function has fallback logic that may not match RFC 3986 exactly in edge cases
   - No validation that resolved IRIs are valid

4. **Blank Node Scoping**: 
   - Blank nodes in Turtle/TriG may not be correctly scoped across graph boundaries
   - Need to verify blank node identity preservation

5. **Literal Datatype Handling**: 
   - No validation that datatype IRIs are valid
   - No canonicalization of numeric literals (e.g., `"1"^^xsd:integer` vs `"01"^^xsd:integer`)

**Recommendation**: Add explicit spec version claims in documentation:
- "Turtle: W3C Turtle 1.1 (2014)"
- "JSON-LD: Partial JSON-LD 1.1 (toRdf only, no framing/compaction)"
- "RDF/XML: RDF/XML Syntax (2004), external entities disabled"

---

## 4) Streaming/Performance Review

### Allocation Hotspots

1. **JSON-LD**: Entire document loaded into `[]Triple`. For 1M triples, this is ~100MB+ of memory.
   ```go
   // jsonld.go:43-77
   func (d *jsonldTripleDecoder) load(r io.Reader) error {
       var data interface{}
       json.NewDecoder(r).Decode(&data)  // Loads entire JSON
       // ... processes everything into []Triple
       d.triples = make([]Triple, len(quads))  // Allocates all at once
   }
   ```

2. **Turtle/TriG Pending Buffer**: `pending []Triple` can grow unbounded for predicate-object lists. Should have a limit.

3. **RDF/XML Queue**: `queue []Triple` can grow large for deeply nested structures.

4. **String Allocations**: Heavy use of `strings.TrimSpace`, `strings.HasPrefix`, `fmt.Sprintf` in hot paths.

### Streaming Gaps

**Critical: JSON-LD Must Stream**

Current implementation:
```go
func (d *jsonldTripleDecoder) Next() (Triple, error) {
    // Just returns from pre-loaded slice
    if d.index >= len(d.triples) {
        return Triple{}, io.EOF
    }
    return d.triples[d.index], nil
}
```

**Required Refactor**: Implement streaming JSON-LD parser that:
- Uses `json.Decoder` token-by-token
- Processes `@graph` arrays incrementally
- Emits triples as nodes are parsed
- Handles `@context` incrementally (with depth limits)

**Example Structure:**
```go
type jsonldStreamDecoder struct {
    dec     *json.Decoder
    stack   []jsonldContext
    current *jsonldNodeBuilder
    // ... state for streaming
}

func (d *jsonldStreamDecoder) Next() (Triple, error) {
    // Parse one node at a time, emit triples immediately
}
```

### Concrete Performance Improvements

1. **Reduce String Allocations**:
   - Use `[]byte` for line parsing where possible
   - Pool `strings.Builder` instances
   - Avoid `fmt.Sprintf` in hot paths (use `strconv` directly)

2. **Bounded Buffers**:
   ```go
   const maxPendingTriples = 1000
   if len(d.pending) > maxPendingTriples {
       return nil, fmt.Errorf("too many pending triples (max %d)", maxPendingTriples)
   }
   ```

3. **Line Length Limits**:
   ```go
   const maxLineLength = 1 << 20 // 1MB
   line, err := d.readLine()
   if len(line) > maxLineLength {
       return Triple{}, fmt.Errorf("line too long: %d bytes (max %d)", len(line), maxLineLength)
   }
   ```

4. **Profile-Guided Optimizations**:
   - Add benchmarks for each format
   - Use `go test -benchmem` to track allocations
   - Consider `sync.Pool` for frequently allocated types

---

## 5) Error Handling & Diagnostics

### Current State

Errors are minimal and unhelpful:
```go
// ntriples.go:621
func (c *ntCursor) errorf(format string, args ...interface{}) error {
    return fmt.Errorf("ntriples: "+format, args...)
}
// Produces: "ntriples: unexpected end of line"
```

**Problems:**
- No line number
- No column position
- No context excerpt (surrounding text)
- No file/stream identifier
- Generic messages don't help debug real files

### Recommended Error Types

```go
type ParseError struct {
    Format    string // "ntriples", "turtle", etc.
    Line      int    // 1-based line number
    Column    int    // 1-based column (byte offset)
    Offset    int64  // Byte offset in stream
    Message   string
    Context   string // Excerpt of input around error
    Err       error  // Wrapped error
}

func (e *ParseError) Error() string {
    if e.Line > 0 {
        return fmt.Sprintf("%s:%d:%d: %s", e.Format, e.Line, e.Column, e.Message)
    }
    return fmt.Sprintf("%s: %s", e.Format, e.Message)
}

func (e *ParseError) Unwrap() error { return e.Err }
```

### Implementation Strategy

1. **Track Position in Decoders**:
   ```go
   type ntTripleDecoder struct {
       reader   *bufio.Reader
       lineNum  int
       lineBuf  []byte // Current line for context
       err      error
   }
   ```

2. **Update Cursor to Include Position**:
   ```go
   type ntCursor struct {
       input  string
       pos    int
       line   int
       column int
   }
   ```

3. **Error Construction**:
   ```go
   func (c *ntCursor) errorf(format string, args ...interface{}) error {
       ctx := c.contextExcerpt() // 50 chars before/after
       return &ParseError{
           Format:  "ntriples",
           Line:    c.line,
           Column:  c.column,
           Message: fmt.Sprintf(format, args...),
           Context: ctx,
       }
   }
   ```

4. **Sentinel Errors for Common Cases**:
   ```go
   var (
       ErrUnsupportedFormat = errors.New("unsupported RDF format")
       ErrInvalidIRI        = errors.New("invalid IRI")
       ErrLineTooLong       = errors.New("line exceeds maximum length")
       ErrDepthExceeded     = errors.New("nesting depth exceeded")
   )
   ```

---

## 6) Security & Robustness

### Threat Model Findings

**Critical Vulnerabilities:**

1. **Unbounded Recursion (JSON-LD)**:
   ```go
   // jsonld.go:125-148
   func parseJSONLDNode(node map[string]interface{}, ctx jsonldContext, quads *[]Quad) error {
       ctx = ctx.withContext(node["@context"])  // Recursive context
       // ... processes nested nodes
       // NO DEPTH LIMIT
   }
   ```
   **Attack**: Deeply nested `@context` or nodes can cause stack overflow.

2. **Unbounded Line Length**:
   ```go
   // ntriples.go:87-95
   func (d *ntTripleDecoder) readLine() (string, error) {
       line, err := d.reader.ReadString('\n')  // No size limit
   }
   ```
   **Attack**: Single line with 1GB of data exhausts memory.

3. **Unbounded Memory Growth (Turtle pending buffer)**:
   ```go
   // turtle.go:20
   pending []Triple // Can grow without bound
   ```
   **Attack**: Predicate-object list with 1M objects allocates huge slice.

4. **RDF/XML Entity Expansion**:
   - Go's `xml.Decoder` disables external entities (good)
   - But internal entities can still cause expansion
   - No explicit limits on entity expansion depth/size

5. **JSON-LD Context Explosion**:
   ```go
   // jsonld.go:89-106
   func (c jsonldContext) withContext(raw interface{}) jsonldContext {
       // No cycle detection
       // No limit on context size
   }
   ```
   **Attack**: Circular context references or huge context maps.

6. **No Timeouts**: 
   - `context.Context` is passed to `ParseTriples` but not checked in format-specific decoders
   - Long-running parses can't be cancelled mid-stream

### Recommended Limits & Safe Defaults

```go
const (
    // Safe defaults for untrusted input
    DefaultMaxDepth      = 100
    DefaultMaxLineLength = 1 << 20      // 1MB
    DefaultMaxTriples    = 10_000_000   // 10M triples
    DefaultMaxContextDepth = 10         // JSON-LD context nesting
    DefaultMaxPendingTriples = 1000     // Turtle/TriG buffer
)

type SecurityLimits struct {
    MaxDepth          int
    MaxLineLength     int
    MaxTriples        int64
    MaxContextDepth   int
    MaxPendingTriples int
    Timeout           time.Duration
}
```

**Implementation Points:**

1. **Add Limits to All Decoders**:
   ```go
   type ntTripleDecoder struct {
       reader      *bufio.Reader
       lineNum     int
       tripleCount int64
       limits      *SecurityLimits
   }
   
   func (d *ntTripleDecoder) Next() (Triple, error) {
       if d.tripleCount >= d.limits.MaxTriples {
           return Triple{}, fmt.Errorf("max triples exceeded: %d", d.limits.MaxTriples)
       }
       // ... parse with line length check
   }
   ```

2. **Context Cancellation in Decoders**:
   ```go
   func (d *ntTripleDecoder) Next() (Triple, error) {
       // Check context in hot loop
       select {
       case <-d.ctx.Done():
           return Triple{}, d.ctx.Err()
       default:
       }
       // ... parse
   }
   ```

3. **Fuzz Tests**:
   ```go
   func FuzzNTriplesDecode(f *testing.F) {
       f.Add("<http://example.org/s> <http://example.org/p> \"v\" .\n")
       f.Fuzz(func(t *testing.T, data []byte) {
           dec, err := NewTripleDecoder(bytes.NewReader(data), TripleFormatNTriples)
           if err != nil {
               return
           }
           for i := 0; i < 1000; i++ { // Limit iterations
               _, err := dec.Next()
               if err != nil {
                   break
               }
           }
       })
   }
   ```

---

## 7) Architecture & Package Structure

### Current Structure

```
rdf/
  - model.go          (Term types)
  - format.go         (Format constants)
  - parser.go          (ParseTriples/ParseQuads helpers)
  - encoder.go         (New*Decoder/Encoder constructors)
  - errors.go          (ErrUnsupportedFormat)
  - ntriples.go        (N-Triples/N-Quads implementation)
  - turtle.go          (Turtle/TriG implementation)
  - jsonld.go          (JSON-LD implementation)
  - rdfxml.go          (RDF/XML implementation)
  - *_test.go          (Tests)
```

**Assessment**: Clean, flat structure. No unnecessary nesting.

### Modularization Improvements

**Option 1: Keep Flat (Recommended for Now)**

The current structure is fine. Only consider splitting if:
- Format implementations exceed 1000 lines (none do currently)
- Shared utilities emerge (lexer, IRI resolution)

**Option 2: Format Subpackages (If Growth Needed)**

```
rdf/
  - format.go
  - model.go
  - parser.go
  - encoder.go
  - ntriples/
    - decoder.go
    - encoder.go
  - turtle/
    - decoder.go
    - encoder.go
  - jsonld/
    - decoder.go
    - encoder.go
```

**Recommendation**: Stay flat. The formats are tightly coupled (shared `Term` types), and subpackages add import complexity without clear benefit.

### What to Internalize vs Export

**Currently Exported (Review):**

- ✅ `Term`, `IRI`, `BlankNode`, `Literal`, `TripleTerm`, `Triple`, `Quad` — **Correctly exported** (core model)
- ✅ `TripleFormat`, `QuadFormat` — **Correctly exported** (user-facing)
- ✅ `TripleDecoder`, `QuadDecoder`, `TripleEncoder`, `QuadEncoder` — **Correctly exported** (interfaces)
- ✅ `NewTripleDecoder`, `NewQuadDecoder`, etc. — **Correctly exported** (constructors)
- ✅ `ParseTriples`, `ParseQuads` — **Correctly exported** (helpers)

**Nothing needs to be unexported.** The API surface is appropriately minimal.

### Internal Utilities

The `internal/` directory exists but appears unused. Consider moving shared utilities there if they emerge:
- IRI validation/normalization
- Common lexing utilities
- Error formatting helpers

---

## 8) Tests & Conformance Plan

### Current Test Coverage

**Strengths:**
- W3C test suite integration (`compliance_test.go`)
- Format-specific tests (`*_test.go` files)
- Examples in `examples_test.go`
- Coverage tests (`coverage_*_test.go`)

**Gaps:**

1. **No Fuzz Tests**: Critical for parser security.
2. **No Roundtrip Tests**: Encode → Decode → Compare equivalence.
3. **No Cross-Format Tests**: Parse Turtle → Encode N-Triples → Verify.
4. **No Malformed Input Tests**: Explicit tests for malicious inputs.
5. **No Performance Regression Tests**: No benchmarks in CI.

### Missing Test Categories

1. **Fuzz Tests** (Critical):
   ```go
   func FuzzNTriplesDecode(f *testing.F) { ... }
   func FuzzTurtleDecode(f *testing.F) { ... }
   func FuzzJSONLDDecode(f *testing.F) { ... }
   func FuzzRDFXMLDecode(f *testing.F) { ... }
   ```

2. **Roundtrip Tests**:
   ```go
   func TestRoundtripNTriples(t *testing.T) {
       input := "<http://ex/s> <http://ex/p> \"v\" .\n"
       dec, _ := NewTripleDecoder(strings.NewReader(input), TripleFormatNTriples)
       triple, _ := dec.Next()
       
       var buf bytes.Buffer
       enc, _ := NewTripleEncoder(&buf, TripleFormatNTriples)
       enc.Write(triple)
       enc.Close()
       
       // Parse output and compare
       dec2, _ := NewTripleDecoder(&buf, TripleFormatNTriples)
       triple2, _ := dec2.Next()
       if !triplesEqual(triple, triple2) {
           t.Errorf("roundtrip failed")
       }
   }
   ```

3. **Security Limit Tests**:
   ```go
   func TestMaxLineLength(t *testing.T) {
       longLine := strings.Repeat("a", 2<<20) + " .\n"
       dec, err := NewTripleDecoder(strings.NewReader(longLine), TripleFormatNTriples, WithMaxLineLength(1<<20))
       // Should error on first Next()
   }
   ```

4. **Error Message Tests**:
   ```go
   func TestErrorIncludesLineNumber(t *testing.T) {
       input := "valid line\ninvalid line without dot\n"
       dec, _ := NewTripleDecoder(strings.NewReader(input), TripleFormatNTriples)
       _, err := dec.Next() // Skip first
       _, err = dec.Next()  // Should error
       var parseErr *ParseError
       if errors.As(err, &parseErr) {
           if parseErr.Line != 2 {
               t.Errorf("expected line 2, got %d", parseErr.Line)
           }
       }
   }
   ```

### Recommended Conformance Suites

1. **W3C Test Suites** (Already Integrated):
   - ✅ N-Triples, N-Quads, Turtle, TriG, RDF/XML
   - ⚠️ JSON-LD (partial — only toRdf, not expand/compact/flatten)

2. **RDF-star Tests**:
   - `w3c-tests/rdf-star-tests/` exists — ensure full coverage

3. **Golden Files**:
   - Create `testdata/` with known-good inputs/outputs
   - Use `go-cmp` for comparison

4. **Cross-Format Equivalence**:
   - Parse format A → encode format B → parse format B → compare

---

## 9) Concrete Refactor Suggestions

### Refactor 1: Add Position Tracking to Errors

**Before:**
```go
func (c *ntCursor) errorf(format string, args ...interface{}) error {
    return fmt.Errorf("ntriples: "+format, args...)
}
```

**After:**
```go
type ntCursor struct {
    input  string
    pos    int
    line   int    // NEW
    column int    // NEW
}

func (c *ntCursor) errorf(format string, args ...interface{}) error {
    ctx := c.contextExcerpt(50) // 50 chars before/after
    return &ParseError{
        Format:  "ntriples",
        Line:    c.line,
        Column:  c.column,
        Message: fmt.Sprintf(format, args...),
        Context: ctx,
    }
}

func (c *ntCursor) contextExcerpt(n int) string {
    start := max(0, c.pos-n)
    end := min(len(c.input), c.pos+n)
    return c.input[start:end]
}
```

### Refactor 2: Add Security Limits

**Before:**
```go
func newNTriplesTripleDecoder(r io.Reader) TripleDecoder {
    return &ntTripleDecoder{reader: bufio.NewReader(r)}
}
```

**After:**
```go
type DecoderConfig struct {
    MaxLineLength int
    MaxTriples    int64
    MaxDepth      int
    // ... other limits
}

func newNTriplesTripleDecoder(r io.Reader, cfg *DecoderConfig) TripleDecoder {
    if cfg == nil {
        cfg = &DecoderConfig{
            MaxLineLength: DefaultMaxLineLength,
            MaxTriples:    DefaultMaxTriples,
        }
    }
    return &ntTripleDecoder{
        reader:      bufio.NewReader(r),
        limits:      cfg,
        lineNum:     1,
        tripleCount: 0,
    }
}

func (d *ntTripleDecoder) readLine() (string, error) {
    line, err := d.reader.ReadString('\n')
    if err != nil && err != io.EOF {
        return "", err
    }
    if len(line) > d.limits.MaxLineLength {
        return "", &ParseError{
            Format:  "ntriples",
            Line:    d.lineNum,
            Message: fmt.Sprintf("line too long: %d bytes (max %d)", len(line), d.limits.MaxLineLength),
        }
    }
    d.lineNum++
    return line, nil
}
```

### Refactor 3: Stream JSON-LD

**Before:**
```go
func (d *jsonldTripleDecoder) load(r io.Reader) error {
    var data interface{}
    json.NewDecoder(r).Decode(&data)  // Loads everything
    // ... process into []Triple
    d.triples = make([]Triple, len(quads))
    return nil
}
```

**After:**
```go
type jsonldStreamDecoder struct {
    dec     *json.Decoder
    stack   []jsonldContext
    pending []Triple
    limits  *DecoderConfig
    depth   int
}

func (d *jsonldStreamDecoder) Next() (Triple, error) {
    if len(d.pending) > 0 {
        t := d.pending[0]
        d.pending = d.pending[1:]
        return t, nil
    }
    
    // Parse one token at a time
    tok, err := d.dec.Token()
    if err != nil {
        return Triple{}, err
    }
    
    // Process token, emit triples incrementally
    // ... streaming logic
}
```

### Refactor 4: Context-Aware Decoders

**Before:**
```go
func ParseTriples(ctx context.Context, r io.Reader, format TripleFormat, handler TripleHandler) error {
    decoder, err := NewTripleDecoder(r, format)
    // ... ctx not passed to decoder
}
```

**After:**
```go
type TripleDecoder interface {
    Next() (Triple, error)
    Err() error
    Close() error
    SetContext(ctx context.Context)  // NEW
}

func ParseTriples(ctx context.Context, r io.Reader, format TripleFormat, handler TripleHandler) error {
    decoder, err := NewTripleDecoder(r, format)
    if ctx != nil {
        decoder.SetContext(ctx)  // Pass context to decoder
    }
    // ... rest
}

func (d *ntTripleDecoder) Next() (Triple, error) {
    if d.ctx != nil {
        select {
        case <-d.ctx.Done():
            return Triple{}, d.ctx.Err()
        default:
        }
    }
    // ... parse
}
```

### Ideal Usage Snippet (After Refactors)

```go
// Safe, streaming, with limits
cfg := &rdf.DecoderConfig{
    MaxLineLength: 1 << 20,  // 1MB
    MaxTriples:    10_000_000,
    MaxDepth:      100,
}

ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

dec, err := rdf.NewTripleDecoder(reader, rdf.TripleFormatNTriples, rdf.WithConfig(cfg))
if err != nil {
    return err
}
dec.SetContext(ctx)
defer dec.Close()

for {
    triple, err := dec.Next()
    if err == io.EOF {
        break
    }
    if err != nil {
        var parseErr *rdf.ParseError
        if errors.As(err, &parseErr) {
            log.Printf("Parse error at %s:%d:%d: %s\nContext: %s",
                parseErr.Format, parseErr.Line, parseErr.Column,
                parseErr.Message, parseErr.Context)
        }
        return err
    }
    // Process triple
}
```

---

## 10) Scorecard (1–5)

| Category | Score | Notes |
|----------|-------|-------|
| **API Elegance** | 4/5 | Clean interfaces, type safety excellent. Missing options pattern and format detection. |
| **Go Idiomaticity** | 4/5 | Interfaces, error handling patterns are good. Missing `%w` error wrapping in some places. |
| **Standards Correctness** | 3/5 | Core formats appear correct. JSON-LD incomplete, missing spec version claims. |
| **Streaming/Performance** | 2/5 | JSON-LD not streaming. Some allocation hotspots. No performance guarantees documented. |
| **Robustness/Security** | 1/5 | **Critical**: No limits, no fuzzing, vulnerable to DoS. Not safe for untrusted input. |
| **Test Quality** | 3/5 | W3C tests integrated, but missing fuzz tests, roundtrip tests, security tests. |
| **Maintainability** | 4/5 | Clean structure, good separation. Error messages need improvement for debugging. |

**Overall: 3.0/5.0** — Good foundation, but needs security hardening and streaming completeness before production use.

---

## Priority Action Items

### Critical (Must Fix Before Production)

1. ✅ **Add security limits** (max depth, line length, triple count, timeouts)
2. ✅ **Implement streaming JSON-LD** (or document that it's not supported)
3. ✅ **Add fuzz tests** for all parsers
4. ✅ **Improve error messages** (line numbers, context)

### High Priority

5. ✅ **Add functional options** for decoder/encoder configuration
6. ✅ **Add context cancellation** to all decoders
7. ✅ **Add roundtrip tests** (encode → decode → compare)
8. ✅ **Document security limits** in README

### Medium Priority

9. ✅ **Complete JSON-LD support** (or document limitations)
10. ✅ **Add format detection** function
11. ✅ **Performance benchmarks** in CI
12. ✅ **Cross-format equivalence tests**

### Low Priority

13. ✅ **Consider subpackages** if formats grow >1000 lines
14. ✅ **Add golden file tests** for known-good inputs
15. ✅ **Document spec versions** explicitly

---

## Conclusion

`rdf-go` has a **solid architectural foundation** with clean APIs and good streaming support for most formats. However, **critical security gaps** (no limits, no fuzzing) and **streaming incompleteness** (JSON-LD) prevent it from being production-ready for untrusted inputs.

**Recommendation**: Address the Critical and High Priority items before v1.0. The library is close to reference-quality but needs hardening.

**Estimated Effort**: 
- Critical items: 2-3 weeks
- High priority: 1-2 weeks
- Medium/Low: Ongoing

The codebase shows good engineering practices and is well-positioned to become a reference implementation with focused effort on security and completeness.

