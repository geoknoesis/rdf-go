# W3C Test Suite Download Scripts

This directory contains scripts to download W3C RDF test suites for conformance testing.

## Available Scripts

### Go Script (Cross-platform)
```bash
go run scripts/download-w3c-tests.go ./w3c-tests
```

### Bash Script (Unix/Linux/macOS)
```bash
chmod +x scripts/download-w3c-tests.sh
./scripts/download-w3c-tests.sh ./w3c-tests
```

### PowerShell Script (Windows)
```powershell
.\scripts\download-w3c-tests.ps1 .\w3c-tests
```

## What Gets Downloaded

The scripts download the following W3C test suites:

1. **rdf-tests** - Main W3C RDF test suite containing:
   - Turtle tests
   - N-Triples tests
   - TriG tests
   - N-Quads tests
   - RDF/XML tests

2. **rdf-star-tests** - RDF-star extension tests

3. **json-ld-api** - JSON-LD test suite

## Directory Structure

After downloading, the tests are organized as:

```
w3c-tests/
  ├── turtle/        # Turtle test files
  ├── ntriples/      # N-Triples test files
  ├── trig/          # TriG test files
  ├── nquads/        # N-Quads test files
  ├── rdfxml/        # RDF/XML test files
  └── jsonld/         # JSON-LD test files
```

## Usage

1. Download the test suites:
   ```bash
   go run scripts/download-w3c-tests.go ./w3c-tests
   ```

2. Set the environment variable:
   ```bash
   export W3C_TESTS_DIR=./w3c-tests
   # or on Windows:
   set W3C_TESTS_DIR=.\w3c-tests
   ```

3. Run conformance tests:
   ```bash
   go test ./rdf -run TestW3CConformance -v
   ```

## Requirements

- **Go script**: Go 1.16+ (uses standard library only)
- **Bash script**: `git`, `curl`, `unzip` (or `git` only if cloning)
- **PowerShell script**: PowerShell 5.1+ (or `git` if available)

## Notes

- The scripts will attempt to use `git clone` if available (faster and allows updates)
- If `git` is not available, they fall back to downloading ZIP archives
- Existing directories will be updated (git pull) rather than re-downloaded
- Test files are organized into format-specific directories for easy use with the conformance tests

