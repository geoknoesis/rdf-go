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
	
	// Map of incorrect receiver names to correct ones
	fixes := map[string]string{
		"*jsonldTripleReader": "*jsonldTripleDecoder",
		"*jsonldQuadReader":   "*jsonldQuadDecoder",
		"*trigQuadReader":    "*trigQuadDecoder",
		"*rdfxmlTripleReader": "*rdfxmlTripleDecoder",
		"*ntTripleReader":    "*ntTripleDecoder",
		"*ntQuadReader":      "*ntQuadDecoder",
		"*turtleTripleReader": "*turtleTripleDecoder",
		"*ntTripleWriter":    "*ntTripleEncoder",
		"*ntQuadWriter":      "*ntQuadEncoder",
		"*turtleTripleWriter": "*turtleTripleEncoder",
		"*trigQuadWriter":     "*trigQuadEncoder",
		"*rdfxmlTripleWriter": "*rdfxmlTripleEncoder",
		"*jsonldTripleWriter": "*jsonldTripleEncoder",
		"*jsonldQuadWriter":   "*jsonldQuadEncoder",
	}

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

		// Fix method receivers
		for wrong, correct := range fixes {
			// Match function declarations with receivers
			pattern := regexp.MustCompile(`func\s+\([^)]*\s+` + regexp.QuoteMeta(wrong) + `\)`)
			content = pattern.ReplaceAllString(content, strings.Replace(`func ($RECEIVER `+wrong+`)`, wrong, correct, 1))
			
			// More direct replacement
			content = strings.ReplaceAll(content, "("+wrong+")", "("+correct+")")
		}

		// Fix test file type assertion
		content = strings.ReplaceAll(content, "(*rdfxmlTripleReader)", "(*rdfxmlTripleDecoder)")

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



