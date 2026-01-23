//go:build ignore

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <directory>\n", os.Args[0])
		os.Exit(1)
	}

	dir := os.Args[1]
	
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		data, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		content := string(data)
		original := content

		// Fix missing receiver names - pattern: func ( *TypeName)
		// Replace with: func (d *TypeName) or func (e *TypeName) based on type
		re := regexp.MustCompile(`func\s+\(\s+\*([a-zA-Z][a-zA-Z0-9]*)\)`)
		content = re.ReplaceAllStringFunc(content, func(match string) string {
			// Extract type name
			typeMatch := regexp.MustCompile(`\*([a-zA-Z][a-zA-Z0-9]*)`)
			typeName := typeMatch.FindStringSubmatch(match)[1]
			
			// Determine receiver name based on type
			var receiverName string
			if strings.Contains(typeName, "Decoder") || strings.Contains(typeName, "Reader") {
				receiverName = "d"
			} else if strings.Contains(typeName, "Encoder") || strings.Contains(typeName, "Writer") {
				receiverName = "e"
			} else {
				receiverName = "x" // fallback
			}
			
			return strings.Replace(match, "( *", "("+receiverName+" *", 1)
		})

		if content != original {
			fmt.Printf("Updating %s\n", path)
			return ioutil.WriteFile(path, []byte(content), info.Mode())
		}

		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Done!")
}

