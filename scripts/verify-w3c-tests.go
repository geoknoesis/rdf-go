//go:build ignore

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <w3c-tests-directory>\n", os.Args[0])
		os.Exit(1)
	}

	baseDir := os.Args[1]

	formatConfigs := map[string][]string{
		"turtle":    {".ttl"},
		"ntriples":  {".nt"},
		"trig":      {".trig"},
		"nquads":    {".nq"},
		"rdfxml":    {".rdf", ".xml"},
		"jsonld":    {".jsonld", ".json"},
	}

	fmt.Println("W3C Test Suite Verification")
	fmt.Println("=" + strings.Repeat("=", 50))
	fmt.Println()

	totalFiles := 0
	allOK := true

	for format, exts := range formatConfigs {
		formatDir := filepath.Join(baseDir, format)
		
		fmt.Printf("Format: %s\n", format)
		fmt.Printf("  Directory: %s\n", formatDir)
		
		// Check if directory exists
		if _, err := os.Stat(formatDir); os.IsNotExist(err) {
			fmt.Printf("  ❌ Directory does not exist!\n\n")
			allOK = false
			continue
		}

		// Count files with expected extensions
		fileCount := 0
		topLevelCount := 0
		
		err := filepath.Walk(formatDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}

			// Check if file has expected extension
			for _, ext := range exts {
				if strings.HasSuffix(strings.ToLower(path), ext) {
					fileCount++
					
					// Check if it's top-level
					relPath, _ := filepath.Rel(formatDir, path)
					if !strings.Contains(relPath, string(filepath.Separator)) {
						topLevelCount++
					}
					break
				}
			}
			return nil
		})

		if err != nil {
			fmt.Printf("  ❌ Error scanning directory: %v\n\n", err)
			allOK = false
			continue
		}

		totalFiles += fileCount
		
		if fileCount == 0 {
			fmt.Printf("  ⚠️  No test files found with extensions: %v\n", exts)
			allOK = false
		} else {
			fmt.Printf("  ✓ Found %d test files", fileCount)
			if topLevelCount > 0 {
				fmt.Printf(" (%d at top level, %d in subdirectories)", topLevelCount, fileCount-topLevelCount)
			}
			fmt.Println()
		}
		fmt.Println()
	}

	fmt.Println("=" + strings.Repeat("=", 50))
	fmt.Printf("Total test files: %d\n", totalFiles)
	
	if allOK {
		fmt.Println("✓ All test directories are properly organized!")
	} else {
		fmt.Println("⚠️  Some issues detected. Check the output above.")
		os.Exit(1)
	}
}

