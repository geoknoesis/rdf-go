# rdf-go v0.1.0

Initial release of `rdf-go`, a comprehensive, standards-compliant RDF parsing/encoding library for Go.

## 🎉 Features

### Standards Compliance
- ✅ **RDF 1.1 & 1.2** - Full conformance with W3C RDF specifications
- ✅ **JSON-LD 1.1** - Complete support for JSON-LD 1.1 Processing Algorithms
- ✅ **Turtle 1.1 & 1.2** - Full syntax support including RDF-star
- ✅ **RDF/XML 1.0** - Complete support with container membership expansion
- ✅ **W3C Test Suite Compliance** - Validated against official W3C test suites

### Format Support
- **Triple formats**: Turtle, N-Triples, RDF/XML, JSON-LD
- **Quad formats**: TriG, N-Quads, JSON-LD (with named graphs)
- **All formats**: Parsing ✅, Encoding ✅, Blank Nodes ✅
- **RDF-star**: Turtle ✅, TriG ✅, N-Triples ✅, N-Quads ✅

### Architecture
- **Unified API** - Single `Reader` and `Writer` interface for all formats
- **Streaming-first** - O(1) memory usage with true streaming parsers
- **Automatic format detection** - No need to specify format explicitly
- **RDF-star support** - Native quoted triples via `TripleTerm` type

### Performance
- Low-allocation design optimized for high-throughput scenarios
- Typically processes 10K-100K+ triples/second
- Comprehensive benchmark suite included
- Performance regression tests ensure consistent performance

### Developer Experience
- Simple, intuitive API with optional context parameter
- Convenient `Parse()` function for common use cases
- Comprehensive error handling with structured error codes
- Extensive documentation with examples

### Security
- Built-in security limits for untrusted input
- `OptSafeLimits()` for conservative defaults
- Structured error codes for programmatic error handling

## 📦 Installation

```bash
go get github.com/geoknoesis/rdf-go@v0.1.0
```

## 📚 Documentation

- **Full documentation**: [https://geoknoesis.github.io/rdf-go/](https://geoknoesis.github.io/rdf-go/)
- Getting Started guide, Concepts, How-to guides, and complete API reference
- Error handling guide: [ERROR_HANDLING.md](rdf/ERROR_HANDLING.md)
- Comprehensive examples in code and documentation

## 🔗 Links

- **Repository**: https://github.com/geoknoesis/rdf-go
- **Package**: https://pkg.go.dev/github.com/geoknoesis/rdf-go@v0.1.0
- **Issues**: https://github.com/geoknoesis/rdf-go/issues

## 🙏 Support

If you find rdf-go valuable, please consider:
- ⭐ Starring the repository
- 💰 [GitHub Sponsors](https://github.com/sponsors/geoknoesis)
- ☕ [Ko-fi](https://ko-fi.com/fellahst)

---

**Developed by [GeoKnoesis LLC](https://geoknoesis.com/)**

