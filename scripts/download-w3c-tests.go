//go:build ignore

package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// testSuite represents a W3C test suite to download
type testSuite struct {
	name        string
	description string
	url         string
	subdir      string // subdirectory name in the downloaded structure
	method      string // "git", "zip", or "tar"
}

var testSuites = []testSuite{
	{
		name:        "rdf-tests",
		description: "W3C RDF Test Suite (Turtle, N-Triples, TriG, N-Quads, RDF/XML)",
		url:         "https://github.com/w3c/rdf-tests/archive/refs/heads/main.zip",
		subdir:      "",
		method:      "zip",
	},
	{
		name:        "rdf-star-tests",
		description: "W3C RDF-star Test Suite",
		url:         "https://github.com/w3c/rdf-star/archive/refs/heads/main.zip",
		subdir:      "",
		method:      "zip",
	},
	{
		name:        "json-ld-tests",
		description: "W3C JSON-LD Test Suite",
		url:         "https://github.com/w3c/json-ld-api/archive/refs/heads/main.zip",
		subdir:      "tests",
		method:      "zip",
	},
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <output-directory>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nDownloads W3C RDF test suites to the specified directory.\n")
		fmt.Fprintf(os.Stderr, "The directory will be organized as:\n")
		fmt.Fprintf(os.Stderr, "  <output-directory>/\n")
		fmt.Fprintf(os.Stderr, "    turtle/\n")
		fmt.Fprintf(os.Stderr, "    ntriples/\n")
		fmt.Fprintf(os.Stderr, "    trig/\n")
		fmt.Fprintf(os.Stderr, "    nquads/\n")
		fmt.Fprintf(os.Stderr, "    rdfxml/\n")
		fmt.Fprintf(os.Stderr, "    jsonld/\n")
		fmt.Fprintf(os.Stderr, "\nExample: %s ./w3c-tests\n", os.Args[0])
		os.Exit(1)
	}

	outputDir := os.Args[1]
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Downloading W3C test suites to: %s\n\n", outputDir)

	for _, suite := range testSuites {
		fmt.Printf("Downloading %s...\n", suite.description)
		if err := downloadTestSuite(suite, outputDir); err != nil {
			fmt.Fprintf(os.Stderr, "Error downloading %s: %v\n", suite.name, err)
			continue
		}
		fmt.Printf("✓ Downloaded %s\n\n", suite.name)
	}

	fmt.Println("Organizing test files...")
	if err := organizeTestFiles(outputDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error organizing test files: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✓ All test suites downloaded and organized!\n")
	fmt.Printf("Set W3C_TESTS_DIR=%s to run conformance tests.\n", outputDir)
}

func downloadTestSuite(suite testSuite, outputDir string) error {
	tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("%s-download.zip", suite.name))
	defer os.Remove(tempFile)

	// Download the file
	fmt.Printf("  Fetching from %s...\n", suite.url)
	resp, err := http.Get(suite.url)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Save to temp file
	out, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("failed to save download: %w", err)
	}
	out.Close()

	// Extract based on method
	fmt.Printf("  Extracting...\n")
	switch suite.method {
	case "zip":
		return extractZip(tempFile, outputDir, suite)
	case "tar", "targz":
		return extractTarGz(tempFile, outputDir, suite)
	default:
		return fmt.Errorf("unsupported method: %s", suite.method)
	}
}

func extractZip(zipFile, outputDir string, suite testSuite) error {
	r, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	defer r.Close()

	// Find the base directory (usually repo-name-main/)
	var baseDir string
	for _, f := range r.File {
		if strings.HasSuffix(f.Name, "/") && baseDir == "" {
			baseDir = f.Name
			break
		}
	}

	// Extract files
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}

		// Skip if not in the subdir (if specified)
		if suite.subdir != "" && !strings.Contains(f.Name, suite.subdir+"/") {
			continue
		}

		// Remove base directory from path
		relPath := strings.TrimPrefix(f.Name, baseDir)
		if suite.subdir != "" {
			// Remove subdir prefix
			idx := strings.Index(relPath, suite.subdir+"/")
			if idx >= 0 {
				relPath = relPath[idx+len(suite.subdir)+1:]
			}
		}

		// Skip if path is empty or just the subdir
		if relPath == "" || relPath == suite.subdir+"/" {
			continue
		}

		destPath := filepath.Join(outputDir, suite.name, relPath)
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		out, err := os.Create(destPath)
		if err != nil {
			rc.Close()
			return err
		}

		_, err = io.Copy(out, rc)
		rc.Close()
		out.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

func extractTarGz(tarFile, outputDir string, suite testSuite) error {
	f, err := os.Open(tarFile)
	if err != nil {
		return err
	}
	defer f.Close()

	var r io.Reader = f
	if strings.HasSuffix(tarFile, ".gz") {
		gzr, err := gzip.NewReader(f)
		if err != nil {
			return err
		}
		defer gzr.Close()
		r = gzr
	}

	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if hdr.FileInfo().IsDir() {
			continue
		}

		// Skip if not in the subdir (if specified)
		if suite.subdir != "" && !strings.Contains(hdr.Name, suite.subdir+"/") {
			continue
		}

		// Remove base directory from path
		parts := strings.Split(hdr.Name, "/")
		if len(parts) > 1 {
			hdr.Name = strings.Join(parts[1:], "/")
		}

		if suite.subdir != "" {
			idx := strings.Index(hdr.Name, suite.subdir+"/")
			if idx >= 0 {
				hdr.Name = hdr.Name[idx+len(suite.subdir)+1:]
			}
		}

		if hdr.Name == "" {
			continue
		}

		destPath := filepath.Join(outputDir, suite.name, hdr.Name)
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		out, err := os.Create(destPath)
		if err != nil {
			return err
		}

		if _, err := io.Copy(out, tr); err != nil {
			out.Close()
			return err
		}
		out.Close()
	}

	return nil
}

// organizeTestFiles reorganizes downloaded test files into the expected directory structure
func organizeTestFiles(outputDir string) error {
	// Map of format names to their test directories in the downloaded repos
	formatDirs := map[string][]string{
		"turtle": {
			"rdf-tests/turtle",
			"rdf-star-tests/turtle",
		},
		"ntriples": {
			"rdf-tests/ntriples",
			"rdf-star-tests/ntriples",
		},
		"trig": {
			"rdf-tests/trig",
			"rdf-star-tests/trig",
		},
		"nquads": {
			"rdf-tests/nquads",
			"rdf-star-tests/nquads",
		},
		"rdfxml": {
			"rdf-tests/rdf-xml",
			"rdf-tests/rdfxml",
		},
		"jsonld": {
			"json-ld-tests",
		},
	}

	for format, sourceDirs := range formatDirs {
		targetDir := filepath.Join(outputDir, format)
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return err
		}

		for _, sourceDir := range sourceDirs {
			sourcePath := filepath.Join(outputDir, sourceDir)
			if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
				continue
			}

			// Copy or move files from source to target
			if err := copyDirectory(sourcePath, targetDir); err != nil {
				fmt.Printf("  Warning: Could not copy %s: %v\n", sourcePath, err)
			}
		}
	}

	return nil
}

func copyDirectory(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		// Copy file
		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return err
		}

		dstFile, err := os.Create(dstPath)
		if err != nil {
			return err
		}
		defer dstFile.Close()

		_, err = io.Copy(dstFile, srcFile)
		return err
	})
}

