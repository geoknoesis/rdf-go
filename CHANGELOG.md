# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- `OptStrictIRIValidation()` option to enable strict IRI validation according to RFC 3987
- `ValidateIRI()` helper function for programmatic IRI validation
- `CanonicalizeJSONLD()`, `CanonicalizeJSONLDReader()`, and `CanonicalizeJSONLDWriter()` functions for deterministic JSON-LD output
- `OptExpandRDFXMLContainers()` and `OptDisableRDFXMLContainerExpansion()` options for RDF/XML container membership expansion control
- RDF/XML container membership expansion (enabled by default) - automatically converts `rdf:li` to `rdf:_1`, `rdf:_2`, etc.
- Determinism tests for Turtle and TriG prefix ordering
- Round-trip byte-for-byte determinism tests for all deterministic formats
- Large-scale benchmarks (1MB and 10MB) for performance testing
- `CHANGELOG.md` to track version history

### Changed
- Go version requirement updated to 1.25.5
- RDF/XML container expansion is now implemented and enabled by default

### Fixed
- Go version requirement in `go.mod` (was incorrectly set to 1.24.0)

### Enhanced
- IRI validation integrated into Turtle parser when `OptStrictIRIValidation()` is enabled
- RDF/XML container membership expansion integrated into RDF/XML parser
- Documentation updated with IRI validation examples, JSON-LD canonicalization usage, and RDF/XML container expansion

## [1.0.0] - TBD

### Added
- Initial release
- Unified `Reader` and `Writer` interfaces for all RDF formats
- Support for Turtle, TriG, N-Triples, N-Quads, RDF/XML, JSON-LD
- RDF-star support via `TripleTerm`
- Automatic format detection with `FormatAuto`
- Streaming architecture for efficient processing
- Security limits with `OptSafeLimits()` for untrusted input
- Comprehensive error reporting with `ParseError` (line/column/excerpt)
- Programmatic error codes via `Code()` function
- Context cancellation support
- W3C test suite integration
- Fuzzing for all parsers
- Round-trip tests for semantic equivalence

### Features
- **Streaming-first design**: All parsers stream without buffering entire documents
- **Unified API**: Single `Reader`/`Writer` interface works with all formats
- **Security-conscious**: Configurable limits prevent resource exhaustion
- **Excellent error diagnostics**: Parse errors include position, excerpts, and caret indicators
- **Go-idiomatic**: Uses `io.Reader`/`io.Writer`, functional options, clear naming

### Known Limitations
- JSON-LD output is non-deterministic due to Go map iteration order
- JSON-LD may buffer when `@graph` appears before `@context`
- RDF/XML container membership expansion not implemented
- JSON-LD compaction and framing not supported (I/O library only)

---

## Version History Notes

### Semantic Versioning Policy
- **v1.x.x**: Backward compatible changes only (new features, bug fixes)
- **v2.x.x**: Breaking changes (if needed in the future)

### API Stability
The following APIs are considered stable and will maintain backward compatibility:
- `Reader` and `Writer` interfaces
- `Statement`, `Triple`, `Quad` types
- `Term` interface and implementations (`IRI`, `BlankNode`, `Literal`, `TripleTerm`)
- `NewReader()`, `NewWriter()`, `Parse()` functions
- Format constants (`FormatTurtle`, `FormatNTriples`, etc.)
- Option functions (`OptMaxDepth`, `OptSafeLimits`, etc.)

### Deprecation Policy
- Deprecated APIs will be marked with `// Deprecated:` comments
- Deprecated APIs will be removed in the next major version
- At least one minor version will include deprecation warnings before removal

