# Error Handling Guide

This document describes error handling strategies, error codes, and recovery patterns in the rdf-go library.

## Error Types

### Error Codes

The library uses structured error codes for programmatic error handling:

- `ErrCodeUnsupportedFormat` - Format not supported
- `ErrCodeLineTooLong` - Line exceeded configured limit
- `ErrCodeStatementTooLong` - Statement exceeded configured limit
- `ErrCodeDepthExceeded` - Nesting depth exceeded configured limit
- `ErrCodeTripleLimitExceeded` - Maximum number of triples/quads exceeded
- `ErrCodeParseError` - General parse error
- `ErrCodeIOError` - I/O error
- `ErrCodeContextCanceled` - Context was canceled
- `ErrCodeInvalidIRI` - Invalid IRI encountered
- `ErrCodeInvalidLiteral` - Invalid literal encountered

### Error Structures

#### ParseError

`ParseError` provides structured context for parse failures:

```go
type ParseError struct {
    Format    string // Format name (e.g., "turtle", "ntriples")
    Statement string // Offending statement or input excerpt
    Line      int    // 1-based line number (0 if unknown)
    Column    int    // 1-based column number (0 if unknown)
    Offset    int    // Byte offset in input (0 if unknown)
    Err       error  // Underlying error
}
```

## Error Recovery Strategies

### 1. Line/Statement Too Long Errors

**Error Code:** `ErrCodeLineTooLong`, `ErrCodeStatementTooLong`

**Recovery:**
- Increase the limit using `OptMaxLineBytes()` or `OptMaxStatementBytes()`
- For untrusted input, use safe defaults from `SafeMode()`
- For trusted input, you can disable limits (not recommended)

**Example:**
```go
opts := []DecodeOption{
    OptMaxLineBytes(2 << 20), // 2MB per line
    OptMaxStatementBytes(8 << 20), // 8MB per statement
}
dec, err := NewReader(reader, FormatTurtle, opts...)
```

### 2. Depth Exceeded Errors

**Error Code:** `ErrCodeDepthExceeded`

**Recovery:**
- Increase nesting depth limit using `OptMaxDepth()`
- Check input for deeply nested structures (collections, blank node lists)
- Consider flattening the input structure if possible

**Example:**
```go
opts := []DecodeOption{
    OptMaxDepth(200), // Allow deeper nesting
}
dec, err := NewReader(reader, FormatTurtle, opts...)
```

### 3. Triple Limit Exceeded Errors

**Error Code:** `ErrCodeTripleLimitExceeded`

**Recovery:**
- Increase the limit using `OptMaxTriples()`
- Process input in chunks if possible
- For unlimited processing, set to 0 (not recommended for untrusted input)

**Example:**
```go
opts := []DecodeOption{
    OptMaxTriples(50_000_000), // 50M triples
}
dec, err := NewReader(reader, FormatTurtle, opts...)
```

### 4. Context Cancellation Errors

**Error Code:** `ErrCodeContextCanceled`

**Recovery:**
- This is not an error condition - it's a cancellation signal
- Check if operation was intentionally canceled
- Clean up resources and return gracefully

**Example:**
```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

err := Parse(ctx, reader, FormatTurtle, handler)
if errors.Is(err, context.Canceled) {
    // Operation was canceled - this is expected
    return nil
}
```

### 5. Invalid IRI Errors

**Error Code:** `ErrCodeInvalidIRI`

**Recovery:**
- Validate IRIs before parsing if possible
- Use `StrictIRIValidation` option for strict validation
- Some parsers may accept invalid IRIs - check format-specific behavior

**Example:**
```go
opts := []DecodeOption{
    OptStrictIRIValidation(true),
}
dec, err := NewReader(reader, FormatTurtle, opts...)
```

### 6. Parse Errors

**Error Code:** `ErrCodeParseError`

**Recovery:**
- Check `ParseError` for format, line, column, and statement context
- Use error message to locate the problem in input
- Fix input syntax and retry
- For streaming parsers, you may be able to continue after an error

**Example:**
```go
var parseErr *ParseError
if errors.As(err, &parseErr) {
    fmt.Printf("Parse error in %s at line %d, column %d\n",
        parseErr.Format, parseErr.Line, parseErr.Column)
    fmt.Printf("Statement: %s\n", parseErr.Statement)
}
```

### 7. I/O Errors

**Error Code:** `ErrCodeIOError`

**Recovery:**
- Check underlying I/O error for details
- Verify file/network connection is available
- Retry transient I/O errors
- Check disk space for write operations

**Example:**
```go
if code := Code(err); code == ErrCodeIOError {
    // Check underlying error
    if errors.Is(err, io.ErrUnexpectedEOF) {
        // Handle truncated input
    }
}
```

## Best Practices

### 1. Always Check Error Codes

Use the `Code()` function to get structured error codes:

```go
err := Parse(ctx, reader, FormatTurtle, handler)
if err != nil {
    switch Code(err) {
    case ErrCodeLineTooLong:
        // Handle line too long
    case ErrCodeDepthExceeded:
        // Handle depth exceeded
    default:
        // Handle other errors
    }
}
```

### 2. Use Error Wrapping

When returning errors from your code, preserve error context:

```go
if err != nil {
    return fmt.Errorf("failed to process RDF: %w", err)
}
```

### 3. Check for Specific Errors

Use `errors.Is()` and `errors.As()` for error handling:

```go
if errors.Is(err, ErrLineTooLong) {
    // Handle line too long
}

var parseErr *ParseError
if errors.As(err, &parseErr) {
    // Access structured error information
}
```

### 4. Handle Context Cancellation Gracefully

Context cancellation is not an error - handle it appropriately:

```go
err := Parse(ctx, reader, FormatTurtle, handler)
if err != nil && !errors.Is(err, context.Canceled) {
    // Only log/return if not canceled
    return err
}
```

### 5. Use Safe Defaults for Untrusted Input

Always use safe limits for untrusted input:

```go
opts := SafeMode() // Uses conservative limits
dec, err := NewReader(untrustedReader, FormatTurtle, opts...)
```

## Format-Specific Error Handling

### Turtle/TriG

- Undefined prefix errors: Define all prefixes before use
- Invalid IRI errors: Check IRI syntax
- Unterminated string errors: Check string delimiters

### N-Triples

- Line format errors: Each line must be a complete triple
- IRI syntax errors: IRIs must be properly formatted

### JSON-LD

- JSON parse errors: Check JSON syntax first
- Context errors: Verify @context is valid
- Invalid @id errors: Check IRI format

### RDF/XML

- XML parse errors: Validate XML syntax
- Namespace errors: Check namespace declarations
- Invalid element errors: Verify RDF/XML structure

## Error Message Format

Error messages follow this format:

```
format:line:column: error message
  statement excerpt
  ^
```

Example:
```
turtle:5:12: undefined prefix 'ex'
  ex:subject ex:predicate ex:object .
            ^
```

## Testing Error Handling

When testing error handling:

1. Test all error codes are returned correctly
2. Test error wrapping preserves context
3. Test ParseError provides accurate position information
4. Test error recovery strategies work as expected

## Migration Guide

If upgrading from an older version:

1. Replace string error checks with `Code()` function
2. Use `errors.Is()` and `errors.As()` for error matching
3. Check for `ErrCodeContextCanceled` instead of checking context directly
4. Use structured `ParseError` for detailed error information

