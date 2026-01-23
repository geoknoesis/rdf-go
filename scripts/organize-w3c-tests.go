//go:build ignore

package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <w3c-tests-directory>\n", os.Args[0])
		os.Exit(1)
	}

	baseDir := os.Args[1]

	// Map format names to their source directories in rdf-tests
	formatMap := map[string][]string{
		"turtle": {
			filepath.Join(baseDir, "rdf-tests", "rdf", "rdf11", "rdf-turtle"),
			filepath.Join(baseDir, "rdf-tests", "rdf", "rdf12", "rdf-turtle"),
			filepath.Join(baseDir, "rdf-star-tests", "tests", "turtle"),
		},
		"ntriples": {
			filepath.Join(baseDir, "rdf-tests", "rdf", "rdf11", "rdf-n-triples"),
			filepath.Join(baseDir, "rdf-tests", "rdf", "rdf12", "rdf-n-triples"),
			filepath.Join(baseDir, "rdf-star-tests", "tests", "ntriples"),
		},
		"trig": {
			filepath.Join(baseDir, "rdf-tests", "rdf", "rdf11", "rdf-trig"),
			filepath.Join(baseDir, "rdf-tests", "rdf", "rdf12", "rdf-trig"),
			filepath.Join(baseDir, "rdf-star-tests", "tests", "trig"),
		},
		"nquads": {
			filepath.Join(baseDir, "rdf-tests", "rdf", "rdf11", "rdf-n-quads"),
			filepath.Join(baseDir, "rdf-tests", "rdf", "rdf12", "rdf-n-quads"),
			filepath.Join(baseDir, "rdf-star-tests", "tests", "nquads"),
		},
		"rdfxml": {
			filepath.Join(baseDir, "rdf-tests", "rdf", "rdf11", "rdf-xml"),
			filepath.Join(baseDir, "rdf-tests", "rdf", "rdf12", "rdf-xml"),
		},
		"jsonld": {
			filepath.Join(baseDir, "json-ld-tests", "tests"),
		},
	}

	for format, sourceDirs := range formatMap {
		targetDir := filepath.Join(baseDir, format)
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			fmt.Printf("Error creating %s: %v\n", targetDir, err)
			continue
		}

		for _, sourceDir := range sourceDirs {
			if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
				continue
			}

			fmt.Printf("Copying %s tests from %s...\n", format, filepath.Base(sourceDir))
			if err := copyDirectory(sourceDir, targetDir); err != nil {
				fmt.Printf("  Warning: Could not copy %s: %v\n", sourceDir, err)
			}
		}
	}

	fmt.Println("\n✓ Test files organized!")
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

		// Skip manifest files and other non-test files if desired
		// For now, copy everything

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

