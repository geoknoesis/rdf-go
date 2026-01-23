# Additional Code Smells and Improvements - rdf-go

This document identifies additional code smells and improvement opportunities beyond those already addressed.

## 🔴 High Priority Additional Issues

### 1. **Cursor Parser Still Has Duplicate Escape Sequence Parsing**
**Location:** `rdf/turtle.go:parseLiteralWithQuote()` (lines 778-932), `rdf/turtle.go:parseLongLiteral()` (lines 935-1100)

**Problem:** The `turtleCursor` parser still contains inline escape sequence parsing logic that duplicates `UnescapeString()` functionality. While `turtleCursor` is still used (by `parseTurtleTripleLine` which is called from `trig_decoder.go`), it should use the shared `UnescapeString()` function.

**Impact:**
- ~300 lines of duplicate escape handling code
- Bug fixes must be applied in multiple places
- Inconsistent behavior risk

**Recommendation:** Refactor `turtleCursor.parseLiteralWithQuote()` and `parseLongLiteral()` to use `UnescapeString()` from `parse_utils.go`.

---

### 2. **Redundant Statement Length Checks**
**Location:** `rdf/trig_decoder.go:Next()` (multiple occurrences)

**Problem:** `statement.Len() == 0` is checked multiple times in the same function:
- Line 70: `if statement.Len() == 0`
- Line 82: `if statement.Len() == 0`
- Line 88: `if statement.Len() == 0 && isTrigDirectiveLine(trimmed)`
- Line 105: `if statement.Len() == 0 && d.handleDirective(trimmed)`
- Line 159: `if statement.Len() > 0`

**Impact:**
- Code clutter
- Minor performance overhead (repeated method calls)
- Harder to read

**Recommendation:** Extract `isEmpty := statement.Len() == 0` at the start of the loop iteration and reuse.

---

### 3. **Long Parameter List: parseTurtleStatement**
**Location:** `rdf/turtle.go:parseTurtleStatement()` (5 parameters)

**Problem:** Function signature:
```go
func parseTurtleStatement(prefixes map[string]string, baseIRI string, allowQuoted bool, debugStatements bool, line string) ([]Triple, error)
```

**Impact:**
- Hard to call correctly
- Easy to mix up parameter order
- Hard to extend with new options
- Used in multiple places (trig_decoder.go calls it twice)

**Recommendation:** Create a `TurtleParseOptions` struct:
```go
type TurtleParseOptions struct {
    Prefixes    map[string]string
    BaseIRI     string
    AllowQuoted bool
    DebugStatements bool
}
```

---

## 🟡 Medium Priority Additional Issues

### 4. **Very Long Function: trigQuadDecoder.Next()**
**Location:** `rdf/trig_decoder.go:Next()` (~210 lines)

**Problem:** This function handles too many concerns:
- Pending quad management
- Context checking
- Statement building
- Directive handling
- Graph block parsing
- Inline graph handling
- Statement parsing
- Error handling

**Impact:**
- High cyclomatic complexity (~15+)
- Difficult to test individual concerns
- Hard to understand control flow
- Risk of bugs

**Recommendation:** Extract methods:
- `buildStatement()` - handles statement accumulation
- `handleGraphBlock()` - handles graph block logic
- `processStatement()` - handles statement parsing and quad generation
- `handleDirectiveOrGraph()` - handles directive and graph block detection

---

### 5. **Very Long Function: turtleCursor.parseLiteralWithQuote()**
**Location:** `rdf/turtle.go:parseLiteralWithQuote()` (~150 lines)

**Problem:** This function:
- Parses escape sequences inline (should use UnescapeString)
- Handles language tags
- Handles datatypes
- Has complex nested conditionals

**Impact:**
- Hard to test
- Duplicates escape logic
- High complexity

**Recommendation:** 
- Use `UnescapeString()` for escape handling
- Extract language tag parsing to helper
- Extract datatype parsing to helper

