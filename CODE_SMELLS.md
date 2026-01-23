# Code Smells Analysis - rdf-go

This document identifies code smells across the entire codebase, prioritized by severity and impact.

## 🔴 High Priority Code Smells

### 1. **Massive Code Duplication: String Escape Sequence Parsing**
**Location:** `rdf/ntriples.go`, `rdf/turtle.go`, `rdf/turtle_parser.go`

**Problem:** The same escape sequence parsing logic is duplicated in at least 3 places:
- `ntCursor.parseLiteral()` (lines 413-538)
- `turtleCursor.parseLiteralWithQuote()` (lines 784-1042) 
- `turtleParser.unescapeString()` (lines 594-626) - *This one is refactored, but others aren't*

**Impact:** 
- Bug fixes must be applied in multiple places
- Inconsistent behavior between parsers
- Maintenance burden

**Recommendation:** Extract shared escape sequence parsing into `rdf/parse_utils.go` as a reusable function.

---

### 2. **Magic Numbers: Unicode Surrogate Pair Constants**
**Location:** `rdf/turtle.go` (lines 845, 850, 853, 858, 1020, 1025, 1028, 1033)

**Problem:** Hardcoded Unicode surrogate pair values (`0xD800`, `0xDBFF`, `0xDC00`, `0xDFFF`, `0x10000`) appear in `turtle.go` even though constants were extracted to the top of the file. The cursor parser still uses hardcoded values.

**Impact:** 
- Inconsistent use of constants vs magic numbers
- Harder to understand code intent
- Risk of typos

**Recommendation:** Replace all hardcoded values in `turtleCursor.parseLiteralWithQuote()` with the constants defined at the top of `turtle.go`.

---

### 3. **Duplicate Statement Building Logic**
**Location:** `rdf/turtle_parser.go`, `rdf/trig_decoder.go`

**Problem:** Both parsers build statements using `strings.Builder` with similar patterns:
- Check for EOF
- Handle directives
- Accumulate lines
- Check for statement completion

**Impact:**
- Code duplication
- Potential for divergence in behavior

**Recommendation:** Extract shared statement building logic into a helper function.

---

### 4. **Duplicate Statement Completion Detection**
**Location:** `rdf/turtle_parse_helpers.go` (lines 64-163, 165-316)

**Problem:** `isStatementComplete()` and `splitTurtleStatements()` contain nearly identical logic for tracking:
- String state (inString, stringQuote, longString)
- IRI state (inIRI)
- Bracket/paren/annotation depth

**Impact:**
- ~150 lines of duplicated state tracking logic
- Maintenance burden
- Risk of bugs when one is updated but not the other

**Recommendation:** Extract shared state tracking into a reusable struct/helper.

---

## 🟡 Medium Priority Code Smells

### 5. **Long Functions with High Cyclomatic Complexity**
**Location:** Multiple files

**Examples:**
- `rdf/turtle.go:parseLiteralWithQuote()` - ~260 lines, complex nested conditionals
- `rdf/ntriples.go:parseLiteral()` - ~140 lines, many switch cases
- `rdf/turtle_parse_helpers.go:splitTurtleStatements()` - ~150 lines, complex state machine
- `rdf/trig_decoder.go:Next()` - ~200+ lines, deeply nested conditionals

**Impact:**
- Hard to test
- Hard to understand
- High risk of bugs

**Recommendation:** Break down into smaller, focused functions.

---

### 6. **Inconsistent Error Wrapping**
**Location:** `rdf/ntriples.go`, `rdf/trig_decoder.go`, `rdf/rdfxml.go`

**Problem:** 
- `ntriples.go` uses `WrapParseError()` directly (line 45, 93)
- `trig_decoder.go` uses `fmt.Errorf()` directly (lines 101, 122, 133)
- `turtle_parser.go` uses `wrapParseError()` method (standardized)

**Impact:**
- Inconsistent error messages
- Missing context in some parsers
- Harder debugging

**Recommendation:** Standardize error wrapping across all parsers.

---

### 7. **Hardcoded IRI Strings**
**Location:** Multiple files

**Problem:** Magic strings for RDF namespaces appear throughout:
- `rdf/ntriples.go:531-533` - Hardcoded `langString` and `dirLangString` IRIs
- `rdf/rdfxml.go:502` - Hardcoded `XMLLiteral` IRI
- `rdf/compliance_test.go:919` - Hardcoded `xsdBoolean` IRI

**Impact:**
- Typos risk
- Hard to maintain
- Inconsistent with centralized constants in `jsonld.go`

**Recommendation:** Extract all RDF namespace IRIs to constants in a shared location (e.g., `rdf/constants.go`).

