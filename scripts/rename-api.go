//go:build ignore

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
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

		// Replace function names
		content = strings.ReplaceAll(content, "NewDecoder", "NewReader")
		content = strings.ReplaceAll(content, "NewEncoder", "NewWriter")

		// Replace type names (but be careful with variable names)
		// Replace "Decoder" interface/type references
		content = strings.ReplaceAll(content, " Decoder ", " Reader ")
		content = strings.ReplaceAll(content, " Decoder,", " Reader,")
		content = strings.ReplaceAll(content, " Decoder)", " Reader)")
		content = strings.ReplaceAll(content, "(Decoder ", "(Reader ")
		content = strings.ReplaceAll(content, " Decoder{", " Reader{")
		content = strings.ReplaceAll(content, " Decoder\n", " Reader\n")
		content = strings.ReplaceAll(content, " Decoder\n", " Reader\n")
		content = strings.ReplaceAll(content, "type Decoder", "type Reader")
		content = strings.ReplaceAll(content, "Decoder interface", "Reader interface")
		content = strings.ReplaceAll(content, "Decoder,", "Reader,")
		content = strings.ReplaceAll(content, "Decoder)", "Reader)")

		// Replace "Encoder" interface/type references
		content = strings.ReplaceAll(content, " Encoder ", " Writer ")
		content = strings.ReplaceAll(content, " Encoder,", " Writer,")
		content = strings.ReplaceAll(content, " Encoder)", " Writer)")
		content = strings.ReplaceAll(content, "(Encoder ", "(Writer ")
		content = strings.ReplaceAll(content, " Encoder{", " Writer{")
		content = strings.ReplaceAll(content, "type Encoder", "type Writer")
		content = strings.ReplaceAll(content, "Encoder interface", "Writer interface")
		content = strings.ReplaceAll(content, "Encoder,", "Writer,")
		content = strings.ReplaceAll(content, "Encoder)", "Writer)")

		// Replace adapter types
		content = strings.ReplaceAll(content, "quadDecoderAdapter", "quadReaderAdapter")
		content = strings.ReplaceAll(content, "quadEncoderAdapter", "quadWriterAdapter")

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