---

### 6. **Very Long Function: turtleCursor.parseLongLiteral()**
**Location:** `rdf/turtle.go:parseLongLiteral()` (~165 lines)

**Problem:** Similar to `parseLiteralWithQuote()` but for triple-quoted strings. Contains duplicate escape sequence parsing.

**Impact:**
- Code duplication
- Maintenance burden

**Recommendation:** Refactor to use `UnescapeString()` and extract helpers.

---

### 7. **Repeated String Building Pattern**
**Location:** `rdf/trig_decoder.go`, `rdf/turtle_parser.go`

**Problem:** Both parsers build statements with similar patterns but slight differences:
- Both use `strings.Builder`
- Both check for EOF
- Both handle directives
- Both check statement completion
- But implementation details differ

**Impact:**
- Code duplication
- Potential for divergence

**Recommendation:** Extract shared statement building logic into a helper struct/function that can be configured for differences.

---

### 8. **Inefficient String Operations in Loops**
**Location:** `rdf/trig_decoder.go:Next()`

**Problem:** 
- `strings.TrimSpace(statement.String())` called multiple times in the same loop
- `statement.String()` creates a new string allocation each time
- Line 197: `stmt := strings.TrimSpace(statement.String())`
- Line 208: `line := strings.TrimSpace(statement.String())`

**Impact:**
- Unnecessary allocations
- Performance overhead in tight loops

**Recommendation:** Build the trimmed string once and reuse.

---

### 9. **Magic String: "GRAPH "**
**Location:** `rdf/trig_decoder.go:parseGraphToken()` (line 275)

**Problem:** Hardcoded string `"GRAPH "` used for parsing graph tokens.

**Impact:**
- Should use the `directiveGraph` constant we just added
- Minor inconsistency

**Recommendation:** Use `directiveGraph + " "` constant.

---

### 10. **Complex Conditional Logic in splitTurtleStatements**
**Location:** `rdf/turtle_parse_helpers.go:splitTurtleStatements()` (~110 lines)

**Problem:** Complex nested conditionals for determining statement boundaries, especially around the period (`.`) character with many edge cases.

**Impact:**
- Hard to understand
- High cyclomatic complexity
- Risk of bugs

**Recommendation:** Extract helper functions:
- `isStatementTerminator()` - checks if a period is a terminator
- `handlePeriod()` - handles period character logic
- Simplify the main loop

---

## 🟢 Low Priority Additional Issues

### 11. **Inconsistent Error Message Formatting**
**Location:** Multiple files

**Problem:** Error messages use different formats:
- Some use `fmt.Errorf("message")`
- Some use `c.errorf("message")`
- Some use `p.wrapParseError("", fmt.Errorf("message"))`
- Error messages don't always include context

**Impact:**
- Inconsistent user experience
- Harder debugging

**Recommendation:** Standardize error message format and ensure all include relevant context.

---

### 12. **Missing Input Validation**
**Location:** Various parser entry points

**Problem:** Some parsers don't validate input before processing:
- No check for nil readers
- No check for empty input
- No early validation of format

**Impact:**
- Potential panics
- Wasted processing on invalid input

**Recommendation:** Add input validation at parser entry points.

---

### 13. **Repeated Context Checking**
**Location:** `rdf/trig_decoder.go:Next()`

**Problem:** `d.checkContext()` is called multiple times in nested loops (lines 50, 61).

**Impact:**
- Minor performance overhead
- Code duplication

**Recommendation:** Consider checking context less frequently or at strategic points.

---

### 14. **Magic Numbers in String Length Checks**
**Location:** Multiple files

**Problem:** Hardcoded length checks:
- `if len(lexeme) < 6` (for long strings)
- `if len(lexeme) < 2` (for regular strings)
- `if pos+5 >= len(s)` (for unicode escapes)
- `if pos+9 >= len(s)` (for long unicode escapes)

**Impact:**
- Magic numbers reduce readability
- Hard to understand intent