---

### 8. **Duplicate Hex Digit Parsing Logic**
**Location:** `rdf/ntriples.go`, `rdf/turtle.go`, `rdf/turtle_parser.go`

**Problem:** The same hex digit parsing logic appears in multiple escape sequence handlers:
```go
if hex >= '0' && hex <= '9' {
    digit = int(hex - '0')
} else if hex >= 'a' && hex <= 'f' {
    digit = int(hex - 'a' + 10)
} else if hex >= 'A' && hex <= 'F' {
    digit = int(hex - 'A' + 10)
}
```

**Impact:**
- Code duplication
- Maintenance burden

**Recommendation:** Extract to `parseHexDigit(byte) (int, bool)` helper in `parse_utils.go`.

---

### 9. **Complex Nested Conditionals in TriG Parser**
**Location:** `rdf/trig_decoder.go:Next()` (lines 41-200+)

**Problem:** Deeply nested conditionals handling:
- Pending quads
- Context checks
- Statement building
- Directive handling
- Graph block handling
- Inline graph parsing

**Impact:**
- Hard to follow control flow
- High cyclomatic complexity
- Difficult to test edge cases

**Recommendation:** Extract methods for each major concern (directive handling, graph block parsing, etc.).

---

### 10. **Inconsistent Blank Node ID Generation**
**Location:** Multiple parsers

**Problem:** Different patterns for generating blank node IDs:
- `turtleParser.newBlankNode()` - Uses counter: `fmt.Sprintf("b%d", counter)`
- `rdfxmlTripleDecoder.newBlankNode()` - Uses counter: `fmt.Sprintf("b%d", d.blankIDGen)`
- `turtleCursor.newBlankNode()` - Similar pattern

**Impact:**
- Potential ID collisions if parsers are used together
- Inconsistent naming

**Recommendation:** Standardize blank node ID generation (consider UUIDs or namespaced IDs).

---

## 🟢 Low Priority Code Smells

### 11. **Dead/Unused Code**
**Location:** `rdf/turtle.go`

**Problem:** The `turtleCursor` struct and its methods are largely unused after the token-based parser refactoring, but still present in the codebase.

**Impact:**
- Code bloat
- Confusion about which parser to use
- Maintenance burden

**Recommendation:** Remove unused cursor-based parser code, or document it as a fallback.

---

### 12. **Magic Strings in Directive Detection**
**Location:** `rdf/turtle_parser.go:isLikelyDirective()`, `rdf/trig_decoder.go`

**Problem:** Hardcoded directive strings like `"@prefix"`, `"@base"`, `"PREFIX"`, etc.

**Impact:**
- Minor maintenance issue
- Could use constants for consistency

**Recommendation:** Extract directive strings to constants.

---

### 13. **Inconsistent Context Checking**
**Location:** Multiple parsers

**Problem:** Different patterns for context checking:
- `turtleParser` uses `checkDecodeContext()`
- `trigQuadDecoder` uses `checkContext()` method
- `jsonldTripleDecoder` uses `checkJSONLDContext()`

**Impact:**
- Minor inconsistency
- Could be unified

**Recommendation:** Standardize context checking pattern.

---

### 14. **Repeated String Trim Operations**
**Location:** Multiple parsers

**Problem:** `strings.TrimSpace()` called multiple times on the same strings in parsing loops.

**Impact:**
- Minor performance issue
- Code clutter

**Recommendation:** Trim once at entry points.

---

### 15. **Long Parameter Lists**
**Location:** Various functions

**Problem:** Some functions have 4+ parameters, making them hard to call and test.

**Examples:**
- `parseTurtleStatement(prefixes, baseIRI, allowQuoted, debugStatements, line)`
- Various parsing functions with many options

**Impact:**
- Harder to use
- Harder to test
- Risk of parameter order mistakes

**Recommendation:** Use option structs or functional options pattern.

---

## Summary Statistics

- **High Priority:** 4 code smells (critical duplication and inconsistency)
- **Medium Priority:** 6 code smells (maintainability and quality issues)
- **Low Priority:** 5 code smells (minor improvements)

**Total:** 15 identified code smells

## Recommended Action Plan

1. **Phase 1 (High Priority):**
   - Extract shared escape sequence parsing
   - Replace magic numbers with constants in turtle.go
   - Extract shared statement building logic

2. **Phase 2 (Medium Priority):**
   - Standardize error wrapping
   - Extract hex digit parsing
   - Refactor long functions

3. **Phase 3 (Low Priority):**
   - Clean up dead code
   - Extract constants
   - Standardize patterns

