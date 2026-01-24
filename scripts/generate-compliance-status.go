//go:build ignore

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

type formatResult struct {
	Format   string
	Pass     int
	Fail     int
	Skip     int
	Total    int
	PassRate float64
	Status   string // "pass", "fail", "partial"
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <w3c-tests-dir> [output-file]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  If output-file is not specified, writes to COMPLIANCE_STATUS.md\n")
		os.Exit(1)
	}

	w3cDir := os.Args[1]
	outputFile := "COMPLIANCE_STATUS.md"
	if len(os.Args) >= 3 {
		outputFile = os.Args[2]
	}

	// Set environment variable for tests
	os.Setenv("W3C_TESTS_DIR", w3cDir)

	formats := []string{"turtle", "ntriples", "trig", "nquads", "rdfxml", "jsonld"}

	fmt.Printf("Running compliance tests for %d formats...\n", len(formats))
	fmt.Println("This may take several minutes...")
	fmt.Println()

	results := make([]formatResult, 0, len(formats))
	totalPass := 0
	totalFail := 0
	totalSkip := 0

	for _, format := range formats {
		fmt.Printf("Testing %s... ", format)
		result := runFormatTests(format)
		results = append(results, result)
		totalPass += result.Pass
		totalFail += result.Fail
		totalSkip += result.Skip

		statusIcon := "✅"
		if result.Status == "fail" {
			statusIcon = "❌"
		} else if result.Status == "partial" {
			statusIcon = "⚠️"
		}
		fmt.Printf("%s Pass: %d, Fail: %d, Skip: %d, Total: %d (%.1f%%)\n",
			statusIcon, result.Pass, result.Fail, result.Skip, result.Total, result.PassRate)
	}

	fmt.Println()
	fmt.Printf("Total: Pass: %d, Fail: %d, Skip: %d, Total: %d\n",
		totalPass, totalFail, totalSkip, totalPass+totalFail+totalSkip)

	// Generate markdown
	markdown := generateMarkdown(results, totalPass, totalFail, totalSkip, w3cDir)

	// Write to file
	if err := os.WriteFile(outputFile, []byte(markdown), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing to %s: %v\n", outputFile, err)
		os.Exit(1)
	}

	fmt.Printf("\nStatus page written to: %s\n", outputFile)
}

func runFormatTests(format string) formatResult {
	result := formatResult{
		Format: format,
		Status: "pass",
	}

	// Run the test for this format
	cmd := exec.Command("go", "test", "./rdf", "-run", fmt.Sprintf("^TestW3CConformance$/%s$", format), "-v")
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		// Test command failed, but we still want to parse the output
		// to see how many tests passed/failed
	}

	// Count test results
	// Look for patterns like:
	// --- PASS: TestW3CConformance/turtle/test_name (0.01s)
	// --- FAIL: TestW3CConformance/turtle/test_name (0.01s)
	// --- SKIP: TestW3CConformance/turtle/test_name (0.01s)
	lines := strings.Split(outputStr, "\n")
	for _, line := range lines {
		if strings.Contains(line, "--- PASS:") && strings.Contains(line, format) {
			result.Pass++
		} else if strings.Contains(line, "--- FAIL:") && strings.Contains(line, format) {
			result.Fail++
		} else if strings.Contains(line, "--- SKIP:") && strings.Contains(line, format) {
			result.Skip++
		}
	}

	// Also count from summary lines like:
	// PASS
	// ok      github.com/geoknoesis/rdf-go/rdf        1.234s
	// or
	// FAIL
	if strings.Contains(outputStr, "FAIL") && result.Fail == 0 {
		// If we see FAIL but no individual failures counted, there might be a setup issue
		if !strings.Contains(outputStr, "ok") {
			result.Fail = 1
		}
	}

	result.Total = result.Pass + result.Fail + result.Skip

	// Calculate pass rate (excluding skipped tests)
	if result.Total > 0 {
		nonSkipped := result.Pass + result.Fail
		if nonSkipped > 0 {
			result.PassRate = 100.0 * float64(result.Pass) / float64(nonSkipped)
		} else {
			result.PassRate = 0.0
		}
	}

	// Determine status
	if result.Total == 0 {
		result.Status = "fail" // No tests found
	} else if result.Fail > 0 {
		if result.Pass == 0 {
			result.Status = "fail"
		} else {
			result.Status = "partial"
		}
	} else if result.Pass > 0 {
		result.Status = "pass"
	}

	return result
}

