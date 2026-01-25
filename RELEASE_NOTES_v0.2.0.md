# rdf-go v0.2.0

Release v0.2.0 focuses on standards compliance improvements and enhanced testing infrastructure.

## 🐛 Bug Fixes

### Turtle Numeric Literal Parsing
- **Fixed**: Bare integers (e.g., `30`) now correctly parse as `xsd:integer` instead of `xsd:decimal`
- **Fixed**: Numeric literals now correctly distinguish between:
  - **Integers** (e.g., `30`, `-30`, `+30`) → `xsd:integer`
  - **Decimals** (e.g., `30.0`, `30.5`, `.5`) → `xsd:decimal`
  - **Doubles** (e.g., `30e10`, `3.0E-1`) → `xsd:double`
- **Compliance**: Now fully conforms to RDF 1.1 Turtle specification Section 2.5.2

## ✨ Enhancements

### Enhanced Conformance Testing
- **Added**: Strict datatype validation for numeric literals in W3C conformance tests
- **Added**: `validateTurtleNumericLiteralDatatypes()` function to catch datatype mismatches
- **Benefit**: Ensures compliance with RDF 1.1 Turtle spec even when W3C test suites don't enforce exact datatype matching
- **Coverage**: Comprehensive unit tests added for all numeric literal forms

### Testing Infrastructure
- **Added**: `.gitignore` to exclude build artifacts, test binaries, and temporary files
- **Added**: `create-release.ps1` script for automated GitHub release creation
- **Improved**: Cleaner repository structure with proper ignore patterns

## 📦 Installation

```bash
go get github.com/geoknoesis/rdf-go@v0.2.0
```

## 📚 Documentation

- **Full documentation**: [https://geoknoesis.github.io/rdf-go/](https://geoknoesis.github.io/rdf-go/)
- **Error handling guide**: [ERROR_HANDLING.md](rdf/ERROR_HANDLING.md)
- **Release notes**: [RELEASE_NOTES_v0.1.0.md](RELEASE_NOTES_v0.1.0.md)

## 🔗 Links

- **Repository**: https://github.com/geoknoesis/rdf-go
- **Package**: https://pkg.go.dev/github.com/geoknoesis/rdf-go@v0.2.0
- **Issues**: https://github.com/geoknoesis/rdf-go/issues

## 🙏 Support

If you find rdf-go valuable, please consider:
- ⭐ Starring the repository
- 💰 [GitHub Sponsors](https://github.com/sponsors/geoknoesis)
- ☕ [Ko-fi](https://ko-fi.com/fellahst)

---

**Developed by [GeoKnoesis LLC](https://geoknoesis.com/)**

