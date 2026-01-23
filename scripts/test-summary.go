//go:build ignore

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <w3c-tests-dir>\n", os.Args[0])
		os.Exit(1)
	}

	w3cDir := os.Args[1]
	os.Setenv("W3C_TESTS_DIR", w3cDir)

	formats := []string{"turtle", "ntriples", "trig", "nquads", "rdfxml", "jsonld"}

	fmt.Println("W3C Conformance Test Summary")
	fmt.Println("=" + strings.Repeat("=", 60))
	fmt.Println()

	totalPass := 0
	totalFail := 0

	for _, format := range formats {
		// Use -v flag to get verbose output that shows all test results
		cmd := exec.Command("go", "test", "./rdf", "-run", fmt.Sprintf("TestW3CConformance/%s", format), "-v")
		output, _ := cmd.CombinedOutput()
		outputStr := string(output)
		passCount := strings.Count(outputStr, "--- PASS:")
		failCount := strings.Count(outputStr, "--- FAIL:")

		totalPass += passCount
		totalFail += failCount

		total := passCount + failCount
		var passRate float64
		if total > 0 {
			passRate = 100.0 * float64(passCount) / float64(total)
		} else {
			passRate = 0.0
		}

		fmt.Printf("%-10s: PASS=%4d  FAIL=%4d  Total=%4d  Pass Rate=%.1f%%\n",
			format, passCount, failCount, total, passRate)
	}

	fmt.Println()
	fmt.Println("=" + strings.Repeat("=", 60))
	fmt.Printf("TOTAL:      PASS=%4d  FAIL=%4d  Total=%4d  Pass Rate=%.1f%%\n",
		totalPass, totalFail, totalPass+totalFail,
		100.0*float64(totalPass)/float64(totalPass+totalFail))
}