func generateMarkdown(results []formatResult, totalPass, totalFail, totalSkip int, w3cDir string) string {
	now := time.Now().UTC()
	timestamp := now.Format("2006-01-02 15:04:05 UTC")

	var sb strings.Builder

	sb.WriteString("# W3C Compliance Test Status\n\n")
	sb.WriteString("This page shows the current status of W3C compliance tests for each RDF format.\n\n")
	sb.WriteString(fmt.Sprintf("**Last Updated:** %s\n\n", timestamp))
	sb.WriteString(fmt.Sprintf("**W3C Tests Directory:** `%s`\n\n", w3cDir))
	sb.WriteString("---\n\n")

	// Summary table
	sb.WriteString("## Summary\n\n")
	sb.WriteString("| Format | Status | Pass | Fail | Skip | Total | Pass Rate |\n")
	sb.WriteString("|--------|--------|------|------|------|-------|-----------|\n")

	totalTests := totalPass + totalFail + totalSkip
	var totalPassRate float64
	if totalTests > 0 {
		nonSkipped := totalPass + totalFail
		if nonSkipped > 0 {
			totalPassRate = 100.0 * float64(totalPass) / float64(nonSkipped)
		}
	}

	for _, r := range results {
		statusIcon := "✅"
		statusText := "PASS"
		if r.Status == "fail" {
			statusIcon = "❌"
			statusText = "FAIL"
		} else if r.Status == "partial" {
			statusIcon = "⚠️"
			statusText = "PARTIAL"
		}

		sb.WriteString(fmt.Sprintf("| %s | %s %s | %d | %d | %d | %d | %.1f%% |\n",
			strings.ToUpper(r.Format), statusIcon, statusText, r.Pass, r.Fail, r.Skip, r.Total, r.PassRate))
	}

	sb.WriteString(fmt.Sprintf("| **TOTAL** | | **%d** | **%d** | **%d** | **%d** | **%.1f%%** |\n\n",
		totalPass, totalFail, totalSkip, totalTests, totalPassRate))

	// Detailed results for each format
	sb.WriteString("## Detailed Results\n\n")

	for _, r := range results {
		sb.WriteString(fmt.Sprintf("### %s\n\n", strings.ToUpper(r.Format)))

		statusIcon := "✅"
		statusText := "All tests passing"
		if r.Status == "fail" {
			statusIcon = "❌"
			statusText = "Tests failing"
		} else if r.Status == "partial" {
			statusIcon = "⚠️"
			statusText = "Some tests failing"
		}

		sb.WriteString(fmt.Sprintf("**Status:** %s %s\n\n", statusIcon, statusText))
		sb.WriteString(fmt.Sprintf("- **Pass:** %d\n", r.Pass))
		sb.WriteString(fmt.Sprintf("- **Fail:** %d\n", r.Fail))
		if r.Skip > 0 {
			sb.WriteString(fmt.Sprintf("- **Skip:** %d\n", r.Skip))
		}
		sb.WriteString(fmt.Sprintf("- **Total:** %d\n", r.Total))
		sb.WriteString(fmt.Sprintf("- **Pass Rate:** %.1f%%\n\n", r.PassRate))

		if r.Total == 0 {
			sb.WriteString("⚠️ No tests found for this format.\n\n")
		}

		sb.WriteString("---\n\n")
	}

	// Footer
	sb.WriteString("## Notes\n\n")
	sb.WriteString("- This status page is automatically generated by running compliance tests.\n")
	sb.WriteString("- To update this page, run: `go run scripts/generate-compliance-status.go <w3c-tests-dir>`\n")
	sb.WriteString("- Tests are run from the W3C RDF Test Suite.\n")
	sb.WriteString("- Skipped tests are not included in pass rate calculations.\n\n")

	return sb.String()
}



