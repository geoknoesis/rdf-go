#!/bin/bash
# Download W3C RDF test suites
# Usage: ./download-w3c-tests.sh [output-directory]

set -e

OUTPUT_DIR="${1:-./w3c-tests}"

echo "Downloading W3C test suites to: $OUTPUT_DIR"
echo ""

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Function to download and extract a GitHub repo
download_repo() {
    local repo=$1
    local name=$2
    local subdir=$3
    
    echo "Downloading $name..."
    cd "$OUTPUT_DIR"
    
    # Clone the repository
    if [ -d "$name" ]; then
        echo "  $name already exists, updating..."
        cd "$name"
        git pull || true
        cd ..
    else
        git clone "https://github.com/w3c/$repo.git" "$name" || {
            echo "  Error: git clone failed. Trying zip download..."
            curl -L "https://github.com/w3c/$repo/archive/refs/heads/main.zip" -o "${name}.zip"
            unzip -q "${name}.zip" || true
            mv "${repo}-main" "$name" || true
            rm -f "${name}.zip"
        }
    fi
    
    cd - > /dev/null
    echo "✓ Downloaded $name"
    echo ""
}

# Download test suites
download_repo "rdf-tests" "rdf-tests" ""
download_repo "rdf-star" "rdf-star-tests" ""
download_repo "json-ld-api" "json-ld-tests" "tests"

# Organize test files
echo "Organizing test files..."
mkdir -p "$OUTPUT_DIR/turtle"
mkdir -p "$OUTPUT_DIR/ntriples"
mkdir -p "$OUTPUT_DIR/trig"
mkdir -p "$OUTPUT_DIR/nquads"
mkdir -p "$OUTPUT_DIR/rdfxml"
mkdir -p "$OUTPUT_DIR/jsonld"

# Copy files from rdf-tests
if [ -d "$OUTPUT_DIR/rdf-tests/turtle" ]; then
    cp -r "$OUTPUT_DIR/rdf-tests/turtle"/* "$OUTPUT_DIR/turtle/" 2>/dev/null || true
fi
if [ -d "$OUTPUT_DIR/rdf-tests/ntriples" ]; then
    cp -r "$OUTPUT_DIR/rdf-tests/ntriples"/* "$OUTPUT_DIR/ntriples/" 2>/dev/null || true
fi
if [ -d "$OUTPUT_DIR/rdf-tests/trig" ]; then
    cp -r "$OUTPUT_DIR/rdf-tests/trig"/* "$OUTPUT_DIR/trig/" 2>/dev/null || true
fi
if [ -d "$OUTPUT_DIR/rdf-tests/nquads" ]; then
    cp -r "$OUTPUT_DIR/rdf-tests/nquads"/* "$OUTPUT_DIR/nquads/" 2>/dev/null || true
fi
if [ -d "$OUTPUT_DIR/rdf-tests/rdf-xml" ]; then
    cp -r "$OUTPUT_DIR/rdf-tests/rdf-xml"/* "$OUTPUT_DIR/rdfxml/" 2>/dev/null || true
fi
if [ -d "$OUTPUT_DIR/rdf-tests/rdfxml" ]; then
    cp -r "$OUTPUT_DIR/rdf-tests/rdfxml"/* "$OUTPUT_DIR/rdfxml/" 2>/dev/null || true
fi

# Copy files from rdf-star-tests
if [ -d "$OUTPUT_DIR/rdf-star-tests/turtle" ]; then
    cp -r "$OUTPUT_DIR/rdf-star-tests/turtle"/* "$OUTPUT_DIR/turtle/" 2>/dev/null || true
fi
if [ -d "$OUTPUT_DIR/rdf-star-tests/ntriples" ]; then
    cp -r "$OUTPUT_DIR/rdf-star-tests/ntriples"/* "$OUTPUT_DIR/ntriples/" 2>/dev/null || true
fi
if [ -d "$OUTPUT_DIR/rdf-star-tests/trig" ]; then
    cp -r "$OUTPUT_DIR/rdf-star-tests/trig"/* "$OUTPUT_DIR/trig/" 2>/dev/null || true
fi
if [ -d "$OUTPUT_DIR/rdf-star-tests/nquads" ]; then
    cp -r "$OUTPUT_DIR/rdf-star-tests/nquads"/* "$OUTPUT_DIR/nquads/" 2>/dev/null || true
fi

# Copy JSON-LD tests
if [ -d "$OUTPUT_DIR/json-ld-tests/tests" ]; then
    cp -r "$OUTPUT_DIR/json-ld-tests/tests"/* "$OUTPUT_DIR/jsonld/" 2>/dev/null || true
fi

echo ""
echo "✓ All test suites downloaded and organized!"
echo "Set W3C_TESTS_DIR=$OUTPUT_DIR to run conformance tests."