**Recommendation:** Extract to named constants:
```go
const (
    minLongStringLength = 6  // """..."""
    minStringLength = 2      // "..."
    unicodeEscapeLength = 6  // \uXXXX
    unicodeLongEscapeLength = 10 // \UXXXXXXXX
)
```

---

### 15. **Potential Memory Leak: Expansion Triples**
**Location:** `rdf/turtle_parser.go`, `rdf/turtle.go`

**Problem:** `expansionTriples` slice is reset with `expansionTriples[:0]` but may retain underlying capacity, leading to memory growth over time if not properly managed.

**Impact:**
- Potential memory bloat
- Unnecessary memory retention

**Recommendation:** Consider using `expansionTriples = nil` or ensuring proper capacity management.

---

### 16. **Inconsistent Blank Node Counter Initialization**
**Location:** Multiple parsers

**Problem:** Some parsers initialize `blankNodeCounter` to 0 explicitly, others rely on zero value:
- `turtleParser`: `blankNodeCounter: 0` (explicit)
- `turtleCursor`: `blankNodeCounter: 0` (explicit)
- `rdfxmlTripleDecoder`: `blankIDGen` (zero value)

**Impact:**
- Minor inconsistency
- Could be standardized

**Recommendation:** Use consistent initialization pattern (zero value is fine, but be consistent).

---

### 17. **Complex State Management in TriG Parser**
**Location:** `rdf/trig_decoder.go:Next()`

**Problem:** Multiple state variables managed in the same function:
- `statement` (strings.Builder)
- `graphForStatement` (Term)
- `closeGraphAfter` (bool)
- `hitEOF` (bool)
- `d.graph` (Term)
- `d.inGraphBlock` (bool)

**Impact:**
- Hard to track state transitions
- Risk of state inconsistencies

**Recommendation:** Extract state management into a struct or use a state machine pattern.

---

### 18. **Duplicate Hex Digit Parsing in Cursor Parser**
**Location:** `rdf/turtle.go:parseLiteralWithQuote()`, `parseLongLiteral()`

**Problem:** Still contains inline hex digit parsing that duplicates `decodeUChar()` logic:
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

**Recommendation:** Use `decodeUChar()` or extract to `parseHexDigit()` helper.

---

### 19. **Missing Early Returns**
**Location:** Various functions

**Problem:** Some functions have deep nesting that could be simplified with early returns:
- `rdf/trig_decoder.go:Next()` - deeply nested conditionals
- `rdf/turtle.go:parseLiteralWithQuote()` - nested if-else chains

**Impact:**
- Reduced readability
- Higher cognitive load

**Recommendation:** Use early returns and guard clauses to reduce nesting.

---

### 20. **Inefficient String Concatenation**
**Location:** `rdf/trig_decoder.go:Next()`

**Problem:** 
- Line 210: `line = line + " ."` - string concatenation instead of `strings.Builder`
- Line 228: `stmt = stmt + " ."` - same issue

**Impact:**
- Minor performance issue
- Unnecessary allocations

**Recommendation:** Use `strings.Builder` or `fmt.Sprintf()` for string building.

---

## Summary Statistics

- **High Priority:** 3 additional issues (duplication, redundancy, long parameter lists)
- **Medium Priority:** 7 additional issues (long functions, inefficiencies)
- **Low Priority:** 10 additional issues (minor improvements)

**Total:** 20 additional code smells identified

## Recommended Action Plan

1. **Immediate (High Priority):**
   - Refactor cursor parser to use `UnescapeString()`
   - Extract redundant checks
   - Refactor `parseTurtleStatement` to use options struct

2. **Short Term (Medium Priority):**
   - Break down `trigQuadDecoder.Next()` into smaller functions
   - Optimize string operations in loops
   - Extract magic numbers to constants

3. **Long Term (Low Priority):**
   - Standardize error messages
   - Add input validation
   - Improve state management patterns

